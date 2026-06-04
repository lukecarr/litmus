package reporter

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.carr.sh/litmus/internal/types"
)

// GitHub outputs results as GitHub Actions workflow commands: an inline error
// annotation for every failed or errored test, a per-model summary line, and a
// Markdown table appended to $GITHUB_STEP_SUMMARY when running inside a job.
type GitHub struct {
	w io.Writer
}

// NewGitHub creates a new GitHub Actions reporter.
func NewGitHub(w io.Writer) *GitHub {
	return &GitHub{w: w}
}

// Report writes annotations and a per-model summary to the writer, then appends
// a Markdown table to $GITHUB_STEP_SUMMARY when that file is available.
func (g *GitHub) Report(report *types.RunReport) error {
	for _, mr := range report.Models {
		for _, r := range mr.Results {
			if msg := failureMessage(r); msg != "" {
				g.annotate(report.TestFile, r.SourceLine, mr.Model, msg)
			}
		}
		m := mr.Metrics
		fmt.Fprintf(g.w, "litmus: %s - %d passed, %d failed, %d errors\n",
			mr.Model, m.Passed, m.Failed, m.Errors)
	}

	return writeJobSummary(report)
}

// failureMessage returns the annotation body for a failed or errored result, or
// "" when the test passed.
func failureMessage(r types.TestResult) string {
	switch {
	case r.Error != "":
		return fmt.Sprintf("Test %q errored: %s", r.TestName, r.Error)
	case r.Passed:
		return ""
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "Test %q failed:", r.TestName)
		for _, d := range r.Diffs {
			fmt.Fprintf(&b, "\n%s: expected %s, got %s",
				d.Path, formatValue(d.Expected), formatValue(d.Actual))
		}
		return b.String()
	}
}

// annotate writes a single ::error workflow command for the given test.
func (g *GitHub) annotate(file string, line int, model, message string) {
	var props strings.Builder
	fmt.Fprintf(&props, "file=%s", escapeProperty(file))
	if line > 0 {
		fmt.Fprintf(&props, ",line=%d", line)
	}
	fmt.Fprintf(&props, ",title=%s", escapeProperty("litmus: "+model))
	fmt.Fprintf(g.w, "::error %s::%s\n", props.String(), escapeData(message))
}

// writeJobSummary appends a Markdown results table to $GITHUB_STEP_SUMMARY. It
// is a no-op when the variable is unset, so it does nothing outside Actions.
func writeJobSummary(report *types.RunReport) error {
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		return nil
	}

	// GITHUB_STEP_SUMMARY is set by the Actions runner. Clean the path and refuse
	// any parent-directory traversal before opening the file.
	path = filepath.Clean(path)
	if strings.Contains(path, "..") {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_STEP_SUMMARY: %w", err)
	}
	defer f.Close()

	var b strings.Builder
	b.WriteString("## Litmus results\n\n")
	b.WriteString("| Model | Passed | Failed | Errors | Accuracy |\n")
	b.WriteString("|-------|--------|--------|--------|----------|\n")
	for _, mr := range report.Models {
		m := mr.Metrics
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %.1f%% |\n",
			mr.Model, m.Passed, m.Failed, m.Errors, m.Accuracy)
	}
	b.WriteString("\n")

	if _, err := io.WriteString(f, b.String()); err != nil {
		return fmt.Errorf("failed to write GITHUB_STEP_SUMMARY: %w", err)
	}
	return nil
}

// escapeData escapes a workflow command message body.
func escapeData(s string) string {
	return strings.NewReplacer(
		"%", "%25",
		"\r", "%0D",
		"\n", "%0A",
	).Replace(s)
}

// escapeProperty escapes a workflow command property value, which additionally
// must not contain the separators ':' and ','.
func escapeProperty(s string) string {
	return strings.NewReplacer(
		"%", "%25",
		"\r", "%0D",
		"\n", "%0A",
		":", "%3A",
		",", "%2C",
	).Replace(s)
}
