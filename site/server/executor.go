package main

import (
	"fmt"
	"time"
)

// ExecuteFunc processes a task input and returns the output.
type ExecuteFunc func(input map[string]interface{}) (interface{}, error)

// Registry of built-in executors.
var executors = map[string]ExecuteFunc{
	"word-count":  executeWordCount,
	"sort":        executeSort,
	"matrix-mul":  executeMatrixMul,
	"pi-estimate": executePiEstimate,
	"grep":        executeGrep,
	"prime-count": executePrimeCount,
	"hash-crack":  executeHashCrack,
}

// runTask dispatches a TaskRequest to the appropriate executor.
func runTask(instanceID string, req TaskRequest) TaskResponse {
	fn, ok := executors[req.TypeID]
	if !ok {
		return TaskResponse{
			TaskID:     req.TaskID,
			InstanceID: instanceID,
			Status:     "error",
			Error:      fmt.Sprintf("unknown task type: %s", req.TypeID),
		}
	}

	start := time.Now()
	output, err := fn(req.Input)
	dur := time.Since(start).Milliseconds()

	if err != nil {
		return TaskResponse{
			TaskID:     req.TaskID,
			InstanceID: instanceID,
			Status:     "error",
			Error:      err.Error(),
			DurationMs: dur,
		}
	}
	return TaskResponse{
		TaskID:     req.TaskID,
		InstanceID: instanceID,
		Status:     "done",
		Output:     output,
		DurationMs: dur,
	}
}
