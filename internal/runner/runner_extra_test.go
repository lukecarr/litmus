package runner

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"go.carr.sh/litmus/internal/provider"
	"go.carr.sh/litmus/internal/types"
)

// stubProvider returns a configurable response and error for every completion.
type stubProvider struct {
	resp json.RawMessage
	err  error
}

func (s stubProvider) Complete(ctx context.Context, model, systemPrompt, userInput string, schema json.RawMessage) (*provider.CompletionResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &provider.CompletionResult{Response: s.resp, Provider: "stub", TokensIn: 3, TokensOut: 2, Latency: time.Millisecond}, nil
}

func TestNewClampsParallel(t *testing.T) {
	r := New(stubProvider{}, 0)
	if r.parallel != 1 {
		t.Errorf("parallel = %d, want 1 (clamped)", r.parallel)
	}
}

func TestLoadTestFileMissing(t *testing.T) {
	if _, err := LoadTestFile("/no/such/file.json"); err == nil {
		t.Error("expected an error for a missing file, got nil")
	}
}

func TestLoadTestFileMalformedElement(t *testing.T) {
	if _, err := LoadTestFile(writeTempFile(t, `[{"name": 123}]`)); err == nil {
		t.Error("expected an error for a malformed element, got nil")
	}
}

func TestLoadTestFileEmpty(t *testing.T) {
	// An empty file fails on the very first token read.
	if _, err := LoadTestFile(writeTempFile(t, "")); err == nil {
		t.Error("expected an error for an empty file, got nil")
	}
}

func TestLoadSchema(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		schema, err := LoadSchema(writeTempFile(t, `{"type":"object"}`))
		if err != nil {
			t.Fatalf("LoadSchema returned error: %v", err)
		}
		if string(schema) != `{"type":"object"}` {
			t.Errorf("schema = %s, want the file contents", schema)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := LoadSchema("/no/such/schema.json"); err == nil {
			t.Error("expected an error for a missing file, got nil")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		if _, err := LoadSchema(writeTempFile(t, `{not json`)); err == nil {
			t.Error("expected an error for invalid JSON, got nil")
		}
	})
}

func TestRunRecordsProviderError(t *testing.T) {
	tests := []types.TestCase{{Name: "a", Input: "x", Expected: json.RawMessage(`{}`)}}
	r := New(stubProvider{err: errors.New("upstream boom")}, 1)
	run := r.Run(context.Background(), "m", "p", json.RawMessage(`{}`), tests)

	if got := run.Results[0].Error; got != "upstream boom" {
		t.Errorf("result error = %q, want upstream boom", got)
	}
	if run.Metrics.Errors != 1 {
		t.Errorf("Errors = %d, want 1", run.Metrics.Errors)
	}
}

func TestRunRecordsComparisonError(t *testing.T) {
	tests := []types.TestCase{{Name: "a", Input: "x", Expected: json.RawMessage(`{"x":1}`)}}
	r := New(stubProvider{resp: json.RawMessage(`not json`)}, 1)
	run := r.Run(context.Background(), "m", "p", json.RawMessage(`{}`), tests)

	if got := run.Results[0].Error; !strings.Contains(got, "comparison error") {
		t.Errorf("result error = %q, want a comparison error", got)
	}
}

func TestRunPassesAndFails(t *testing.T) {
	tests := []types.TestCase{
		{Name: "pass", Input: "x", Expected: json.RawMessage(`{"x":1}`)},
	}
	r := New(stubProvider{resp: json.RawMessage(`{"x":1}`)}, 1)
	run := r.Run(context.Background(), "m", "p", json.RawMessage(`{}`), tests)

	if !run.Results[0].Passed {
		t.Errorf("result should have passed: %+v", run.Results[0])
	}
	if run.Metrics.Passed != 1 || run.Metrics.Accuracy != 100 {
		t.Errorf("metrics = %+v, want 1 passed at 100%% accuracy", run.Metrics)
	}
}

func TestCalculateMetrics(t *testing.T) {
	results := []types.TestResult{
		{Passed: true, Latency: 10 * time.Millisecond, TokensIn: 5, TokensOut: 3},
		{Passed: false, Latency: 20 * time.Millisecond, TokensIn: 4, TokensOut: 2},
		{Error: "boom"},
	}
	m := calculateMetrics("m", results, time.Second)

	if m.TotalTests != 3 || m.Passed != 1 || m.Failed != 1 || m.Errors != 1 {
		t.Errorf("counts = %+v, want 3/1/1/1", m)
	}
	if m.TotalTokensIn != 9 || m.TotalTokensOut != 5 {
		t.Errorf("tokens = %d/%d, want 9/5", m.TotalTokensIn, m.TotalTokensOut)
	}
	if m.Accuracy < 33.3 || m.Accuracy > 33.4 {
		t.Errorf("Accuracy = %f, want ~33.33", m.Accuracy)
	}
	if m.Throughput != 5 {
		t.Errorf("Throughput = %f, want 5", m.Throughput)
	}
	if m.LatencyP50 <= 0 {
		t.Errorf("LatencyP50 = %v, want > 0", m.LatencyP50)
	}
}

func TestPercentile(t *testing.T) {
	if got := percentile(nil, 50); got != 0 {
		t.Errorf("percentile(nil) = %v, want 0", got)
	}
	if got := percentile([]time.Duration{42}, 95); got != 42 {
		t.Errorf("percentile(single) = %v, want 42", got)
	}

	sorted := []time.Duration{10, 20, 30, 40}
	if got := percentile(sorted, 50); got != 25 {
		t.Errorf("percentile(p50) = %v, want 25 (interpolated)", got)
	}
	if got := percentile(sorted, 100); got != 40 {
		t.Errorf("percentile(p100) = %v, want 40 (clamped to max)", got)
	}
}
