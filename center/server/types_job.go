package main

import "context"

// TaskType describes a registered computation type.
type TaskType struct {
	TypeID      string `json:"typeId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Job represents a user-submitted distributed computation.
type Job struct {
	JobID      string                 `json:"jobId"`
	TypeID     string                 `json:"typeId"`
	Status     string                 `json:"status"` // pending|splitting|running|aggregating|done|failed|cancelled
	Input      map[string]interface{} `json:"input"`
	Result     interface{}            `json:"result,omitempty"`
	Tasks      []*Task                `json:"tasks"`
	Error      string                 `json:"error,omitempty"`
	CreatedAt  int64                  `json:"createdAt"`
	FinishedAt int64                  `json:"finishedAt,omitempty"`
	MaxRetry   int                    `json:"maxRetry"`
	TimeoutSec int                    `json:"timeoutSec"`

	cancel context.CancelFunc `json:"-"`
}

// Task is a sub-unit of a Job, executed on a single worker.
type Task struct {
	TaskID     string                 `json:"taskId"`
	JobID      string                 `json:"jobId"`
	TypeID     string                 `json:"typeId"`
	Status     string                 `json:"status"` // pending|running|done|failed
	Input      map[string]interface{} `json:"input"`
	Output     interface{}            `json:"output,omitempty"`
	WorkerID   string                 `json:"workerId"`
	WorkerAddr string                 `json:"workerAddr"`
	RetryCount int                    `json:"retryCount"`
	Error      string                 `json:"error,omitempty"`
	StartedAt  int64                  `json:"startedAt,omitempty"`
	FinishedAt int64                  `json:"finishedAt,omitempty"`
	DurationMs int64                  `json:"durationMs,omitempty"`
}

// JobSubmitRequest is the API request to submit a new job.
type JobSubmitRequest struct {
	TypeID     string                 `json:"typeId"`
	Input      map[string]interface{} `json:"input"`
	MaxRetry   int                    `json:"maxRetry,omitempty"`
	TimeoutSec int                    `json:"timeoutSec,omitempty"`
}
