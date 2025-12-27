package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"go.carr.sh/litmus/internal/reporter"
	"go.carr.sh/litmus/internal/runner"
	"go.carr.sh/litmus/internal/types"
	"go.carr.sh/litmus/internal/util"
)

var (
	testsFile  string
	schemaFile string
	prompt     string
	promptFile string
	models     []string
	parallel   int
	jsonOutput bool
	apiKey     string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run tests against LLM models",
	Long: `Run specification tests against one or more LLM models via OpenRouter.

Examples:
  # Basic usage
  litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt --model openai/gpt-4o

  # Multiple models
  litmus run --tests tests.json --schema schema.json --prompt "Extract entities" \
    --model openai/gpt-4o --model anthropic/claude-3.5-sonnet

  # JSON output for CI/CD
  litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt \
    --model openai/gpt-4o --json

  # Parallel execution
  litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt \
    --model openai/gpt-4o --parallel 5`,
	RunE: runTests,
}

func init() {
	runCmd.Flags().StringVarP(&testsFile, "tests", "t", "", "Path to test cases JSON file (required)")
	runCmd.Flags().StringVarP(&schemaFile, "schema", "s", "", "Path to JSON schema file (required)")
	runCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "System prompt for the LLM")
	runCmd.Flags().StringVar(&promptFile, "prompt-file", "", "Path to file containing system prompt")
	runCmd.Flags().StringArrayVarP(&models, "model", "m", nil, "Model(s) to test against (required, can be repeated)")
	runCmd.Flags().IntVarP(&parallel, "parallel", "P", 1, "Number of parallel requests per model")
	runCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	runCmd.Flags().StringVar(&apiKey, "api-key", "", "OpenRouter API key (or use OPENROUTER_API_KEY env var)")

	runCmd.MarkFlagRequired("tests")
	runCmd.MarkFlagRequired("schema")
	runCmd.MarkFlagRequired("model")
}

func runTests(cmd *cobra.Command, args []string) error {
	// Get API key
	key := apiKey
	if key == "" {
		key = os.Getenv("OPENROUTER_API_KEY")
	}
	if key == "" {
		return fmt.Errorf("API key required: use --api-key or set OPENROUTER_API_KEY environment variable")
	}

	// Get prompt
	if prompt != "" && promptFile != "" {
		return fmt.Errorf("--prompt and --prompt-file are mutually exclusive")
	}

	systemPrompt := prompt
	if promptFile != "" {
		data, err := os.ReadFile(promptFile)
		if err != nil {
			return fmt.Errorf("failed to read prompt file: %w", err)
		}
		systemPrompt = string(data)
	}
	if systemPrompt == "" {
		return fmt.Errorf("prompt required: use --prompt or --prompt-file")
	}

	// Load test file
	tests, err := runner.LoadTestFile(testsFile)
	if err != nil {
		return err
	}

	if len(tests) == 0 {
		return fmt.Errorf("no tests found in %s", testsFile)
	}

	// Load schema
	schema, err := runner.LoadSchema(schemaFile)
	if err != nil {
		return err
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nInterrupted, cancelling...")
		cancel()
	}()

	// Create runner
	r := runner.New(key, parallel)

	// Prepare report
	report := &types.RunReport{
		Timestamp: time.Now(),
		Prompt:    util.Truncate(systemPrompt, 100),
		Schema:    schemaFile,
		TestFile:  testsFile,
		Models:    make([]types.ModelRun, 0, len(models)),
	}

	// Run tests for each model
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}

		if !jsonOutput {
			fmt.Fprintf(os.Stderr, "Running %d tests against %s...\n", len(tests), model)
		}

		modelRun := r.Run(ctx, model, systemPrompt, schema, tests)
		report.Models = append(report.Models, *modelRun)

		// Check for context cancellation
		if ctx.Err() != nil {
			break
		}
	}

	// Output results
	if jsonOutput {
		jsonReporter := reporter.NewJSON(os.Stdout)
		return jsonReporter.Report(report)
	}

	termReporter := reporter.NewTerminal(os.Stdout)
	if err := termReporter.Report(report); err != nil {
		return err
	}

	// Return error if any tests failed
	for _, mr := range report.Models {
		if mr.Metrics.Failed > 0 || mr.Metrics.Errors > 0 {
			cmd.SilenceUsage = true
			return ErrTestsFailed
		}
	}

	return nil
}
