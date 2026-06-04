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

	"go.carr.sh/litmus/internal/cloudflare"
	"go.carr.sh/litmus/internal/openrouter"
	"go.carr.sh/litmus/internal/provider"
	"go.carr.sh/litmus/internal/reporter"
	"go.carr.sh/litmus/internal/runner"
	"go.carr.sh/litmus/internal/types"
	"go.carr.sh/litmus/internal/util"
)

var (
	testsFile    string
	schemaFile   string
	prompt       string
	promptFile   string
	models       []string
	parallel     int
	outputFormat string
	jsonOutput   bool // Deprecated: use --output=json instead
	providerName string
	apiKey       string
	cfAccountID  string
	cfGateway    string
	cfToken      string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run tests against LLM models",
	Long: `Run specification tests against one or more LLM models via OpenRouter or
Cloudflare AI Gateway.

Providers (--provider):
  openrouter  (default)  Uses --api-key or OPENROUTER_API_KEY.
  cloudflare             Uses --cf-account-id and --cf-gateway plus a credential:
                         --api-key (the downstream provider key) and/or --cf-token
                         (the gateway token, for authenticated gateways).

Models use the same {provider}/{model} form for both providers, e.g. openai/gpt-4o.

Examples:
  # Basic usage (OpenRouter)
  litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt --model openai/gpt-4o

  # Cloudflare AI Gateway
  litmus run --provider cloudflare \
    --cf-account-id $CLOUDFLARE_ACCOUNT_ID --cf-gateway my-gateway --api-key $OPENAI_KEY \
    --tests tests.json --schema schema.json --prompt-file prompt.txt --model openai/gpt-4o

  # Multiple models
  litmus run --tests tests.json --schema schema.json --prompt "Extract entities" \
    --model openai/gpt-4o --model anthropic/claude-3.5-sonnet

  # JSON output for CI/CD
  litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt \
    --model openai/gpt-4o --output=json

  # HTML report
  litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt \
    --model openai/gpt-4o --output=html > report.html

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
	runCmd.Flags().StringVarP(&outputFormat, "output", "o", "terminal", "Output format: terminal, json, html")
	runCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON (deprecated: use --output=json)")
	runCmd.Flags().MarkDeprecated("json", "use --output=json instead")
	runCmd.Flags().StringVar(&providerName, "provider", "openrouter", "LLM provider: openrouter or cloudflare")
	runCmd.Flags().StringVar(&apiKey, "api-key", "", "API key (OpenRouter: OPENROUTER_API_KEY; Cloudflare: downstream provider key or CLOUDFLARE_API_KEY)")
	runCmd.Flags().StringVar(&cfAccountID, "cf-account-id", "", "Cloudflare account ID (or CLOUDFLARE_ACCOUNT_ID env var)")
	runCmd.Flags().StringVar(&cfGateway, "cf-gateway", "", "Cloudflare AI Gateway ID (or CLOUDFLARE_GATEWAY_ID env var)")
	runCmd.Flags().StringVar(&cfToken, "cf-token", "", "Cloudflare AI Gateway token for authenticated gateways (or CF_AIG_TOKEN env var)")

	runCmd.MarkFlagRequired("tests")
	runCmd.MarkFlagRequired("schema")
	runCmd.MarkFlagRequired("model")
}

func runTests(cmd *cobra.Command, args []string) error {
	// Handle deprecated --json flag
	if jsonOutput {
		outputFormat = "json"
	}

	// Build the provider from flags and environment.
	prov, err := buildProvider(cmd)
	if err != nil {
		return err
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
	r := runner.New(prov, parallel)

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

		if outputFormat == "terminal" {
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
	var rep reporter.Reporter
	switch outputFormat {
	case "json":
		rep = reporter.NewJSON(os.Stdout)
	case "html":
		rep = reporter.NewHTML(os.Stdout)
	case "terminal":
		rep = reporter.NewTerminal(os.Stdout)
	default:
		return fmt.Errorf("unknown output format: %s (valid: terminal, json, html)", outputFormat)
	}

	if err := rep.Report(report); err != nil {
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

// buildProvider constructs the LLM provider selected by --provider, resolving
// credentials from flags or environment variables and validating that the
// required values are present.
func buildProvider(cmd *cobra.Command) (provider.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "", "openrouter":
		warnUnusedFlags(cmd, "openrouter", "cf-account-id", "cf-gateway", "cf-token")
		key := firstNonEmpty(apiKey, os.Getenv("OPENROUTER_API_KEY"))
		if key == "" {
			return nil, fmt.Errorf("API key required: use --api-key or set OPENROUTER_API_KEY environment variable")
		}
		return openrouter.New(key), nil

	case "cloudflare":
		cfg := cloudflare.Config{
			AccountID:    firstNonEmpty(cfAccountID, os.Getenv("CLOUDFLARE_ACCOUNT_ID")),
			GatewayID:    firstNonEmpty(cfGateway, os.Getenv("CLOUDFLARE_GATEWAY_ID")),
			APIKey:       firstNonEmpty(apiKey, os.Getenv("CLOUDFLARE_API_KEY")),
			GatewayToken: firstNonEmpty(cfToken, os.Getenv("CF_AIG_TOKEN")),
		}
		if cfg.AccountID == "" {
			return nil, fmt.Errorf("cloudflare: --cf-account-id or CLOUDFLARE_ACCOUNT_ID required")
		}
		if cfg.GatewayID == "" {
			return nil, fmt.Errorf("cloudflare: --cf-gateway or CLOUDFLARE_GATEWAY_ID required")
		}
		if cfg.APIKey == "" && cfg.GatewayToken == "" {
			return nil, fmt.Errorf("cloudflare: a credential is required: use --api-key (provider key) and/or --cf-token (gateway token)")
		}
		return cloudflare.New(cfg)

	default:
		return nil, fmt.Errorf("unknown provider %q (valid: openrouter, cloudflare)", providerName)
	}
}

// warnUnusedFlags prints a warning to stderr for each named flag that was set
// but is ignored by the selected provider.
func warnUnusedFlags(cmd *cobra.Command, provider string, names ...string) {
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			fmt.Fprintf(os.Stderr, "warning: --%s is ignored with --provider %s\n", name, provider)
		}
	}
}

// firstNonEmpty returns the first non-empty string from vals, or "" if none.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
