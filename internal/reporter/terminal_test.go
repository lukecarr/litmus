package reporter

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"

	"go.carr.sh/litmus/internal/types"
)

const longValue = "a value that is definitely longer than sixty characters so truncation kicks in"

// terminalReport builds a three-model report exercising the passing, failing,
// and errored paths plus each accuracy color band and latency unit.
func terminalReport() *types.RunReport {
	return &types.RunReport{
		Timestamp: time.Unix(0, 0).UTC(),
		TestFile:  "tests.json",
		Schema:    "schema.json",
		Models: []types.ModelRun{
			{
				Model: "openai/gpt-4o",
				Results: []types.TestResult{
					{TestName: "fast", Passed: true, Provider: "openai", Latency: 500 * time.Microsecond},
					{TestName: "medium", Passed: true, Latency: 50 * time.Millisecond},
					{TestName: "slow", Passed: true, Latency: 2 * time.Second},
				},
				Metrics: types.ModelMetrics{
					Model: "openai/gpt-4o", TotalTests: 3, Passed: 3, Accuracy: 100,
					TotalTokensIn: 30, TotalTokensOut: 12, TotalDuration: 3 * time.Second,
					LatencyP50: 50 * time.Millisecond, Throughput: 4,
				},
			},
			{
				Model: "anthropic/claude",
				Results: []types.TestResult{
					{
						TestName: "this is a very long test name that exceeds forty characters total",
						Passed:   false,
						Diffs: []types.FieldDiff{
							{Path: "name", Expected: nil, Actual: longValue},
							{Path: "age", Expected: 25, Actual: 24},
						},
					},
					{TestName: "boom", Error: "timeout"},
				},
				Metrics: types.ModelMetrics{
					Model: "anthropic/claude", TotalTests: 4, Passed: 3, Failed: 1, Errors: 1, Accuracy: 75,
				},
			},
			{
				Model:   "google/gemini",
				Results: []types.TestResult{{TestName: "fail", Passed: false}},
				Metrics: types.ModelMetrics{Model: "google/gemini", TotalTests: 2, Passed: 1, Failed: 1, Accuracy: 50},
			},
		},
	}
}

func TestTerminalReport(t *testing.T) {
	color.NoColor = true

	var buf bytes.Buffer
	if err := NewTerminal(&buf).Report(terminalReport()); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	out := buf.String()

	checks := []string{
		"Litmus Test Report",
		"Model: openai/gpt-4o",
		"Provider: openai",
		"3 passed",
		"100.0%",  // green accuracy band
		"75.0%",   // yellow accuracy band
		"50.0%",   // red accuracy band
		"500µs",   // sub-millisecond latency
		"50ms",    // sub-second latency
		"2.00s",   // second-scale latency
		"⚠ ERROR", // errored result status
		"✗ FAIL",  // failed result status
		"✓ PASS",  // passing result status
		"Failure Details:",
		"timeout",          // error detail
		"<missing>",        // nil diff value via formatValue
		"Model Comparison", // multi-model comparison table
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	// A long failure value is truncated by formatValue (57 chars + "...").
	if !strings.Contains(out, longValue[:57]+"...") {
		t.Errorf("expected truncated long value in:\n%s", out)
	}
	// A long test name is truncated to 37 chars + "..." in the results table.
	if !strings.Contains(out, "this is a very long test name that ex...") {
		t.Errorf("expected truncated test name in:\n%s", out)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		500 * time.Microsecond: "500µs",
		50 * time.Millisecond:  "50ms",
		2 * time.Second:        "2.00s",
	}
	for d, want := range cases {
		if got := formatDuration(d); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestGetProvider(t *testing.T) {
	if got := getProvider(nil); got != "" {
		t.Errorf("getProvider(nil) = %q, want empty", got)
	}
	results := []types.TestResult{{}, {Provider: "openai"}}
	if got := getProvider(results); got != "openai" {
		t.Errorf("getProvider = %q, want openai", got)
	}
}
