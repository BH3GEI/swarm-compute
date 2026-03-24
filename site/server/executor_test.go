package main

import (
	"crypto/md5"
	"encoding/hex"
	"testing"
)

func TestExecuteWordCount(t *testing.T) {
	result, err := executeWordCount(map[string]interface{}{"text": "hello world hello"})
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]int)
	if m["hello"] != 2 {
		t.Fatalf("expected hello=2, got %d", m["hello"])
	}
	if m["world"] != 1 {
		t.Fatalf("expected world=1, got %d", m["world"])
	}
}

func TestExecuteSort(t *testing.T) {
	result, err := executeSort(map[string]interface{}{
		"data": []interface{}{5.0, 1.0, 3.0, 2.0, 4.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	arr := result.([]float64)
	expected := []float64{1, 2, 3, 4, 5}
	for i, v := range arr {
		if v != expected[i] {
			t.Fatalf("index %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestExecuteSort_TooLarge(t *testing.T) {
	big := make([]interface{}, 10_000_001)
	for i := range big {
		big[i] = 1.0
	}
	_, err := executeSort(map[string]interface{}{"data": big})
	if err == nil {
		t.Fatal("expected error for oversized array")
	}
}

func TestExecuteMatrixMul(t *testing.T) {
	result, err := executeMatrixMul(map[string]interface{}{
		"rows": []interface{}{[]interface{}{1.0, 2.0}},
		"b":    []interface{}{[]interface{}{3.0}, []interface{}{4.0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	rows := m["rows"].([][]float64)
	if rows[0][0] != 11.0 { // 1*3 + 2*4
		t.Fatalf("expected 11, got %v", rows[0][0])
	}
}

func TestExecutePiEstimate(t *testing.T) {
	result, err := executePiEstimate(map[string]interface{}{"samples": 100000.0})
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	inside := m["inside"].(int)
	total := m["total"].(int)
	if total != 100000 {
		t.Fatalf("expected total=100000, got %d", total)
	}
	// Pi should be roughly 3.14 ± 0.1
	pi := 4.0 * float64(inside) / float64(total)
	if pi < 3.0 || pi > 3.3 {
		t.Fatalf("pi estimate out of range: %v", pi)
	}
}

func TestExecuteGrep(t *testing.T) {
	result, err := executeGrep(map[string]interface{}{
		"text":      "hello world\nfoo bar\nhello again",
		"pattern":   "hello",
		"startLine": 0.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	count := m["count"].(int)
	if count != 2 {
		t.Fatalf("expected 2 matches, got %d", count)
	}
}

func TestExecutePrimeCount(t *testing.T) {
	result, err := executePrimeCount(map[string]interface{}{
		"from": 1.0,
		"to":   100.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	count := m["count"].(int)
	if count != 25 { // there are 25 primes below 100
		t.Fatalf("expected 25 primes in [1,100], got %d", count)
	}
}

func TestExecuteHashCrack_Found(t *testing.T) {
	target := md5.Sum([]byte("ab"))
	hash := hex.EncodeToString(target[:])

	result, err := executeHashCrack(map[string]interface{}{
		"hash":     hash,
		"charset":  "ab",
		"maxLen":   2.0,
		"startIdx": 0.0,
		"endIdx":   2.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	if !m["found"].(bool) {
		t.Fatal("expected to find 'ab'")
	}
	if m["value"].(string) != "ab" {
		t.Fatalf("expected value='ab', got %v", m["value"])
	}
}

func TestExecuteHashCrack_NotFound(t *testing.T) {
	result, err := executeHashCrack(map[string]interface{}{
		"hash":     "0000000000000000000000000000dead",
		"charset":  "ab",
		"maxLen":   2.0,
		"startIdx": 0.0,
		"endIdx":   2.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	if m["found"].(bool) {
		t.Fatal("expected not found")
	}
}
