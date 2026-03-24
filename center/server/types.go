package main

// Worker represents a compute node in the cluster.
type Worker struct {
	ID   string `json:"id"`
	Addr string `json:"addr"` // e.g. "http://worker-1:9000"
}
