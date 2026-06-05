package compare

import (
	"encoding/json"
	"testing"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		name      string
		expected  string
		actual    string
		wantDiffs int
		// wantPath, when set, is checked against the first diff's path.
		wantPath string
	}{
		{name: "equal scalars", expected: `1`, actual: `1`, wantDiffs: 0},
		{name: "equal objects", expected: `{"a":1,"b":"x"}`, actual: `{"a":1,"b":"x"}`, wantDiffs: 0},
		{name: "equal arrays", expected: `[1,2,3]`, actual: `[1,2,3]`, wantDiffs: 0},
		{name: "both null", expected: `null`, actual: `null`, wantDiffs: 0},

		{name: "root scalar mismatch", expected: `1`, actual: `2`, wantDiffs: 1, wantPath: "(root)"},
		{name: "expected null actual value", expected: `null`, actual: `1`, wantDiffs: 1, wantPath: "(root)"},
		{name: "expected value actual null", expected: `1`, actual: `null`, wantDiffs: 1, wantPath: "(root)"},
		{name: "type mismatch", expected: `1`, actual: `"x"`, wantDiffs: 1, wantPath: "(root)"},

		{name: "missing expected key", expected: `{"a":{"b":1}}`, actual: `{"a":{}}`, wantDiffs: 1, wantPath: "a.b"},
		{name: "extra actual key", expected: `{}`, actual: `{"x":1}`, wantDiffs: 1, wantPath: "x"},
		{name: "nested value diff", expected: `{"a":{"b":1}}`, actual: `{"a":{"b":2}}`, wantDiffs: 1, wantPath: "a.b"},

		{name: "array expected longer", expected: `[1,2]`, actual: `[1]`, wantDiffs: 1, wantPath: "[1]"},
		{name: "array actual longer", expected: `[1]`, actual: `[1,2]`, wantDiffs: 1, wantPath: "[1]"},
		{name: "array element diff", expected: `[1]`, actual: `[2]`, wantDiffs: 1, wantPath: "[0]"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diffs, err := Compare(json.RawMessage(tc.expected), json.RawMessage(tc.actual))
			if err != nil {
				t.Fatalf("Compare returned error: %v", err)
			}
			if len(diffs) != tc.wantDiffs {
				t.Fatalf("got %d diffs, want %d: %+v", len(diffs), tc.wantDiffs, diffs)
			}
			if tc.wantPath != "" && diffs[0].Path != tc.wantPath {
				t.Errorf("diff path = %q, want %q", diffs[0].Path, tc.wantPath)
			}
		})
	}
}

func TestCompareInvalidJSON(t *testing.T) {
	if _, err := Compare(json.RawMessage(`{`), json.RawMessage(`{}`)); err == nil {
		t.Error("expected error for invalid expected JSON, got nil")
	}
	if _, err := Compare(json.RawMessage(`{}`), json.RawMessage(`{`)); err == nil {
		t.Error("expected error for invalid actual JSON, got nil")
	}
}
