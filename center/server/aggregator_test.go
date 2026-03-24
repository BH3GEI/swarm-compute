package main

import (
	"testing"
)

func TestAggregateWordCount(t *testing.T) {
	outputs := []interface{}{
		map[string]interface{}{"hello": 2.0, "world": 1.0},
		map[string]interface{}{"hello": 1.0, "foo": 3.0},
	}
	result, err := aggregateWordCount(outputs)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]int)
	if m["hello"] != 3 {
		t.Fatalf("expected hello=3, got %d", m["hello"])
	}
	if m["foo"] != 3 {
		t.Fatalf("expected foo=3, got %d", m["foo"])
	}
}

func TestAggregateSort(t *testing.T) {
	outputs := []interface{}{
		[]interface{}{1.0, 3.0, 5.0},
		[]interface{}{2.0, 4.0, 6.0},
	}
	result, err := aggregateSort(outputs)
	if err != nil {
		t.Fatal(err)
	}
	arr := result.([]float64)
	expected := []float64{1, 2, 3, 4, 5, 6}
	if len(arr) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(arr))
	}
	for i, v := range arr {
		if v != expected[i] {
			t.Fatalf("index %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestAggregatePiEstimate(t *testing.T) {
	outputs := []interface{}{
		map[string]interface{}{"inside": 750000.0, "total": 1000000.0},
		map[string]interface{}{"inside": 780000.0, "total": 1000000.0},
	}
	result, err := aggregatePiEstimate(outputs)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	pi := m["pi"].(float64)
	if pi < 3.0 || pi > 3.3 {
		t.Fatalf("pi out of range: %v", pi)
	}
}

func TestAggregatePrimeCount(t *testing.T) {
	outputs := []interface{}{
		map[string]interface{}{"count": 10.0, "from": 1.0, "to": 50.0, "sample": []interface{}{47.0}},
		map[string]interface{}{"count": 5.0, "from": 51.0, "to": 100.0, "sample": []interface{}{97.0}},
	}
	result, err := aggregatePrimeCount(outputs)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	if m["count"].(int) != 15 {
		t.Fatalf("expected count=15, got %v", m["count"])
	}
}

func TestAggregateGrep(t *testing.T) {
	outputs := []interface{}{
		map[string]interface{}{
			"count":   2.0,
			"matches": []interface{}{
				map[string]interface{}{"line": 3.0, "content": "line3"},
				map[string]interface{}{"line": 1.0, "content": "line1"},
			},
		},
	}
	result, err := aggregateGrep(outputs)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	if m["count"].(int) != 2 {
		t.Fatalf("expected count=2, got %v", m["count"])
	}
	// Verify sorted by line number
	matches := m["matches"].([]map[string]interface{})
	if matches[0]["line"].(float64) > matches[1]["line"].(float64) {
		t.Fatal("matches not sorted by line number")
	}
}

func TestAggregateHashCrack_Found(t *testing.T) {
	outputs := []interface{}{
		map[string]interface{}{"found": false, "tried": 100.0},
		map[string]interface{}{"found": true, "value": "test", "tried": 50.0},
	}
	result, err := aggregateHashCrack(outputs)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	if m["found"].(bool) != true {
		t.Fatal("expected found=true")
	}
	if m["value"].(string) != "test" {
		t.Fatalf("expected value=test, got %v", m["value"])
	}
}

func TestAggregateHashCrack_NotFound(t *testing.T) {
	outputs := []interface{}{
		map[string]interface{}{"found": false, "tried": 100.0},
		map[string]interface{}{"found": false, "tried": 200.0},
	}
	result, err := aggregateHashCrack(outputs)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]interface{})
	if m["found"].(bool) != false {
		t.Fatal("expected found=false")
	}
	if m["totalTried"].(int) != 300 {
		t.Fatalf("expected totalTried=300, got %v", m["totalTried"])
	}
}
