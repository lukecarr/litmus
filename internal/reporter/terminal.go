// Package reporter provides output formatting for test results.
package reporter

import (
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"

	"go.carr.sh/litmus/internal/types"
	"go.carr.sh/litmus/internal/util"
)

const horizontalRule = "──────────────────────────────────────────────────"

// Terminal outputs results to the terminal with colors and tables.
type Terminal struct {
	// w is the writer to output the report to.
	w io.Writer
}

// NewTerminal creates a new Terminal reporter.
func NewTerminal(w io.Writer) *Terminal {
	return &Terminal{w: w}
}

// Report outputs the complete run report.
func (t *Terminal) Report(report *types.RunReport) error {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan)

	bold.Fprintf(t.w, "\nLitmus Test Report\n")
	fmt.Fprintf(t.w, "%s\n", horizontalRule)
	fmt.Fprintf(t.w, "Timestamp: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(t.w, "Test File: %s\n", report.TestFile)
	fmt.Fprintf(t.w, "Schema:    %s\n", report.Schema)
	fmt.Fprintf(t.w, "\n")

	for _, modelRun := range report.Models {
		cyan.Fprintf(t.w, "Model: %s\n", modelRun.Model)
		fmt.Fprintf(t.w, "%s\n", horizontalRule)

		// Show provider if available
		if provider := getProvider(modelRun.Results); provider != "" {
			fmt.Fprintf(t.w, "Provider: %s\n", provider)
		}

		// Summary metrics
		m := modelRun.Metrics
		fmt.Fprintf(t.w, "Results:  ")
		green.Fprintf(t.w, "%d passed", m.Passed)
		fmt.Fprintf(t.w, " / ")
		if m.Failed > 0 {
			red.Fprintf(t.w, "%d failed", m.Failed)
		} else {
			fmt.Fprintf(t.w, "%d failed", m.Failed)
		}
		if m.Errors > 0 {
			fmt.Fprintf(t.w, " / ")
			yellow.Fprintf(t.w, "%d errors", m.Errors)
		}
		fmt.Fprintf(t.w, " (")
		if m.Accuracy >= 90 {
			green.Fprintf(t.w, "%.1f%%", m.Accuracy)
		} else if m.Accuracy >= 70 {
			yellow.Fprintf(t.w, "%.1f%%", m.Accuracy)
		} else {
			red.Fprintf(t.w, "%.1f%%", m.Accuracy)
		}
		fmt.Fprintf(t.w, " accuracy)\n")

		fmt.Fprintf(t.w, "Tokens:   %d in / %d out\n", m.TotalTokensIn, m.TotalTokensOut)
		fmt.Fprintf(t.w, "Latency:  P50=%s  P95=%s  P99=%s\n",
			formatDuration(m.LatencyP50),
			formatDuration(m.LatencyP95),
			formatDuration(m.LatencyP99))
		fmt.Fprintf(t.w, "Duration: %s (%.1f tok/s)\n",
			formatDuration(m.TotalDuration),
			m.Throughput)
		fmt.Fprintf(t.w, "\n")

		// Test results table
		t.printResultsTable(modelRun.Results)
		fmt.Fprintf(t.w, "\n")

		// Show failures details
		t.printFailureDetails(modelRun.Results)
	}

	// Model comparison if multiple models
	if len(report.Models) > 1 {
		t.printComparisonTable(report.Models)
	}

	return nil
}

func (t *Terminal) printResultsTable(results []types.TestResult) {
	table := tablewriter.NewTable(t.w)
	table.Header("Test", "Status", "Latency", "Tokens")

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	for _, r := range results {
		status := green("✓ PASS")
		if r.Error != "" {
			status = yellow("⚠ ERROR")
		} else if !r.Passed {
			status = red("✗ FAIL")
		}

		name := r.TestName
		if len(name) > 40 {
			name = name[:37] + "..."
		}

		tokens := fmt.Sprintf("%d/%d", r.TokensIn, r.TokensOut)
		table.Append(name, status, formatDuration(r.Latency), tokens)
	}

	table.Render()
}

func (t *Terminal) printFailureDetails(results []types.TestResult) {
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)

	hasFailures := false
	for _, r := range results {
		if r.Error != "" || !r.Passed {
			hasFailures = true
			break
		}
	}

	if !hasFailures {
		return
	}

	fmt.Fprintf(t.w, "Failure Details:\n")
	fmt.Fprintf(t.w, "%s\n", horizontalRule)

	for _, r := range results {
		if r.Error != "" {
			yellow.Fprintf(t.w, "⚠ %s\n", r.TestName)
			fmt.Fprintf(t.w, "  Error: %s\n\n", r.Error)
		} else if !r.Passed {
			red.Fprintf(t.w, "✗ %s\n", r.TestName)
			for _, diff := range r.Diffs {
				fmt.Fprintf(t.w, "  • %s\n", diff.Path)
				fmt.Fprintf(t.w, "    Expected: %v\n", formatValue(diff.Expected))
				fmt.Fprintf(t.w, "    Actual:   %v\n", formatValue(diff.Actual))
			}
			fmt.Fprintf(t.w, "\n")
		}
	}
}

func (t *Terminal) printComparisonTable(models []types.ModelRun) {
	bold := color.New(color.Bold)
	bold.Fprintf(t.w, "Model Comparison\n")
	fmt.Fprintf(t.w, "%s\n", horizontalRule)

	table := tablewriter.NewTable(t.w)
	table.Header("Model", "Provider", "Accuracy", "P50 Latency", "Tok/s", "Tokens")

	for _, mr := range models {
		m := mr.Metrics
		table.Append(
			util.Truncate(m.Model, 30),
			getProvider(mr.Results),
			fmt.Sprintf("%.1f%%", m.Accuracy),
			formatDuration(m.LatencyP50),
			fmt.Sprintf("%.1f", m.Throughput),
			fmt.Sprintf("%d", m.TotalTokensIn+m.TotalTokensOut),
		)
	}

	table.Render()
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func formatValue(v any) string {
	if v == nil {
		return "<missing>"
	}
	s := fmt.Sprintf("%v", v)
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}

// getProvider extracts the provider from test results (returns first non-empty provider found).
func getProvider(results []types.TestResult) string {
	for _, r := range results {
		if r.Provider != "" {
			return r.Provider
		}
	}
	return ""
}
