package reporter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.carr.sh/litmus/internal/types"
)

// sampleReport builds a report with one passing, one failing, and one errored
// test against a single model.
func sampleReport() *types.RunReport {
	return &types.RunReport{
		TestFile: "example/tests.json",
		Models: []types.ModelRun{
			{
				Model: "openai/gpt-4o",
				Results: []types.TestResult{
					{TestName: "passing", SourceLine: 2, Passed: true},
					{TestName: "failing", SourceLine: 7, Passed: false, Diffs: []types.FieldDiff{
						{Path: "age", Expected: 25, Actual: 24},
					}},
					{TestName: "erroring", SourceLine: 12, Error: "boom: timeout"},
				},
				Metrics: types.ModelMetrics{
					Model: "openai/gpt-4o", TotalTests: 3,
					Passed: 1, Failed: 1, Errors: 1, Accuracy: 33.3,
				},
			},
		},
	}
}

func TestGitHubReporterAnnotations(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "") // no job summary in this test

	var buf bytes.Buffer
	if err := NewGitHub(&buf).Report(sampleReport()); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	out := buf.String()

	// Failing test: error annotation at the right file and line, with the diff
	// rendered as a (URL-encoded) multi-line message.
	if !strings.Contains(out, "::error file=example/tests.json,line=7,title=litmus%3A openai/gpt-4o::") {
		t.Errorf("missing failure annotation header in:\n%s", out)
	}
	if !strings.Contains(out, `Test "failing" failed:`) {
		t.Errorf("missing failure message in:\n%s", out)
	}
	if !strings.Contains(out, "age: expected 25, got 24") {
		t.Errorf("missing diff detail in:\n%s", out)
	}
	if !strings.Contains(out, "%0A") {
		t.Errorf("expected encoded newline (%%0A) in multi-line message:\n%s", out)
	}

	// Errored test: error annotation at its line.
	if !strings.Contains(out, "::error file=example/tests.json,line=12,") {
		t.Errorf("missing error annotation header in:\n%s", out)
	}
	if !strings.Contains(out, `Test "erroring" errored: boom: timeout`) {
		t.Errorf("missing error message in:\n%s", out)
	}

	// Passing test must not produce an annotation.
	if strings.Contains(out, `Test "passing"`) {
		t.Errorf("passing test should not be annotated:\n%s", out)
	}

	// Per-model summary line.
	if !strings.Contains(out, "litmus: openai/gpt-4o - 1 passed, 1 failed, 1 errors") {
		t.Errorf("missing per-model summary line in:\n%s", out)
	}
}

func TestGitHubReporterEscapesMessage(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "")

	report := &types.RunReport{
		TestFile: "tests.json",
		Models: []types.ModelRun{{
			Model:   "m",
			Results: []types.TestResult{{TestName: "x", SourceLine: 1, Error: "line1\nline2 100%"}},
			Metrics: types.ModelMetrics{Errors: 1},
		}},
	}

	var buf bytes.Buffer
	if err := NewGitHub(&buf).Report(report); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "line1%0Aline2 100%25") {
		t.Errorf("message not escaped (want %%0A and %%25):\n%s", out)
	}
	if strings.Contains(out, "line1\nline2") {
		t.Errorf("raw newline leaked into annotation:\n%s", out)
	}
}

func TestGitHubReporterOmitsLineWhenUnknown(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "")

	report := &types.RunReport{
		TestFile: "tests.json",
		Models: []types.ModelRun{{
			Model:   "m",
			Results: []types.TestResult{{TestName: "x", SourceLine: 0, Error: "boom"}},
			Metrics: types.ModelMetrics{Errors: 1},
		}},
	}

	var buf bytes.Buffer
	if err := NewGitHub(&buf).Report(report); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	if strings.Contains(buf.String(), "line=") {
		t.Errorf("should omit line= when SourceLine is 0:\n%s", buf.String())
	}
}

func TestGitHubReporterJobSummary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "summary.md")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("failed to create summary file: %v", err)
	}
	t.Setenv("GITHUB_STEP_SUMMARY", path)

	if err := NewGitHub(&bytes.Buffer{}).Report(sampleReport()); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read summary file: %v", err)
	}
	summary := string(data)
	if !strings.Contains(summary, "## Litmus results") {
		t.Errorf("missing summary heading:\n%s", summary)
	}
	if !strings.Contains(summary, "| openai/gpt-4o | 1 | 1 | 1 |") {
		t.Errorf("missing summary row:\n%s", summary)
	}
}

func TestGitHubReporterJobSummaryIgnoresTraversalPath(t *testing.T) {
	// A GITHUB_STEP_SUMMARY containing parent-directory traversal is rejected, so
	// Report neither errors nor writes outside the runner's summary file.
	t.Setenv("GITHUB_STEP_SUMMARY", "../../litmus-should-not-write.md")
	if err := NewGitHub(&bytes.Buffer{}).Report(sampleReport()); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	if _, err := os.Stat("../../litmus-should-not-write.md"); err == nil {
		t.Fatal("Report wrote to a traversal path; the guard did not hold")
	}
}
