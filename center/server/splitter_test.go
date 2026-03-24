package main

import (
	"testing"
)

func TestSplitSort(t *testing.T) {
	input := map[string]interface{}{"data": []interface{}{5.0, 3.0, 1.0, 4.0, 2.0, 6.0}}
	chunks, err := splitSort(input, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	// Verify all elements preserved
	total := 0
	for _, c := range chunks {
		total += len(c["data"].([]interface{}))
	}
	if total != 6 {
		t.Fatalf("expected 6 total elements, got %d", total)
	}
}

func TestSplitWordCount(t *testing.T) {
	input := map[string]interface{}{"text": "line1\nline2\nline3\nline4"}
	chunks, err := splitWordCount(input, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
}

func TestSplitPiEstimate(t *testing.T) {
	input := map[string]interface{}{"samples": 1000000.0}
	chunks, err := splitPiEstimate(input, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}
	total := 0
	for _, c := range chunks {
		total += int(c["samples"].(float64))
	}
	if total != 1000000 {
		t.Fatalf("expected 1000000 total samples, got %d", total)
	}
}

func TestSplitPrimeCount(t *testing.T) {
	input := map[string]interface{}{"from": 1.0, "to": 100.0}
	chunks, err := splitPrimeCount(input, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	// Verify ranges are contiguous
	prev := 0.0
	for i, c := range chunks {
		from := c["from"].(float64)
		to := c["to"].(float64)
		if i > 0 && from != prev+1 {
			t.Fatalf("gap between chunk %d and %d", i-1, i)
		}
		prev = to
	}
	if prev != 100.0 {
		t.Fatalf("last chunk should end at 100, got %v", prev)
	}
}

func TestSplitGrep(t *testing.T) {
	input := map[string]interface{}{"text": "a\nb\nc\nd\ne\nf", "pattern": "test"}
	chunks, err := splitGrep(input, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	for _, c := range chunks {
		if c["pattern"].(string) != "test" {
			t.Fatal("pattern not propagated to chunk")
		}
	}
}

func TestSplitHashCrack(t *testing.T) {
	input := map[string]interface{}{
		"hash":    "abc123",
		"charset": "abcdef",
		"maxLen":  3.0,
	}
	chunks, err := splitHashCrack(input, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	// Verify full charset covered
	first := int(chunks[0]["startIdx"].(float64))
	last := int(chunks[len(chunks)-1]["endIdx"].(float64))
	if first != 0 || last != 6 {
		t.Fatalf("charset not fully covered: [%d, %d)", first, last)
	}
}

func TestSplitMatrixMul(t *testing.T) {
	input := map[string]interface{}{
		"a": []interface{}{[]interface{}{1.0, 2.0}, []interface{}{3.0, 4.0}, []interface{}{5.0, 6.0}},
		"b": []interface{}{[]interface{}{7.0, 8.0}, []interface{}{9.0, 10.0}},
	}
	chunks, err := splitMatrixMul(input, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	// Each chunk should have matrix B
	for _, c := range chunks {
		if c["b"] == nil {
			t.Fatal("chunk missing matrix B")
		}
	}
}
