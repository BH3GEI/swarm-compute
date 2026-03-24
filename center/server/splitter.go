package main

import (
	"fmt"
	"math"
	"strings"
)

// SplitFunc splits a job input into N sub-task inputs.
type SplitFunc func(input map[string]interface{}, n int) ([]map[string]interface{}, error)

var splitters = map[string]SplitFunc{
	"word-count":  splitWordCount,
	"sort":        splitSort,
	"matrix-mul":  splitMatrixMul,
	"pi-estimate": splitPiEstimate,
	"grep":        splitGrep,
	"prime-count": splitPrimeCount,
	"hash-crack":  splitHashCrack,
}

// splitWordCount splits text into n chunks by lines.
func splitWordCount(input map[string]interface{}, n int) ([]map[string]interface{}, error) {
	text, ok := input["text"].(string)
	if !ok {
		return nil, fmt.Errorf("input.text must be a string")
	}
	lines := strings.Split(text, "\n")
	chunks := splitSlice(lines, n)
	out := make([]map[string]interface{}, len(chunks))
	for i, chunk := range chunks {
		out[i] = map[string]interface{}{"text": strings.Join(chunk, "\n")}
	}
	return out, nil
}

// splitSort splits a numeric array into n roughly equal parts.
func splitSort(input map[string]interface{}, n int) ([]map[string]interface{}, error) {
	raw, ok := input["data"]
	if !ok {
		return nil, fmt.Errorf("input.data is required")
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("input.data must be an array")
	}
	chunks := splitSliceGeneric(arr, n)
	out := make([]map[string]interface{}, len(chunks))
	for i, chunk := range chunks {
		out[i] = map[string]interface{}{"data": chunk}
	}
	return out, nil
}

// splitMatrixMul splits matrix A by rows; each chunk gets its rows + full B.
func splitMatrixMul(input map[string]interface{}, n int) ([]map[string]interface{}, error) {
	aRaw, ok := input["a"]
	if !ok {
		return nil, fmt.Errorf("input.a is required")
	}
	bRaw, ok := input["b"]
	if !ok {
		return nil, fmt.Errorf("input.b is required")
	}
	aArr, ok := aRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("input.a must be a 2D array")
	}
	chunks := splitSliceGeneric(aArr, n)
	out := make([]map[string]interface{}, len(chunks))
	for i, chunk := range chunks {
		out[i] = map[string]interface{}{
			"rows": chunk,
			"b":    bRaw,
		}
	}
	return out, nil
}

// splitPiEstimate divides samples across workers.
func splitPiEstimate(input map[string]interface{}, n int) ([]map[string]interface{}, error) {
	samplesRaw, ok := input["samples"]
	if !ok {
		return nil, fmt.Errorf("input.samples is required")
	}
	total := int(samplesRaw.(float64))
	if total < 1 {
		return nil, fmt.Errorf("input.samples must be positive")
	}
	perWorker := total / n
	out := make([]map[string]interface{}, n)
	for i := 0; i < n; i++ {
		s := perWorker
		if i == n-1 {
			s = total - perWorker*(n-1) // last worker gets remainder
		}
		out[i] = map[string]interface{}{"samples": float64(s)}
	}
	return out, nil
}

// splitGrep splits text by lines, each chunk gets the pattern + startLine offset.
func splitGrep(input map[string]interface{}, n int) ([]map[string]interface{}, error) {
	text, ok := input["text"].(string)
	if !ok {
		return nil, fmt.Errorf("input.text must be a string")
	}
	pattern, ok := input["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("input.pattern must be a string")
	}
	lines := strings.Split(text, "\n")
	chunks := splitSlice(lines, n)
	out := make([]map[string]interface{}, len(chunks))
	lineOffset := 0
	for i, chunk := range chunks {
		out[i] = map[string]interface{}{
			"text":      strings.Join(chunk, "\n"),
			"pattern":   pattern,
			"startLine": float64(lineOffset),
		}
		lineOffset += len(chunk)
	}
	return out, nil
}

// splitPrimeCount divides a range [from, to] across workers.
func splitPrimeCount(input map[string]interface{}, n int) ([]map[string]interface{}, error) {
	fromRaw, ok := input["from"]
	if !ok {
		return nil, fmt.Errorf("input.from is required")
	}
	toRaw, ok := input["to"]
	if !ok {
		return nil, fmt.Errorf("input.to is required")
	}
	from := int(fromRaw.(float64))
	to := int(toRaw.(float64))
	if from > to {
		return nil, fmt.Errorf("from must be <= to")
	}

	rangeSize := to - from + 1
	perWorker := rangeSize / n
	if perWorker < 1 {
		perWorker = 1
	}

	var out []map[string]interface{}
	cur := from
	for i := 0; i < n && cur <= to; i++ {
		end := cur + perWorker - 1
		if i == n-1 || end > to {
			end = to
		}
		out = append(out, map[string]interface{}{
			"from": float64(cur),
			"to":   float64(end),
		})
		cur = end + 1
	}
	return out, nil
}

// splitHashCrack partitions the charset range across workers.
func splitHashCrack(input map[string]interface{}, n int) ([]map[string]interface{}, error) {
	hash, ok := input["hash"].(string)
	if !ok {
		return nil, fmt.Errorf("input.hash must be a string")
	}
	charset := "abcdefghijklmnopqrstuvwxyz"
	if cs, ok := input["charset"].(string); ok && cs != "" {
		charset = cs
	}
	maxLen := 4.0
	if ml, ok := input["maxLen"].(float64); ok && ml > 0 {
		maxLen = ml
	}

	charCount := len(charset)
	if n > charCount {
		n = charCount
	}
	perWorker := charCount / n

	var out []map[string]interface{}
	for i := 0; i < n; i++ {
		startIdx := i * perWorker
		endIdx := (i + 1) * perWorker
		if i == n-1 {
			endIdx = charCount
		}
		out = append(out, map[string]interface{}{
			"hash":     hash,
			"charset":  charset,
			"maxLen":   maxLen,
			"startIdx": float64(startIdx),
			"endIdx":   float64(endIdx),
		})
	}
	return out, nil
}

// splitSlice splits a string slice into n roughly equal parts.
func splitSlice(s []string, n int) [][]string {
	if n <= 0 {
		n = 1
	}
	if n > len(s) {
		n = len(s)
	}
	if n == 0 {
		return nil
	}
	size := int(math.Ceil(float64(len(s)) / float64(n)))
	var out [][]string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}

// splitSliceGeneric splits an interface slice into n roughly equal parts.
func splitSliceGeneric(s []interface{}, n int) [][]interface{} {
	if n <= 0 {
		n = 1
	}
	if n > len(s) {
		n = len(s)
	}
	if n == 0 {
		return nil
	}
	size := int(math.Ceil(float64(len(s)) / float64(n)))
	var out [][]interface{}
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
