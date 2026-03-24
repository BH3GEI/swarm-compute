package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

func executeHashCrack(input map[string]interface{}) (interface{}, error) {
	hash, ok := input["hash"].(string)
	if !ok {
		return nil, fmt.Errorf("input.hash must be a string")
	}
	hash = strings.ToLower(strings.TrimSpace(hash))

	charset, ok := input["charset"].(string)
	if !ok || charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyz"
	}

	maxLen := 4
	if ml, ok := input["maxLen"].(float64); ok && ml > 0 {
		maxLen = int(ml)
	}
	if maxLen > 6 {
		return nil, fmt.Errorf("maxLen too large (max 6)")
	}

	// Range within charset to search (for splitting across workers)
	startIdx := 0
	endIdx := len(charset)
	if si, ok := input["startIdx"].(float64); ok {
		startIdx = int(si)
	}
	if ei, ok := input["endIdx"].(float64); ok {
		endIdx = int(ei)
	}

	chars := []byte(charset)
	tried := 0

	// Search all strings of length 1..maxLen where first char is in [startIdx, endIdx)
	for length := 1; length <= maxLen; length++ {
		buf := make([]byte, length)
		found, value := searchDepth(buf, 0, chars, startIdx, endIdx, hash, &tried, length)
		if found {
			return map[string]interface{}{
				"found": true,
				"value": value,
				"tried": tried,
			}, nil
		}
	}

	return map[string]interface{}{
		"found": false,
		"tried": tried,
	}, nil
}

func searchDepth(buf []byte, pos int, chars []byte, startIdx, endIdx int, targetHash string, tried *int, maxLen int) (bool, string) {
	if pos == len(buf) {
		*tried++
		h := md5.Sum(buf)
		if hex.EncodeToString(h[:]) == targetHash {
			return true, string(buf)
		}
		return false, ""
	}

	lo, hi := 0, len(chars)
	if pos == 0 {
		lo = startIdx
		hi = endIdx
	}

	for i := lo; i < hi; i++ {
		buf[pos] = chars[i]
		if found, val := searchDepth(buf, pos+1, chars, startIdx, endIdx, targetHash, tried, maxLen); found {
			return true, val
		}
	}
	return false, ""
}
