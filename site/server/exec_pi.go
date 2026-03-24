package main

import (
	"fmt"
	"math/rand"
)

func executePiEstimate(input map[string]interface{}) (interface{}, error) {
	samplesRaw, ok := input["samples"]
	if !ok {
		return nil, fmt.Errorf("input.samples is required")
	}
	samples, ok := samplesRaw.(float64)
	if !ok || samples < 1 {
		return nil, fmt.Errorf("input.samples must be a positive number")
	}
	if samples > 1e9 {
		return nil, fmt.Errorf("input.samples too large (max 1e9)")
	}

	n := int(samples)
	inside := 0
	rng := rand.New(rand.NewSource(rand.Int63()))

	for i := 0; i < n; i++ {
		x := rng.Float64()
		y := rng.Float64()
		if x*x+y*y <= 1.0 {
			inside++
		}
	}

	return map[string]interface{}{
		"inside": inside,
		"total":  n,
	}, nil
}
