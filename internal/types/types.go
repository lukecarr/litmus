// Package types defines shared types for the litmus CLI tool.
package types

import (
	"encoding/json"
	"time"
)

// TestCase represents a single test case from the input file.
type TestCase struct {
	// Name of the test case.
	Name string `json:"name"`
	// Input prompt text to the test case.
	Input string `json:"input"`
	// Expected output of the test case.
	Expected json.RawMessage `json:"expected"`
}

// FieldDiff represents a difference found in a specific field.
type FieldDiff struct {
	// Path to the field that differs.
	Path string `json:"path"`
	// Expected value of the field.
	Expected any `json:"expected"`
	// Actual value of the field.
	Actual any `json:"actual"`
}

// TestResult represents the result of running a single test case.
type TestResult struct {
	// TestName is the name of the test case.
	TestName string `json:"test_name"`
	// Passed is true if the test case passed.
	Passed bool `json:"passed"`
	// Expected is the expected output of the test case.
	Expected json.RawMessage `json:"expected"`
	// Actual is the actual output of the test case.
	Actual json.RawMessage `json:"actual,omitempty"`
	// Diffs are the differences between the expected and actual output.
	Diffs []FieldDiff `json:"diffs,omitempty"`
	// Error is the error message if the test case failed.
	Error string `json:"error,omitempty"`
	// Provider is the provider of the test case.
	Provider string `json:"provider,omitempty"`
	// Latency is the latency of the test case.
	Latency time.Duration `json:"latency_ns"`
	// TokensIn is the number of tokens input to the test case.
	TokensIn int `json:"tokens_in"`
	// TokensOut is the number of tokens output from the test case.
	TokensOut int `json:"tokens_out"`
}

// ModelMetrics represents aggregated metrics for a single model.
type ModelMetrics struct {
	// Model is the name of the model.
	Model string `json:"model"`
	// TotalTests is the total number of test cases.
	TotalTests int `json:"total_tests"`
	// Passed is the number of test cases that passed.
	Passed int `json:"passed"`
	// Failed is the number of test cases that failed.
	Failed int `json:"failed"`
	// Errors is the number of test cases that errored.
	Errors int `json:"errors"`
	// Accuracy is the accuracy of the model.
	Accuracy float64 `json:"accuracy"`
	// TotalTokensIn is the total number of tokens input to the test cases.
	TotalTokensIn int `json:"total_tokens_in"`
	// TotalTokensOut is the total number of tokens output from the test cases.
	TotalTokensOut int `json:"total_tokens_out"`
	// LatencyP50 is the 50th percentile latency of the test cases.
	LatencyP50 time.Duration `json:"latency_p50_ns"`
	// LatencyP95 is the 95th percentile latency of the test cases.
	LatencyP95 time.Duration `json:"latency_p95_ns"`
	// LatencyP99 is the 99th percentile latency of the test cases.
	LatencyP99 time.Duration `json:"latency_p99_ns"`
	// TotalDuration is the total duration of the test cases.
	TotalDuration time.Duration `json:"total_duration_ns"`
	// Throughput is the throughput of the model, in tokens per second.
	Throughput float64 `json:"throughput_tps"`
}

// ModelRun represents all results from running tests against a single model.
type ModelRun struct {
	// Model is the name of the model.
	Model string `json:"model"`
	// Results are the results of the test cases.
	Results []TestResult `json:"results"`
	// Metrics are the metrics of the model.
	Metrics ModelMetrics `json:"metrics"`
}

// RunReport represents the complete output of a test run.
type RunReport struct {
	// Timestamp is the timestamp of the test run.
	Timestamp time.Time `json:"timestamp"`
	// Prompt is the prompt of the test run.
	Prompt string `json:"prompt"`
	// Schema is the schema of the test run.
	Schema string `json:"schema_file"`
	// TestFile is the test file of the test run.
	TestFile string `json:"test_file"`
	// Models are the models of the test run.
	Models []ModelRun `json:"models"`
}
