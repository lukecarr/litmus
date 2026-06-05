package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.carr.sh/litmus/internal/types"
)

// htmlReport covers all three accuracy bands and a failed test whose diff holds
// both a valid value and an invalid raw message (to exercise the json func).
func htmlReport() *types.RunReport {
	return &types.RunReport{
		TestFile: "tests.json",
		Models: []types.ModelRun{
			{
				Model:   "openai/gpt-4o",
				Metrics: types.ModelMetrics{Model: "openai/gpt-4o", Accuracy: 95},
			},
			{
				Model:   "anthropic/claude",
				Metrics: types.ModelMetrics{Model: "anthropic/claude", Accuracy: 80},
			},
			{
				Model: "google/gemini",
				Results: []types.TestResult{
					{TestName: "failing", Passed: false, Diffs: []types.FieldDiff{
						{Path: "age", Expected: 25, Actual: 24},
						{Path: "bad", Expected: json.RawMessage("not-json"), Actual: nil},
					}},
				},
				Metrics: types.ModelMetrics{Model: "google/gemini", Accuracy: 30, Failed: 1},
			},
		},
	}
}

func TestHTMLReport(t *testing.T) {
	var buf bytes.Buffer
	if err := NewHTML(&buf).Report(htmlReport()); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "openai/gpt-4o") {
		t.Errorf("HTML output missing model name:\n%s", out)
	}
	if !strings.Contains(out, "<html") && !strings.Contains(out, "<!DOCTYPE") {
		t.Errorf("output does not look like HTML")
	}
	// The invalid raw message routes through the json func's error branch.
	if !strings.Contains(out, "error marshaling JSON") {
		t.Errorf("expected a JSON marshaling error in output:\n%s", out)
	}
}

func TestHTMLReportEmptyModels(t *testing.T) {
	var buf bytes.Buffer
	if err := NewHTML(&buf).Report(&types.RunReport{TestFile: "tests.json"}); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty HTML output for an empty report")
	}
}

func TestHTMLReportWriteError(t *testing.T) {
	if err := NewHTML(failingWriter{}).Report(htmlReport()); err == nil {
		t.Fatal("expected an error from a failing writer, got nil")
	}
}
