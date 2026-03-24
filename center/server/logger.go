package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type LogEntry struct {
	Time   string `json:"time"`
	Level  string `json:"level"`
	Msg    string `json:"msg"`
	ReqID  string `json:"reqId,omitempty"`
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	Status int    `json:"status,omitempty"`
	Dur    string `json:"dur,omitempty"`
	Error  string `json:"error,omitempty"`
}

func logJSON(level, msg string, fields ...func(*LogEntry)) {
	e := LogEntry{
		Time:  time.Now().UTC().Format(time.RFC3339Nano),
		Level: level,
		Msg:   msg,
	}
	for _, f := range fields {
		f(&e)
	}
	b, _ := json.Marshal(e)
	fmt.Fprintln(os.Stdout, string(b))
}

func logInfo(msg string, fields ...func(*LogEntry)) {
	logJSON("INFO", msg, fields...)
}

func logWarn(msg string, fields ...func(*LogEntry)) {
	logJSON("WARN", msg, fields...)
}

func logError(msg string, fields ...func(*LogEntry)) {
	logJSON("ERROR", msg, fields...)
}

func withReqID(id string) func(*LogEntry) {
	return func(e *LogEntry) { e.ReqID = id }
}

func withErr(err error) func(*LogEntry) {
	return func(e *LogEntry) {
		if err != nil {
			e.Error = err.Error()
		}
	}
}
