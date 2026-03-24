package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var store = NewStore()
var runner *JobRunner

const (
	maxBodyDefault = 1 << 20  // 1MB
	maxBodyJob     = 10 << 20 // 10MB
)

func main() {
	initBuiltinTaskTypes()
	initWorkers()

	// Connect to RabbitMQ
	mq, err := ConnectMQ()
	if err != nil {
		logError("rabbitmq connection failed", withErr(err))
		os.Exit(1)
	}
	defer mq.Close()

	runner = NewJobRunner(store, mq, 8)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	runner.Start(ctx)

	generalRL := NewRateLimiter(50, 100)
	jobRL := NewRateLimiter(10, 20)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Distributed computing API
	mux.HandleFunc("/api/task-types", withCORS(withRateLimit(generalRL, taskTypesHandler)))
	mux.HandleFunc("/api/jobs/submit", withCORS(withRateLimit(jobRL, withMaxBody(maxBodyJob, jobSubmitHandler))))
	mux.HandleFunc("/api/jobs", withCORS(withRateLimit(generalRL, jobsListHandler)))
	mux.HandleFunc("/api/jobs/", withCORS(withRateLimit(generalRL, jobDetailOrCancelHandler)))
	mux.HandleFunc("/api/workers", withCORS(withRateLimit(generalRL, workersHandler)))
	mux.HandleFunc("/api/stats", withCORS(withRateLimit(generalRL, statsHandler)))

	handler := withLogging(mux)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		logInfo("shutting down server...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			logError("server shutdown error", withErr(err))
		}
		runner.Stop()
		if err := store.SaveToDisk(); err != nil {
			logError("final save failed", withErr(err))
		}
	}()

	logInfo(fmt.Sprintf("center listening on :%s", port))
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		logError("server error", withErr(err))
		os.Exit(1)
	}
	logInfo("server stopped")
}

func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// -------- init --------

func initBuiltinTaskTypes() {
	builtins := []TaskType{
		{TypeID: "word-count", Name: "Word Count", Description: "Count word frequencies in text (MapReduce style)"},
		{TypeID: "sort", Name: "Distributed Sort", Description: "Sort a numeric array using distributed merge-sort"},
		{TypeID: "matrix-mul", Name: "Matrix Multiply", Description: "Multiply two matrices with row-block partitioning"},
		{TypeID: "pi-estimate", Name: "Monte Carlo Pi", Description: "Estimate Pi using Monte Carlo random sampling"},
		{TypeID: "grep", Name: "Distributed Grep", Description: "Search text for regex pattern across workers"},
		{TypeID: "prime-count", Name: "Prime Count", Description: "Count prime numbers in a given range"},
		{TypeID: "hash-crack", Name: "Hash Crack", Description: "Brute-force search for MD5 hash collision"},
	}
	store.mu.Lock()
	for _, tt := range builtins {
		if _, exists := store.taskTypes[tt.TypeID]; !exists {
			store.taskTypes[tt.TypeID] = tt
		}
	}
	store.mu.Unlock()
}

func initWorkers() {
	addrs := os.Getenv("WORKER_ADDRS")
	if addrs == "" {
		addrs = "worker-1:9000,worker-2:9000,worker-3:9000,worker-4:9000,worker-5:9000"
	}

	var workers []Worker
	for _, addr := range strings.Split(addrs, ",") {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		host := strings.Split(addr, ":")[0]
		workers = append(workers, Worker{
			ID:   host,
			Addr: "http://" + addr,
		})
	}

	store.mu.Lock()
	store.workers = workers
	store.mu.Unlock()

	logInfo(fmt.Sprintf("registered %d workers", len(workers)))
}

// -------- handlers --------

func taskTypesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	store.mu.RLock()
	var list []TaskType
	for _, v := range store.taskTypes {
		list = append(list, v)
	}
	store.mu.RUnlock()
	writeJSON(w, map[string]any{"taskTypes": list})
}

func jobSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req JobSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	job, err := store.CreateJob(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	runner.Enqueue(job.JobID)
	writeJSON(w, job)
}

func jobsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	store.mu.RLock()
	list := make([]*Job, 0, len(store.jobs))
	for _, j := range store.jobs {
		list = append(list, j)
	}
	store.mu.RUnlock()
	writeJSON(w, map[string]any{"jobs": list})
}

func jobDetailOrCancelHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/jobs/")

	// POST /api/jobs/{id}/cancel
	if strings.HasSuffix(path, "/cancel") {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		jobID := strings.TrimSuffix(path, "/cancel")
		store.mu.Lock()
		job, ok := store.jobs[jobID]
		if ok && job.cancel != nil {
			job.cancel()
			job.Status = "cancelled"
			job.FinishedAt = time.Now().UnixMilli()
		}
		store.mu.Unlock()
		if !ok {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
		return
	}

	// GET /api/jobs/{id}
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	jobID := path
	if jobID == "" {
		http.Error(w, "missing jobId", http.StatusBadRequest)
		return
	}
	store.mu.RLock()
	job, ok := store.jobs[jobID]
	store.mu.RUnlock()
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	writeJSON(w, job)
}

func workersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		store.mu.RLock()
		writeJSON(w, map[string]any{"workers": store.workers})
		store.mu.RUnlock()

	case http.MethodPut:
		// PUT /api/workers — replace worker list (for benchmarking)
		var body struct {
			Workers []Worker `json:"workers"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		store.mu.Lock()
		store.workers = body.Workers
		store.mu.Unlock()
		logInfo(fmt.Sprintf("workers updated: %d workers", len(body.Workers)))
		writeJSON(w, map[string]any{"ok": true, "count": len(body.Workers)})

	default:
		http.Error(w, "GET or PUT", http.StatusMethodNotAllowed)
	}
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, store.ComputeStats())
}
