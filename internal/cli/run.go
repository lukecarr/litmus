package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"go.carr.sh/litmus/internal/cloudflare"
	"go.carr.sh/litmus/internal/openai"
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

// openrouterOpts lets tests point the OpenRouter client at a stub server. It is
// empty in normal operation, so the production defaults apply unchanged.
var openrouterOpts []provider.Option

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
  openai                 Direct OpenAI API. Uses --api-key or OPENAI_API_KEY.

The gateway providers (openrouter, cloudflare) name models in {provider}/{model}
form, e.g. openai/gpt-4o. Direct providers use the bare model name, e.g. gpt-4o.

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

  # GitHub Actions: inline annotations on the test file + a job summary
  litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt \
    --model openai/gpt-4o --output=github

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
	runCmd.Flags().StringVarP(&outputFormat, "output", "o", "terminal", "Output format: terminal, json, html, github")
	runCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON (deprecated: use --output=json)")
	runCmd.Flags().MarkDeprecated("json", "use --output=json instead")
	runCmd.Flags().StringVar(&providerName, "provider", "openrouter", "LLM provider: openrouter, cloudflare, or openai")
	runCmd.Flags().StringVar(&apiKey, "api-key", "", "API key (OpenRouter: OPENROUTER_API_KEY; Cloudflare: downstream provider key or CLOUDFLARE_API_KEY; OpenAI: OPENAI_API_KEY)")
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
	go onInterrupt(sigCh, cancel, cmd.ErrOrStderr())

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

	runModels(ctx, cmd, r, report, systemPrompt, schema, tests)

	// Output results
	var rep reporter.Reporter
	out := cmd.OutOrStdout()
	switch outputFormat {
	case "json":
		rep = reporter.NewJSON(out)
	case "html":
		rep = reporter.NewHTML(out)
	case "github":
		rep = reporter.NewGitHub(out)
	case "terminal":
		rep = reporter.NewTerminal(out)
	default:
		return fmt.Errorf("unknown output format: %s (valid: terminal, json, html, github)", outputFormat)
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

// onInterrupt waits for the first signal on sigCh, reports it, and cancels the
// run. It is split out so the cancellation path is testable without delivering a
// real OS signal.
func onInterrupt(sigCh <-chan os.Signal, cancel context.CancelFunc, w io.Writer) {
	<-sigCh
	fmt.Fprintln(w, "\nInterrupted, cancelling...")
	cancel()
}

// runModels runs the tests against each configured model, appending a ModelRun
// per model. It skips blank model entries and stops early once ctx is cancelled.
func runModels(ctx context.Context, cmd *cobra.Command, r *runner.Runner, report *types.RunReport, systemPrompt string, schema json.RawMessage, tests []types.TestCase) {
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}

		if outputFormat == "terminal" {
			fmt.Fprintf(cmd.ErrOrStderr(), "Running %d tests against %s...\n", len(tests), model)
		}

		modelRun := r.Run(ctx, model, systemPrompt, schema, tests)
		report.Models = append(report.Models, *modelRun)

		// Stop scheduling further models once the run is cancelled.
		if ctx.Err() != nil {
			break
		}
	}
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
		return openrouter.New(key, openrouterOpts...), nil

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

	case "openai":
		warnUnusedFlags(cmd, "openai", "cf-account-id", "cf-gateway", "cf-token")
		key := firstNonEmpty(apiKey, os.Getenv("OPENAI_API_KEY"))
		if key == "" {
			return nil, fmt.Errorf("API key required: use --api-key or set OPENAI_API_KEY environment variable")
		}
		return openai.New(key), nil

	default:
		return nil, fmt.Errorf("unknown provider %q (valid: openrouter, cloudflare, openai)", providerName)
	}
}

// warnUnusedFlags prints a warning to stderr for each named flag that was set
// but is ignored by the selected provider.
func warnUnusedFlags(cmd *cobra.Command, provider string, names ...string) {
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: --%s is ignored with --provider %s\n", name, provider)
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
