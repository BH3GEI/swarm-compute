package main

import (
	"fmt"
	"math"
)

func executePrimeCount(input map[string]interface{}) (interface{}, error) {
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

	if from < 2 {
		from = 2
	}
	if to < from {
		return map[string]interface{}{"count": 0, "primes": []int{}}, nil
	}
	if to-from > 100_000_000 {
		return nil, fmt.Errorf("range too large (max 100M)")
	}

	count := 0
	var lastPrimes []int

	for n := from; n <= to; n++ {
		if isPrime(n) {
			count++
			if len(lastPrimes) < 10 {
				lastPrimes = append(lastPrimes, n)
			} else {
				// Keep last 10
				lastPrimes = append(lastPrimes[1:], n)
			}
		}
	}

	return map[string]interface{}{
		"count":  count,
		"from":   from,
		"to":     to,
		"sample": lastPrimes,
	}, nil
}

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	if n < 4 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}
	limit := int(math.Sqrt(float64(n)))
	for i := 5; i <= limit; i += 6 {
		if n%i == 0 || n%(i+2) == 0 {
			return false
		}
	}
	return true
}
