package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Input size limits
const (
	maxSortElements  = 10_000_000
	maxMatrixDim     = 1000
	maxTextBytes     = 10 * 1024 * 1024 // 10MB
)

// CreateJob validates input and creates a pending Job. Returns immediately.
func (s *Store) CreateJob(req JobSubmitRequest) (*Job, error) {
	if req.TypeID == "" {
		return nil, fmt.Errorf("missing typeId")
	}
	if req.MaxRetry <= 0 {
		req.MaxRetry = 2
	}
	if req.TimeoutSec <= 0 {
		req.TimeoutSec = 60
	}

	s.mu.RLock()
	_, typeOK := s.taskTypes[req.TypeID]
	s.mu.RUnlock()
	if !typeOK {
		return nil, fmt.Errorf("unknown task type: %s", req.TypeID)
	}

	// Input validation
	if err := validateInput(req.TypeID, req.Input); err != nil {
		return nil, fmt.Errorf("input validation: %w", err)
	}

	job := &Job{
		JobID:      newID("job"),
		TypeID:     req.TypeID,
		Status:     "pending",
		Input:      req.Input,
		CreatedAt:  time.Now().UnixMilli(),
		MaxRetry:   req.MaxRetry,
		TimeoutSec: req.TimeoutSec,
	}

	s.mu.Lock()
	s.jobs[job.JobID] = job
	s.mu.Unlock()

	logInfo(fmt.Sprintf("job created: %s type=%s", job.JobID, job.TypeID))
	return job, nil
}

func validateInput(typeID string, input map[string]interface{}) error {
	switch typeID {
	case "sort":
		if data, ok := input["data"].([]interface{}); ok && len(data) > maxSortElements {
			return fmt.Errorf("sort: array too large (%d elements, max %d)", len(data), maxSortElements)
		}
	case "matrix-mul":
		if a, ok := input["a"].([]interface{}); ok && len(a) > maxMatrixDim {
			return fmt.Errorf("matrix: too many rows (%d, max %d)", len(a), maxMatrixDim)
		}
	case "word-count", "grep":
		if text, ok := input["text"].(string); ok && len(text) > maxTextBytes {
			return fmt.Errorf("text too large (%d bytes, max %d)", len(text), maxTextBytes)
		}
	}
	return nil
}

// ExecuteJob runs the full split → dispatch → aggregate pipeline.
func (s *Store) ExecuteJob(parentCtx context.Context, job *Job) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(job.TimeoutSec)*time.Second)
	job.cancel = cancel
	defer cancel()

	// Get healthy workers
	workers := s.getHealthyWorkers(ctx)
	if len(workers) == 0 {
		s.failJob(job, "no available workers")
		return
	}

	// Split
	job.Status = "splitting"
	splitFn, ok := splitters[job.TypeID]
	if !ok {
		s.failJob(job, "no splitter for type: "+job.TypeID)
		return
	}

	subInputs, err := splitFn(job.Input, len(workers))
	if err != nil {
		s.failJob(job, "split error: "+err.Error())
		return
	}

	// Create tasks
	tasks := make([]*Task, len(subInputs))
	for i, input := range subInputs {
		tasks[i] = &Task{
			TaskID: newID("task"),
			JobID:  job.JobID,
			TypeID: job.TypeID,
			Status: "pending",
			Input:  input,
		}
	}

	s.mu.Lock()
	job.Tasks = tasks
	job.Status = "running"
	s.mu.Unlock()

	// Dispatch in parallel with goroutine-safe timeout
	var wg sync.WaitGroup
	done := make(chan struct{})

	for i, task := range tasks {
		workerIdx := i % len(workers)
		worker := workers[workerIdx]

		s.mu.Lock()
		task.WorkerID = worker.instanceID
		task.WorkerAddr = worker.addr
		task.Status = "running"
		task.StartedAt = time.Now().UnixMilli()
		s.mu.Unlock()

		wg.Add(1)
		go func(t *Task, w workerInfo) {
			defer wg.Done()
			executeTaskWithRetry(ctx, t, workers, job.MaxRetry)
		}(task, worker)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All tasks completed
	case <-ctx.Done():
		// Timeout or cancellation
		s.mu.Lock()
		if job.Status == "running" {
			job.Status = "failed"
			job.Error = "timeout or cancelled"
			job.FinishedAt = time.Now().UnixMilli()
		}
		s.mu.Unlock()
		_ = s.saveToDiskPublic()
		return
	}

	// Check if cancelled during execution
	s.mu.RLock()
	if job.Status == "cancelled" {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	// Collect results
	allDone := true
	var failedTasks []string
	var outputs []interface{}

	for _, t := range tasks {
		if t.Status == "done" {
			outputs = append(outputs, t.Output)
		} else {
			allDone = false
			failedTasks = append(failedTasks, fmt.Sprintf("%s: %s", t.TaskID, t.Error))
		}
	}

	if !allDone {
		s.failJob(job, fmt.Sprintf("tasks failed: %v", failedTasks))
		return
	}

	// Aggregate
	s.mu.Lock()
	job.Status = "aggregating"
	s.mu.Unlock()

	aggFn, ok := aggregators[job.TypeID]
	if !ok {
		s.failJob(job, "no aggregator for type: "+job.TypeID)
		return
	}

	result, err := aggFn(outputs)
	if err != nil {
		s.failJob(job, "aggregation error: "+err.Error())
		return
	}

	s.mu.Lock()
	job.Result = result
	job.Status = "done"
	job.FinishedAt = time.Now().UnixMilli()
	s.mu.Unlock()

	_ = s.saveToDiskPublic()
	logInfo(fmt.Sprintf("job done: %s duration=%dms", job.JobID, job.FinishedAt-job.CreatedAt))
}

func (s *Store) failJob(job *Job, errMsg string) {
	s.mu.Lock()
	job.Status = "failed"
	job.Error = errMsg
	job.FinishedAt = time.Now().UnixMilli()
	s.mu.Unlock()
	_ = s.saveToDiskPublic()
	logError(fmt.Sprintf("job failed: %s error=%s", job.JobID, errMsg))
}

type workerInfo struct {
	instanceID string
	addr       string
}

// getHealthyWorkers pings each worker and returns responsive ones.
func (s *Store) getHealthyWorkers(ctx context.Context) []workerInfo {
	s.mu.RLock()
	candidates := make([]workerInfo, len(s.workers))
	for i, w := range s.workers {
		candidates[i] = workerInfo{instanceID: w.ID, addr: w.Addr}
	}
	s.mu.RUnlock()

	// Ping all in parallel
	type result struct {
		w       workerInfo
		healthy bool
	}
	results := make(chan result, len(candidates))
	for _, w := range candidates {
		go func(w workerInfo) {
			healthy := pingWorker(ctx, w.addr)
			results <- result{w, healthy}
		}(w)
	}

	var healthy []workerInfo
	for i := 0; i < len(candidates); i++ {
		r := <-results
		if r.healthy {
			healthy = append(healthy, r.w)
		} else {
			logWarn(fmt.Sprintf("worker %s unhealthy, skipping", r.w.instanceID))
		}
	}
	return healthy
}

func pingWorker(ctx context.Context, addr string) bool {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(pingCtx, http.MethodGet, addr+"/ping", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func executeTaskWithRetry(ctx context.Context, t *Task, workers []workerInfo, maxRetry int) {
	for attempt := 0; attempt <= maxRetry; attempt++ {
		if attempt > 0 {
			t.RetryCount = attempt
			nextIdx := attempt % len(workers)
			t.WorkerID = workers[nextIdx].instanceID
			t.WorkerAddr = workers[nextIdx].addr
			t.StartedAt = time.Now().UnixMilli()
			logInfo(fmt.Sprintf("retrying task %s on %s (attempt %d)", t.TaskID, t.WorkerID, attempt))
		}

		if err := callWorker(ctx, t); err == nil && t.Status == "done" {
			return
		} else if err != nil {
			t.Error = err.Error()
			logWarn(fmt.Sprintf("task %s failed on %s: %v", t.TaskID, t.WorkerID, err))
		}

		// Check context before retry
		if ctx.Err() != nil {
			break
		}
	}
	t.Status = "failed"
	t.FinishedAt = time.Now().UnixMilli()
}

func callWorker(ctx context.Context, t *Task) error {
	payload := map[string]interface{}{
		"taskId": t.TaskID,
		"typeId": t.TypeID,
		"input":  t.Input,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := t.WorkerAddr + "/execute"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Propagate request ID if available
	if reqID, ok := ctx.Value(reqIDKey).(string); ok {
		req.Header.Set("X-Request-ID", reqID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("worker call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("worker returned %d: %s", resp.StatusCode, string(respBody))
	}

	var taskResp struct {
		Status     string      `json:"status"`
		Output     interface{} `json:"output"`
		Error      string      `json:"error"`
		DurationMs int64       `json:"durationMs"`
	}
	if err := json.Unmarshal(respBody, &taskResp); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}

	if taskResp.Status == "error" {
		return fmt.Errorf("worker error: %s", taskResp.Error)
	}

	t.Status = "done"
	t.Output = taskResp.Output
	t.DurationMs = taskResp.DurationMs
	t.FinishedAt = time.Now().UnixMilli()
	return nil
}
