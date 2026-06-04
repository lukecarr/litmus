// Package runner orchestrates test execution against LLM models.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"go.carr.sh/litmus/internal/compare"
	"go.carr.sh/litmus/internal/provider"
	"go.carr.sh/litmus/internal/types"
)

// Runner executes tests against LLM models.
type Runner struct {
	// client is the LLM provider.
	client provider.Provider
	// parallel is the number of parallel requests per model.
	parallel int
}

// New creates a new Runner backed by the given provider.
func New(p provider.Provider, parallel int) *Runner {
	if parallel < 1 {
		parallel = 1
	}
	return &Runner{
		client:   p,
		parallel: parallel,
	}
}

// LoadTestFile loads test cases from a JSON file. It decodes the array element
// by element so each case can record the source line it begins on, which the
// reporters use to point failures back at the test file.
func LoadTestFile(path string) ([]types.TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test file: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to parse test file: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return nil, fmt.Errorf("failed to parse test file: expected a JSON array of test cases")
	}

	var tests []types.TestCase
	line, counted := 1, 0 // running 1-based line, and bytes already scanned for newlines
	for dec.More() {
		start := int(dec.InputOffset())
		var tc types.TestCase
		if err := dec.Decode(&tc); err != nil {
			return nil, fmt.Errorf("failed to parse test file: %w", err)
		}

		// Find this element's opening brace, counting newlines as we advance.
		// Braces are reached in order, so counted only moves forward: O(n) overall.
		brace := start
		for brace < len(data) && data[brace] != '{' {
			brace++
		}
		for ; counted < brace; counted++ {
			if data[counted] == '\n' {
				line++
			}
		}

		tc.SourceLine = line
		tests = append(tests, tc)
	}

	// Reject anything after the array (a truncated or concatenated file), which
	// the previous json.Unmarshal rejected. dec.Token consumes the closing ']'.
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("failed to parse test file: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("failed to parse test file: unexpected data after the test array")
	}

	return tests, nil
}

// LoadSchema loads a JSON schema from a file.
func LoadSchema(path string) (json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Validate it's valid JSON
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON in schema file: %w", err)
	}

	return json.RawMessage(data), nil
}

// Run executes all test cases against a model and returns results.
func (r *Runner) Run(ctx context.Context, model, prompt string, schema json.RawMessage, tests []types.TestCase) *types.ModelRun {
	results := make([]types.TestResult, len(tests))
	startTime := time.Now()

	// Create a semaphore for parallel execution
	sem := make(chan struct{}, r.parallel)
	var wg sync.WaitGroup

	for i, tc := range tests {
		wg.Add(1)
		go func(idx int, test types.TestCase) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			results[idx] = r.runSingleTest(ctx, model, prompt, schema, test)
		}(i, tc)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	metrics := calculateMetrics(model, results, totalDuration)

	return &types.ModelRun{
		Model:   model,
		Results: results,
		Metrics: metrics,
	}
}

// runSingleTest executes a single test case.
func (r *Runner) runSingleTest(ctx context.Context, model, prompt string, schema json.RawMessage, test types.TestCase) types.TestResult {
	result := types.TestResult{
		TestName:   test.Name,
		SourceLine: test.SourceLine,
		Expected:   test.Expected,
	}

	completion, err := r.client.Complete(ctx, model, prompt, test.Input, schema)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Actual = completion.Response
	result.Provider = completion.Provider
	result.Latency = completion.Latency
	result.TokensIn = completion.TokensIn
	result.TokensOut = completion.TokensOut

	// Compare expected vs actual
	diffs, err := compare.Compare(test.Expected, completion.Response)
	if err != nil {
		result.Error = fmt.Sprintf("comparison error: %v", err)
		return result
	}

	result.Diffs = diffs
	result.Passed = len(diffs) == 0

	return result
}

// calculateMetrics computes aggregated metrics from test results.
func calculateMetrics(model string, results []types.TestResult, totalDuration time.Duration) types.ModelMetrics {
	metrics := types.ModelMetrics{
		Model:         model,
		TotalTests:    len(results),
		TotalDuration: totalDuration,
	}

	var latencies []time.Duration

	for _, r := range results {
		if r.Error != "" {
			metrics.Errors++
		} else if r.Passed {
			metrics.Passed++
		} else {
			metrics.Failed++
		}

		metrics.TotalTokensIn += r.TokensIn
		metrics.TotalTokensOut += r.TokensOut

		if r.Latency > 0 {
			latencies = append(latencies, r.Latency)
		}
	}

	if metrics.TotalTests > 0 {
		metrics.Accuracy = float64(metrics.Passed) / float64(metrics.TotalTests) * 100
	}

	if totalDuration > 0 {
		metrics.Throughput = float64(metrics.TotalTokensOut) / totalDuration.Seconds()
	}

	// Calculate latency percentiles
	if len(latencies) > 0 {
		slices.Sort(latencies)

		metrics.LatencyP50 = percentile(latencies, 50)
		metrics.LatencyP95 = percentile(latencies, 95)
		metrics.LatencyP99 = percentile(latencies, 99)
	}

	return metrics
}

// percentile calculates the p-th percentile of a sorted slice.
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	idx := float64(p) / 100.0 * float64(len(sorted)-1)
	lower := int(idx)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	weight := idx - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}
