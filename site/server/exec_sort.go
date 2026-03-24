package main

import (
	"fmt"
	"sort"
)

func executeSort(input map[string]interface{}) (interface{}, error) {
	raw, ok := input["data"]
	if !ok {
		return nil, fmt.Errorf("input.data is required")
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("input.data must be an array")
	}
	if len(arr) > 10_000_000 {
		return nil, fmt.Errorf("array too large (%d elements, max 10M)", len(arr))
	}

	nums := make([]float64, len(arr))
	for i, v := range arr {
		n, ok := v.(float64)
		if !ok {
			return nil, fmt.Errorf("input.data[%d] is not a number", i)
		}
		nums[i] = n
	}

	sort.Float64s(nums)
	return nums, nil
}
