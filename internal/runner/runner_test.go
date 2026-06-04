package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.carr.sh/litmus/internal/provider"
	"go.carr.sh/litmus/internal/types"
)

// writeTempFile writes content to a temp file and returns its path.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tests.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func TestLoadTestFileTracksSourceLine(t *testing.T) {
	// Each object's opening brace sits on a known line; SourceLine should
	// report that line (1-based).
	content := `[
    {
        "name": "first",
        "input": "a",
        "expected": {"x": 1}
    },
    {
        "name": "second",
        "input": "b",
        "expected": {"x": 2}
    }
]`
	tests, err := LoadTestFile(writeTempFile(t, content))
	if err != nil {
		t.Fatalf("LoadTestFile returned error: %v", err)
	}
	if len(tests) != 2 {
		t.Fatalf("loaded %d tests, want 2", len(tests))
	}
	if tests[0].SourceLine != 2 {
		t.Errorf("tests[0].SourceLine = %d, want 2", tests[0].SourceLine)
	}
	if tests[1].SourceLine != 7 {
		t.Errorf("tests[1].SourceLine = %d, want 7", tests[1].SourceLine)
	}
}

func TestLoadTestFileCompactArray(t *testing.T) {
	// A single-line array still resolves every object to line 1.
	content := `[{"name":"a","input":"i","expected":{}},{"name":"b","input":"j","expected":{}}]`
	tests, err := LoadTestFile(writeTempFile(t, content))
	if err != nil {
		t.Fatalf("LoadTestFile returned error: %v", err)
	}
	for i, tc := range tests {
		if tc.SourceLine != 1 {
			t.Errorf("tests[%d].SourceLine = %d, want 1", i, tc.SourceLine)
		}
	}
}

func TestLoadTestFileRejectsNonArray(t *testing.T) {
	if _, err := LoadTestFile(writeTempFile(t, `{"name":"a"}`)); err == nil {
		t.Fatal("expected an error for a non-array test file, got nil")
	}
}

// fakeProvider returns a fixed response for every completion.
type fakeProvider struct {
	resp json.RawMessage
}

func (f fakeProvider) Complete(ctx context.Context, model, systemPrompt, userInput string, schema json.RawMessage) (*provider.CompletionResult, error) {
	return &provider.CompletionResult{Response: f.resp, Provider: "fake"}, nil
}

func TestRunPropagatesSourceLine(t *testing.T) {
	tests := []types.TestCase{
		{Name: "first", Input: "a", Expected: json.RawMessage(`{"x":1}`), SourceLine: 2},
		{Name: "second", Input: "b", Expected: json.RawMessage(`{"x":1}`), SourceLine: 7},
	}
	r := New(fakeProvider{resp: json.RawMessage(`{"x":1}`)}, 1)
	run := r.Run(context.Background(), "m", "prompt", json.RawMessage(`{}`), tests)

	if len(run.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(run.Results))
	}
	if run.Results[0].SourceLine != 2 {
		t.Errorf("Results[0].SourceLine = %d, want 2", run.Results[0].SourceLine)
	}
	if run.Results[1].SourceLine != 7 {
		t.Errorf("Results[1].SourceLine = %d, want 7", run.Results[1].SourceLine)
	}
}
