package main

import (
	"crypto/rand"
	"encoding/hex"
)

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

func minMaxNorm(vals []float64) []float64 {
	if len(vals) == 0 {
		return vals
	}
	minv, maxv := vals[0], vals[0]
	for _, v := range vals {
		if v < minv {
			minv = v
		}
		if v > maxv {
			maxv = v
		}
	}
	out := make([]float64, len(vals))
	den := maxv - minv
	if den == 0 {
		return out // all zeros
	}
	for i, v := range vals {
		out[i] = (v - minv) / den
	}
	return out
}
