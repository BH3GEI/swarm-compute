package main

import (
	"fmt"
	"strings"
	"unicode"
)

func executeWordCount(input map[string]interface{}) (interface{}, error) {
	text, ok := input["text"].(string)
	if !ok {
		return nil, fmt.Errorf("input.text must be a string")
	}

	counts := make(map[string]int)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, w := range words {
		counts[strings.ToLower(w)]++
	}
	return counts, nil
}
