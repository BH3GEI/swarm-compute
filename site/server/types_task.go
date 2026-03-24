package main

// TaskRequest is sent by Center to Worker for distributed task execution.
type TaskRequest struct {
	TaskID string                 `json:"taskId"`
	TypeID string                 `json:"typeId"`
	Input  map[string]interface{} `json:"input"`
}

// TaskResponse is returned by Worker after executing a task.
type TaskResponse struct {
	TaskID     string      `json:"taskId"`
	InstanceID string      `json:"instanceId"`
	Status     string      `json:"status"` // "done" | "error"
	Output     interface{} `json:"output"`
	Error      string      `json:"error,omitempty"`
	DurationMs int64       `json:"durationMs"`
}
