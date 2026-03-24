package main

import "fmt"

// executeMatrixMul multiplies a sub-block of rows from matrix A with full matrix B.
// Input: {"rows": [[...], ...], "b": [[...], ...]}
// Output: {"rows": [[...], ...]}
func executeMatrixMul(input map[string]interface{}) (interface{}, error) {
	rowsRaw, ok := input["rows"]
	if !ok {
		return nil, fmt.Errorf("input.rows is required")
	}
	bRaw, ok := input["b"]
	if !ok {
		return nil, fmt.Errorf("input.b is required")
	}

	rows, err := toMatrix(rowsRaw)
	if err != nil {
		return nil, fmt.Errorf("input.rows: %w", err)
	}
	b, err := toMatrix(bRaw)
	if err != nil {
		return nil, fmt.Errorf("input.b: %w", err)
	}

	if len(rows) == 0 || len(b) == 0 {
		return map[string]interface{}{"rows": [][]float64{}}, nil
	}
	colsA := len(rows[0])
	rowsB := len(b)
	if colsA != rowsB {
		return nil, fmt.Errorf("dimension mismatch: A cols=%d, B rows=%d", colsA, rowsB)
	}
	colsB := len(b[0])

	result := make([][]float64, len(rows))
	for i, row := range rows {
		result[i] = make([]float64, colsB)
		for j := 0; j < colsB; j++ {
			var sum float64
			for k := 0; k < colsA; k++ {
				sum += row[k] * b[k][j]
			}
			result[i][j] = sum
		}
	}
	return map[string]interface{}{"rows": result}, nil
}

func toMatrix(raw interface{}) ([][]float64, error) {
	outerArr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected 2D array")
	}
	mat := make([][]float64, len(outerArr))
	for i, rowRaw := range outerArr {
		rowArr, ok := rowRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("row %d is not an array", i)
		}
		mat[i] = make([]float64, len(rowArr))
		for j, v := range rowArr {
			n, ok := v.(float64)
			if !ok {
				return nil, fmt.Errorf("[%d][%d] is not a number", i, j)
			}
			mat[i][j] = n
		}
	}
	return mat, nil
}
