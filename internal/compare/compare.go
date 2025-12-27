// Package compare provides field-level JSON comparison functionality.
package compare

import (
	"encoding/json"
	"fmt"
	"reflect"

	"go.carr.sh/litmus/internal/types"
)

// Compare performs a deep comparison between expected and actual JSON values.
// It returns a list of field differences found.
func Compare(expected, actual json.RawMessage) ([]types.FieldDiff, error) {
	var expectedVal, actualVal any

	if err := json.Unmarshal(expected, &expectedVal); err != nil {
		return nil, fmt.Errorf("failed to parse expected JSON: %w", err)
	}

	if err := json.Unmarshal(actual, &actualVal); err != nil {
		return nil, fmt.Errorf("failed to parse actual JSON: %w", err)
	}

	var diffs []types.FieldDiff
	compareValues("", expectedVal, actualVal, &diffs)
	return diffs, nil
}

// compareValues recursively compares two values and collects differences.
func compareValues(path string, expected, actual any, diffs *[]types.FieldDiff) {
	// Handle nil cases
	if expected == nil && actual == nil {
		return
	}
	if expected == nil || actual == nil {
		*diffs = append(*diffs, types.FieldDiff{
			Path:     pathOrRoot(path),
			Expected: expected,
			Actual:   actual,
		})
		return
	}

	expectedType := reflect.TypeOf(expected)
	actualType := reflect.TypeOf(actual)

	// Type mismatch
	if expectedType != actualType {
		*diffs = append(*diffs, types.FieldDiff{
			Path:     pathOrRoot(path),
			Expected: expected,
			Actual:   actual,
		})
		return
	}

	switch exp := expected.(type) {
	case map[string]any:
		act := actual.(map[string]any)
		compareObjects(path, exp, act, diffs)

	case []any:
		act := actual.([]any)
		compareArrays(path, exp, act, diffs)

	default:
		// Scalar comparison
		if !reflect.DeepEqual(expected, actual) {
			*diffs = append(*diffs, types.FieldDiff{
				Path:     pathOrRoot(path),
				Expected: expected,
				Actual:   actual,
			})
		}
	}
}

// compareObjects compares two JSON objects field by field.
func compareObjects(path string, expected, actual map[string]any, diffs *[]types.FieldDiff) {
	// Check all expected fields
	for key, expectedVal := range expected {
		newPath := joinPath(path, key)
		if actualVal, exists := actual[key]; exists {
			compareValues(newPath, expectedVal, actualVal, diffs)
		} else {
			*diffs = append(*diffs, types.FieldDiff{
				Path:     newPath,
				Expected: expectedVal,
				Actual:   nil,
			})
		}
	}

	// Check for unexpected fields in actual
	for key, actualVal := range actual {
		if _, exists := expected[key]; !exists {
			newPath := joinPath(path, key)
			*diffs = append(*diffs, types.FieldDiff{
				Path:     newPath,
				Expected: nil,
				Actual:   actualVal,
			})
		}
	}
}

// compareArrays compares two JSON arrays element by element.
func compareArrays(path string, expected, actual []any, diffs *[]types.FieldDiff) {
	maxLen := max(len(expected), len(actual))

	for i := range maxLen {
		newPath := fmt.Sprintf("%s[%d]", path, i)

		if i >= len(expected) {
			*diffs = append(*diffs, types.FieldDiff{
				Path:     newPath,
				Expected: nil,
				Actual:   actual[i],
			})
		} else if i >= len(actual) {
			*diffs = append(*diffs, types.FieldDiff{
				Path:     newPath,
				Expected: expected[i],
				Actual:   nil,
			})
		} else {
			compareValues(newPath, expected[i], actual[i], diffs)
		}
	}
}

// joinPath creates a dot-separated path.
func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

// pathOrRoot returns the path or "(root)" if empty.
func pathOrRoot(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}
