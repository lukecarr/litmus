package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/spf13/cobra"

	"go.carr.sh/litmus/internal/provider"
	"go.carr.sh/litmus/internal/runner"
	"go.carr.sh/litmus/internal/types"
)

// resetFlags restores every package-level flag global to its default. The flags
// persist across runCmd executions, so each test starts from a clean slate.
func resetFlags(t *testing.T) {
	t.Helper()
	testsFile, schemaFile, prompt, promptFile = "", "", "", ""
	models = nil
	parallel = 1
	outputFormat = "terminal"
	jsonOutput = false
	providerName = "openrouter"
	apiKey, cfAccountID, cfGateway, cfToken = "", "", "", ""
	openrouterOpts = nil
	t.Cleanup(func() { openrouterOpts = nil })
}

// newCmd returns a command with captured output streams.
func newCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// completionServer replies with an OpenAI-compatible completion whose content is
// the given JSON document.
func completionServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	body := fmt.Sprintf(`{"choices":[{"index":0,"message":{"content":%q}}],"usage":{}}`, content)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// useStubProvider points the OpenRouter client at srv for the duration of a test.
func useStubProvider(srv *httptest.Server) {
	openrouterOpts = []provider.Option{
		provider.WithBaseURL(srv.URL),
		provider.WithHTTPClient(srv.Client()),
	}
}

func TestFirstNonEmpty(t *testing.T) {
	cases := []struct {
		vals []string
		want string
	}{
		{[]string{"a", "b"}, "a"},
		{[]string{"", "b"}, "b"},
		{[]string{"", ""}, ""},
		{nil, ""},
	}
	for _, tc := range cases {
		if got := firstNonEmpty(tc.vals...); got != tc.want {
			t.Errorf("firstNonEmpty(%v) = %q, want %q", tc.vals, got, tc.want)
		}
	}
}

func TestWarnUnusedFlags(t *testing.T) {
	cmd := &cobra.Command{}
	var errb bytes.Buffer
	cmd.SetErr(&errb)
	cmd.Flags().String("changed", "", "")
	cmd.Flags().String("untouched", "", "")
	if err := cmd.Flags().Set("changed", "x"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	warnUnusedFlags(cmd, "openrouter", "changed", "untouched")

	out := errb.String()
	if !contains(out, "--changed is ignored") {
		t.Errorf("expected warning for changed flag, got %q", out)
	}
	if contains(out, "untouched") {
		t.Errorf("did not expect a warning for the untouched flag, got %q", out)
	}
}

func TestBuildProvider(t *testing.T) {
	t.Run("openrouter from flag", func(t *testing.T) {
		resetFlags(t)
		apiKey = "key"
		if _, err := buildProvider(newCmd()); err != nil {
			t.Errorf("buildProvider returned error: %v", err)
		}
	})

	t.Run("openrouter from env", func(t *testing.T) {
		resetFlags(t)
		t.Setenv("OPENROUTER_API_KEY", "env-key")
		if _, err := buildProvider(newCmd()); err != nil {
			t.Errorf("buildProvider returned error: %v", err)
		}
	})

	t.Run("openrouter missing key", func(t *testing.T) {
		resetFlags(t)
		t.Setenv("OPENROUTER_API_KEY", "")
		if _, err := buildProvider(newCmd()); err == nil {
			t.Error("expected an error when no API key is supplied")
		}
	})

	t.Run("cloudflare full config", func(t *testing.T) {
		resetFlags(t)
		providerName = "cloudflare"
		cfAccountID, cfGateway, apiKey = "acct", "gw", "key"
		if _, err := buildProvider(newCmd()); err != nil {
			t.Errorf("buildProvider returned error: %v", err)
		}
	})

	cloudflareErrs := map[string]func(){
		"missing account":    func() { cfGateway, apiKey = "gw", "key" },
		"missing gateway":    func() { cfAccountID, apiKey = "acct", "key" },
		"missing credential": func() { cfAccountID, cfGateway = "acct", "gw" },
	}
	for name, setup := range cloudflareErrs {
		t.Run("cloudflare "+name, func(t *testing.T) {
			resetFlags(t)
			t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
			t.Setenv("CLOUDFLARE_GATEWAY_ID", "")
			t.Setenv("CLOUDFLARE_API_KEY", "")
			t.Setenv("CF_AIG_TOKEN", "")
			providerName = "cloudflare"
			setup()
			if _, err := buildProvider(newCmd()); err == nil {
				t.Errorf("expected an error for %s", name)
			}
		})
	}

	t.Run("openai from flag", func(t *testing.T) {
		resetFlags(t)
		providerName = "openai"
		apiKey = "key"
		if _, err := buildProvider(newCmd()); err != nil {
			t.Errorf("buildProvider returned error: %v", err)
		}
	})

	t.Run("openai from env", func(t *testing.T) {
		resetFlags(t)
		providerName = "openai"
		t.Setenv("OPENAI_API_KEY", "env-key")
		if _, err := buildProvider(newCmd()); err != nil {
			t.Errorf("buildProvider returned error: %v", err)
		}
	})

	t.Run("openai missing key", func(t *testing.T) {
		resetFlags(t)
		providerName = "openai"
		t.Setenv("OPENAI_API_KEY", "")
		if _, err := buildProvider(newCmd()); err == nil {
			t.Error("expected an error when no API key is supplied")
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		resetFlags(t)
		providerName = "nope"
		if _, err := buildProvider(newCmd()); err == nil {
			t.Error("expected an error for an unknown provider")
		}
	})
}

func TestRunTestsErrorBranches(t *testing.T) {
	good := `[{"name":"t","input":"hi","expected":{"name":"Ada"}}]`

	cases := []struct {
		name  string
		setup func(t *testing.T)
	}{
		{"provider error", func(t *testing.T) { providerName = "nope" }},
		{"prompt conflict", func(t *testing.T) {
			apiKey, prompt, promptFile = "k", "a", "b"
		}},
		{"prompt file missing", func(t *testing.T) {
			apiKey, promptFile = "k", "/no/such/prompt.txt"
		}},
		{"empty prompt", func(t *testing.T) { apiKey = "k" }},
		{"tests file missing", func(t *testing.T) {
			apiKey, prompt, testsFile = "k", "hi", "/no/such/tests.json"
		}},
		{"no tests", func(t *testing.T) {
			apiKey, prompt = "k", "hi"
			testsFile = writeFile(t, "tests.json", "[]")
		}},
		{"schema missing", func(t *testing.T) {
			apiKey, prompt = "k", "hi"
			testsFile = writeFile(t, "tests.json", good)
			schemaFile = "/no/such/schema.json"
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetFlags(t)
			tc.setup(t)
			if err := runTests(newCmd(), nil); err == nil {
				t.Errorf("expected an error for %s, got nil", tc.name)
			}
		})
	}
}

func TestRunTestsSuccess(t *testing.T) {
	resetFlags(t)
	useStubProvider(completionServer(t, `{"name":"Ada"}`))
	apiKey, prompt = "k", "hi"
	outputFormat = "json"
	models = []string{"openai/gpt-4o"}
	testsFile = writeFile(t, "tests.json", `[{"name":"t","input":"hi","expected":{"name":"Ada"}}]`)
	schemaFile = writeFile(t, "schema.json", `{"type":"object"}`)

	if err := runTests(newCmd(), nil); err != nil {
		t.Errorf("runTests returned error: %v", err)
	}
}

func TestRunTestsFailureReturnsErr(t *testing.T) {
	resetFlags(t)
	useStubProvider(completionServer(t, `{"name":"Ada"}`))
	apiKey, prompt = "k", "hi"
	outputFormat = "json"
	models = []string{"openai/gpt-4o"}
	testsFile = writeFile(t, "tests.json", `[{"name":"t","input":"hi","expected":{"name":"Bob"}}]`)
	schemaFile = writeFile(t, "schema.json", `{"type":"object"}`)

	if err := runTests(newCmd(), nil); err == nil {
		t.Error("expected ErrTestsFailed for a mismatched result, got nil")
	}
}

func TestRunTestsPromptFile(t *testing.T) {
	resetFlags(t)
	useStubProvider(completionServer(t, `{"name":"Ada"}`))
	apiKey = "k"
	promptFile = writeFile(t, "prompt.txt", "system prompt from file")
	outputFormat = "json"
	models = []string{"openai/gpt-4o"}
	testsFile = writeFile(t, "tests.json", `[{"name":"t","input":"hi","expected":{"name":"Ada"}}]`)
	schemaFile = writeFile(t, "schema.json", `{"type":"object"}`)

	if err := runTests(newCmd(), nil); err != nil {
		t.Errorf("runTests returned error: %v", err)
	}
}

func TestRunTestsReportError(t *testing.T) {
	resetFlags(t)
	useStubProvider(completionServer(t, `{"name":"Ada"}`))
	// The github reporter fails when it cannot open GITHUB_STEP_SUMMARY.
	t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "no-dir", "summary.md"))
	apiKey, prompt = "k", "hi"
	outputFormat = "github"
	models = []string{"openai/gpt-4o"}
	testsFile = writeFile(t, "tests.json", `[{"name":"t","input":"hi","expected":{"name":"Ada"}}]`)
	schemaFile = writeFile(t, "schema.json", `{"type":"object"}`)

	if err := runTests(newCmd(), nil); err == nil {
		t.Error("expected a reporter error, got nil")
	}
}

func TestRunTestsOutputFormats(t *testing.T) {
	for _, format := range []string{"terminal", "json", "html", "github"} {
		t.Run(format, func(t *testing.T) {
			resetFlags(t)
			t.Setenv("GITHUB_STEP_SUMMARY", "")
			useStubProvider(completionServer(t, `{"name":"Ada"}`))
			apiKey, prompt = "k", "hi"
			outputFormat = format
			// A blank model entry is trimmed and skipped.
			models = []string{"openai/gpt-4o", "  "}
			testsFile = writeFile(t, "tests.json", `[{"name":"t","input":"hi","expected":{"name":"Ada"}}]`)
			schemaFile = writeFile(t, "schema.json", `{"type":"object"}`)

			if err := runTests(newCmd(), nil); err != nil {
				t.Errorf("runTests(%s) returned error: %v", format, err)
			}
		})
	}
}

func TestRunTestsUnknownFormat(t *testing.T) {
	resetFlags(t)
	useStubProvider(completionServer(t, `{"name":"Ada"}`))
	apiKey, prompt = "k", "hi"
	outputFormat = "weird"
	models = []string{"openai/gpt-4o"}
	testsFile = writeFile(t, "tests.json", `[{"name":"t","input":"hi","expected":{"name":"Ada"}}]`)
	schemaFile = writeFile(t, "schema.json", `{"type":"object"}`)

	if err := runTests(newCmd(), nil); err == nil {
		t.Error("expected an error for an unknown output format, got nil")
	}
}

func TestRunTestsDeprecatedJSONFlag(t *testing.T) {
	resetFlags(t)
	useStubProvider(completionServer(t, `{"name":"Ada"}`))
	apiKey, prompt = "k", "hi"
	jsonOutput = true // deprecated alias for --output=json
	models = []string{"openai/gpt-4o"}
	testsFile = writeFile(t, "tests.json", `[{"name":"t","input":"hi","expected":{"name":"Ada"}}]`)
	schemaFile = writeFile(t, "schema.json", `{"type":"object"}`)

	if err := runTests(newCmd(), nil); err != nil {
		t.Errorf("runTests returned error: %v", err)
	}
	if outputFormat != "json" {
		t.Errorf("outputFormat = %q, want json after the deprecated flag", outputFormat)
	}
}

func TestOnInterrupt(t *testing.T) {
	ch := make(chan os.Signal, 1)
	ch <- syscall.SIGTERM
	ctx, cancel := context.WithCancel(context.Background())

	var buf bytes.Buffer
	onInterrupt(ch, cancel, &buf)

	if ctx.Err() == nil {
		t.Error("expected context to be cancelled after the signal")
	}
	if !contains(buf.String(), "Interrupted") {
		t.Errorf("expected an interrupt message, got %q", buf.String())
	}
}

// okProvider returns an empty successful completion for every call.
type okProvider struct{}

func (okProvider) Complete(ctx context.Context, model, sys, user string, schema json.RawMessage) (*provider.CompletionResult, error) {
	return &provider.CompletionResult{Response: json.RawMessage(`{}`)}, nil
}

func TestRunModelsStopsOnCancel(t *testing.T) {
	resetFlags(t)
	models = []string{"m1", "m2"}
	outputFormat = "json"

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before any model runs

	report := &types.RunReport{}
	r := runner.New(okProvider{}, 1)
	runModels(ctx, newCmd(), r, report, "p", json.RawMessage(`{}`), []types.TestCase{{Name: "t"}})

	if len(report.Models) != 1 {
		t.Errorf("ran %d models, want 1 (loop should stop after cancellation)", len(report.Models))
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
