package main

import (
	"fmt"
	"sort"
)

// AggregateFunc merges outputs from all tasks into a single job result.
type AggregateFunc func(outputs []interface{}) (interface{}, error)

var aggregators = map[string]AggregateFunc{
	"word-count":  aggregateWordCount,
	"sort":        aggregateSort,
	"matrix-mul":  aggregateMatrixMul,
	"pi-estimate": aggregatePiEstimate,
	"grep":        aggregateGrep,
	"prime-count": aggregatePrimeCount,
	"hash-crack":  aggregateHashCrack,
}

// aggregateWordCount merges word frequency maps.
func aggregateWordCount(outputs []interface{}) (interface{}, error) {
	merged := map[string]float64{}
	for _, out := range outputs {
		m, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		for k, v := range m {
			if n, ok := v.(float64); ok {
				merged[k] += n
			}
		}
	}
	// Convert to int for cleaner output
	result := map[string]int{}
	for k, v := range merged {
		result[k] = int(v)
	}
	return result, nil
}

// aggregateSort merges pre-sorted sub-arrays via k-way merge.
func aggregateSort(outputs []interface{}) (interface{}, error) {
	var arrays [][]float64
	for _, out := range outputs {
		arr, ok := out.([]interface{})
		if !ok {
			continue
		}
		nums := make([]float64, len(arr))
		for i, v := range arr {
			n, ok := v.(float64)
			if !ok {
				return nil, fmt.Errorf("non-numeric value in sorted output")
			}
			nums[i] = n
		}
		arrays = append(arrays, nums)
	}
	return mergeKSorted(arrays), nil
}

func mergeKSorted(arrays [][]float64) []float64 {
	if len(arrays) == 0 {
		return nil
	}
	if len(arrays) == 1 {
		return arrays[0]
	}
	mid := len(arrays) / 2
	left := mergeKSorted(arrays[:mid])
	right := mergeKSorted(arrays[mid:])
	return mergeTwo(left, right)
}

func mergeTwo(a, b []float64) []float64 {
	result := make([]float64, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			result = append(result, a[i])
			i++
		} else {
			result = append(result, b[j])
			j++
		}
	}
	result = append(result, a[i:]...)
	result = append(result, b[j:]...)
	return result
}

// aggregatePiEstimate combines inside/total from all workers.
func aggregatePiEstimate(outputs []interface{}) (interface{}, error) {
	totalInside := 0.0
	totalSamples := 0.0
	for _, out := range outputs {
		m, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		if v, ok := m["inside"].(float64); ok {
			totalInside += v
		}
		if v, ok := m["total"].(float64); ok {
			totalSamples += v
		}
	}
	if totalSamples == 0 {
		return nil, fmt.Errorf("no samples")
	}
	pi := 4.0 * totalInside / totalSamples
	return map[string]interface{}{
		"pi":           pi,
		"totalInside":  int(totalInside),
		"totalSamples": int(totalSamples),
		"error":        fmt.Sprintf("%.10f", 3.14159265358979323846-pi),
	}, nil
}

// aggregateGrep merges matched lines and sorts by line number.
func aggregateGrep(outputs []interface{}) (interface{}, error) {
	var allMatches []map[string]interface{}
	totalCount := 0
	for _, out := range outputs {
		m, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		if c, ok := m["count"].(float64); ok {
			totalCount += int(c)
		}
		if matches, ok := m["matches"].([]interface{}); ok {
			for _, match := range matches {
				if mm, ok := match.(map[string]interface{}); ok {
					allMatches = append(allMatches, mm)
				}
			}
		}
	}
	// Sort by line number
	sort.Slice(allMatches, func(i, j int) bool {
		li, _ := allMatches[i]["line"].(float64)
		lj, _ := allMatches[j]["line"].(float64)
		return li < lj
	})
	return map[string]interface{}{
		"matches": allMatches,
		"count":   totalCount,
	}, nil
}

// aggregatePrimeCount sums prime counts and merges sample primes.
func aggregatePrimeCount(outputs []interface{}) (interface{}, error) {
	totalCount := 0
	var allSamples []float64
	minFrom := 0.0
	maxTo := 0.0
	first := true
	for _, out := range outputs {
		m, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		if c, ok := m["count"].(float64); ok {
			totalCount += int(c)
		}
		if f, ok := m["from"].(float64); ok {
			if first || f < minFrom {
				minFrom = f
			}
		}
		if t, ok := m["to"].(float64); ok {
			if first || t > maxTo {
				maxTo = t
			}
		}
		first = false
		if sample, ok := m["sample"].([]interface{}); ok {
			for _, s := range sample {
				if n, ok := s.(float64); ok {
					allSamples = append(allSamples, n)
				}
			}
		}
	}
	// Keep last 20 primes
	sort.Float64s(allSamples)
	if len(allSamples) > 20 {
		allSamples = allSamples[len(allSamples)-20:]
	}
	return map[string]interface{}{
		"count":  totalCount,
		"from":   int(minFrom),
		"to":     int(maxTo),
		"sample": allSamples,
	}, nil
}

// aggregateHashCrack returns the first successful crack.
func aggregateHashCrack(outputs []interface{}) (interface{}, error) {
	totalTried := 0
	for _, out := range outputs {
		m, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		if t, ok := m["tried"].(float64); ok {
			totalTried += int(t)
		}
		if found, ok := m["found"].(bool); ok && found {
			return map[string]interface{}{
				"found":      true,
				"value":      m["value"],
				"totalTried": totalTried,
			}, nil
		}
	}
	return map[string]interface{}{
		"found":      false,
		"totalTried": totalTried,
	}, nil
}

// aggregateMatrixMul concatenates row blocks.
func aggregateMatrixMul(outputs []interface{}) (interface{}, error) {
	var allRows []interface{}
	for _, out := range outputs {
		m, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		rows, ok := m["rows"]
		if !ok {
			continue
		}
		rowArr, ok := rows.([]interface{})
		if !ok {
			continue
		}
		allRows = append(allRows, rowArr...)
	}
	return allRows, nil
}
