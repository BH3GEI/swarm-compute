package main

import (
	"fmt"
	"regexp"
	"strings"
)

func executeGrep(input map[string]interface{}) (interface{}, error) {
	text, ok := input["text"].(string)
	if !ok {
		return nil, fmt.Errorf("input.text must be a string")
	}
	pattern, ok := input["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("input.pattern must be a string")
	}
	// startLine offset for correct global line numbering
	startLine := 0
	if sl, ok := input["startLine"].(float64); ok {
		startLine = int(sl)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	lines := strings.Split(text, "\n")
	var matches []map[string]interface{}

	for i, line := range lines {
		if re.MatchString(line) {
			matches = append(matches, map[string]interface{}{
				"line":    startLine + i + 1,
				"content": line,
			})
		}
	}

	return map[string]interface{}{
		"matches": matches,
		"count":   len(matches),
	}, nil
}
