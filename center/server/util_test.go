package main

import (
	"strings"
	"testing"
)

func TestNewID(t *testing.T) {
	id1 := newID("job")
	id2 := newID("job")
	if id1 == id2 {
		t.Fatal("IDs should be unique")
	}
	if !strings.HasPrefix(id1, "job_") {
		t.Fatalf("expected prefix 'job_', got %s", id1)
	}
	if len(id1) != 3+1+16 { // "job" + "_" + 16 hex chars
		t.Fatalf("unexpected ID length: %d", len(id1))
	}
}

func TestMinMaxNorm(t *testing.T) {
	vals := []float64{10, 20, 30}
	result := minMaxNorm(vals)
	if result[0] != 0 {
		t.Fatalf("expected 0, got %v", result[0])
	}
	if result[1] != 0.5 {
		t.Fatalf("expected 0.5, got %v", result[1])
	}
	if result[2] != 1.0 {
		t.Fatalf("expected 1.0, got %v", result[2])
	}
}

func TestMinMaxNorm_AllSame(t *testing.T) {
	vals := []float64{5, 5, 5}
	result := minMaxNorm(vals)
	for i, v := range result {
		if v != 0 {
			t.Fatalf("index %d: expected 0, got %v", i, v)
		}
	}
}

func TestMinMaxNorm_Empty(t *testing.T) {
	result := minMaxNorm(nil)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestMinMaxNorm_Single(t *testing.T) {
	result := minMaxNorm([]float64{42})
	if result[0] != 0 {
		t.Fatalf("expected 0, got %v", result[0])
	}
}
