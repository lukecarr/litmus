# Litmus

[![CI](https://github.com/lukecarr/litmus/actions/workflows/ci.yml/badge.svg)](https://github.com/lukecarr/litmus/actions/workflows/ci.yml)
[![Release](https://github.com/lukecarr/litmus/actions/workflows/release.yml/badge.svg)](https://github.com/lukecarr/litmus/actions/workflows/release.yml)

Specification testing for structured LLM outputs.

Litmus lets you define test cases with input strings and expected JSON outputs, run them against LLM models through providers like OpenRouter, Cloudflare AI Gateway, OpenAI, Google Gemini, xAI, and Anthropic, and compare accuracy, latency, and throughput across models.

## Example output

```plain
$ litmus run --tests example/tests.json --schema example/schema.json --prompt-file example/prompt.txt --model openai/gpt-4.1-nano --model mistralai/mistral-nemo                 
Running 2 tests against openai/gpt-4.1-nano...
Running 2 tests against mistralai/mistral-nemo...

Litmus Test Report
──────────────────────────────────────────────────
Timestamp: 2025-12-27T16:19:30Z
Test File: example/tests.json
Schema:    example/schema.json

Model: openai/gpt-4.1-nano
──────────────────────────────────────────────────
Provider: OpenAI
Results:  2 passed / 0 failed (100.0% accuracy)
Tokens:   148 in / 34 out
Latency:  P50=363ms  P95=454ms  P99=462ms
Duration: 2.11s (16.1 tok/s)

┌────────────────────────┬────────┬─────────┬────────┐
│          TEST          │ STATUS │ LATENCY │ TOKENS │
├────────────────────────┼────────┼─────────┼────────┤
│ Extract person info    │ ✓ PASS │ 263ms   │ 74/17  │
│ Extract another person │ ✓ PASS │ 464ms   │ 74/17  │
└────────────────────────┴────────┴─────────┴────────┘

Model: mistralai/mistral-nemo
──────────────────────────────────────────────────
Provider: Mistral
Results:  2 passed / 0 failed (100.0% accuracy)
Tokens:   64 in / 56 out
Latency:  P50=254ms  P95=262ms  P99=263ms
Duration: 763ms (73.4 tok/s)

┌────────────────────────┬────────┬─────────┬────────┐
│          TEST          │ STATUS │ LATENCY │ TOKENS │
├────────────────────────┼────────┼─────────┼────────┤
│ Extract person info    │ ✓ PASS │ 246ms   │ 32/28  │
│ Extract another person │ ✓ PASS │ 263ms   │ 32/28  │
└────────────────────────┴────────┴─────────┴────────┘

Model Comparison
──────────────────────────────────────────────────
┌────────────────────────┬──────────┬──────────┬──────────────┬─────────┬────────┐
│         MODEL          │ PROVIDER │ ACCURACY │ P 50 LATENCY │ TOK / S │ TOKENS │
├────────────────────────┼──────────┼──────────┼──────────────┼─────────┼────────┤
│ openai/gpt-4.1-nano    │ OpenAI   │ 100.0%   │ 363ms        │ 16.1    │ 182    │
│ mistralai/mistral-nemo │ Mistral  │ 100.0%   │ 254ms        │ 73.4    │ 120    │
└────────────────────────┴──────────┴──────────┴──────────────┴─────────┴────────┘
```

## Installation

Download a pre-built binary from the [latest release](https://github.com/lukecarr/litmus/releases/latest), or install with Go:

```bash
go install go.carr.sh/litmus@latest
```

Or compile from source:

```bash
git clone https://github.com/lukecarr/litmus.git
cd litmus
go build -o litmus .
```

## Quick Start

1. Set your OpenRouter API key:

```bash
export OPENROUTER_API_KEY="your-api-key"
```

2. Create a test file (`tests.json`):

```json
[
  {
    "name": "Extract person info",
    "input": "John Smith is 30 years old and works at Acme Corp",
    "expected": {
      "name": "John Smith",
      "age": 30,
      "company": "Acme Corp"
    }
  },
  {
    "name": "Extract another person",
    "input": "Jane Doe, age 25, is employed by TechStart Inc",
    "expected": {
      "name": "Jane Doe",
      "age": 25,
      "company": "TechStart Inc"
    }
  }
]
```

3. Create a JSON schema (`schema.json`):

```json
{
  "type": "object",
  "properties": {
    "name": { "type": "string" },
    "age": { "type": "integer" },
    "company": { "type": "string" }
  },
  "required": ["name", "age", "company"],
  "additionalProperties": false
}
```

4. Create a prompt file (`prompt.txt`):

```plain
Extract the person's name, age, and company from the given text.
```

5. Run tests:

```bash
litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt --model openai/gpt-4.1-nano
```

## GitHub Action

Run Litmus in a GitHub Actions workflow. The action annotates failing tests inline on the test file and writes a results table to the job summary:

```yaml
- uses: lukecarr/litmus@v0.3.0
  with:
    tests: example/tests.json
    schema: example/schema.json
    prompt-file: example/prompt.txt
    model: openai/gpt-4.1-nano
    api-key: ${{ secrets.OPENROUTER_API_KEY }}
```

Each input maps to a `litmus run` flag, and `output` defaults to `github`. The tag you pin is the Litmus version that runs (`@v0.3.0` runs Litmus v0.3.0; a branch or SHA runs the latest release). The step exits non-zero when any test fails. See the [GitHub Actions guide](https://lukecarr.github.io/litmus/usage/github-actions/) for all inputs and Cloudflare setup.

## Usage

### Basic Command

```bash
litmus run --tests <test-file> --schema <schema-file> --prompt <prompt> --model <model>
```

### Providers

Litmus sends requests through a provider selected with `--provider`.

#### OpenRouter

The default provider. Set your key with `--api-key` or the `OPENROUTER_API_KEY` environment variable:

```bash
export OPENROUTER_API_KEY="your-api-key"

litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt --model openai/gpt-4.1-nano
```

#### OpenAI

Call the OpenAI API directly with `--provider openai`. Set your key with `--api-key` or the `OPENAI_API_KEY` environment variable. Direct providers use the bare model name, without a `provider/` prefix:

```bash
export OPENAI_API_KEY="your-api-key"

litmus run --provider openai --tests tests.json --schema schema.json --prompt-file prompt.txt --model gpt-4o
```

#### Google Gemini

Call the Gemini API directly with `--provider google` (alias `gemini`), through Google's OpenAI-compatible endpoint. Set your key with `--api-key`, `GEMINI_API_KEY`, or `GOOGLE_API_KEY`:

```bash
export GEMINI_API_KEY="your-api-key"

litmus run --provider google --tests tests.json --schema schema.json --prompt-file prompt.txt --model gemini-2.5-flash
```

#### xAI (Grok)

Call the xAI API directly with `--provider xai` (alias `grok`). Set your key with `--api-key` or the `XAI_API_KEY` environment variable:

```bash
export XAI_API_KEY="your-api-key"

litmus run --provider xai --tests tests.json --schema schema.json --prompt-file prompt.txt --model grok-4
```

#### Anthropic (Claude)

Call the Anthropic API directly with `--provider anthropic` (alias `claude`). Set your key with `--api-key` or the `ANTHROPIC_API_KEY` environment variable:

```bash
export ANTHROPIC_API_KEY="your-api-key"

litmus run --provider anthropic --tests tests.json --schema schema.json --prompt-file prompt.txt --model claude-opus-4-8
```

Anthropic uses its native Messages API rather than an OpenAI-compatible endpoint. Litmus enforces your schema by forcing a tool call whose input is the structured response.

#### Cloudflare AI Gateway

Pass `--provider cloudflare` and point Litmus at your gateway with `--cf-account-id` and `--cf-gateway`. Models use the same `provider/model` names as OpenRouter.

There are two ways to supply credentials, and you can combine them:

- A downstream provider key via `--api-key` (or `CLOUDFLARE_API_KEY`). Litmus sends it as the `Authorization` header. This is the key for the model's own provider, for example your OpenAI key.
- A gateway token via `--cf-token` (or `CF_AIG_TOKEN`). Litmus sends it as the `cf-aig-authorization` header. It is required when the gateway has authentication enabled, and it is sufficient on its own when the gateway stores provider keys for you.

```bash
export CLOUDFLARE_ACCOUNT_ID="your-account-id"
export CLOUDFLARE_GATEWAY_ID="your-gateway"

litmus run \
  --provider cloudflare \
  --api-key "$OPENAI_API_KEY" \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano
```

A single `--api-key` is sent as the upstream `Authorization` header on every request, so it only works when all the models you compare share one upstream provider. To compare models from different upstream providers in one run, store the provider keys in the gateway and authenticate with `--cf-token` alone.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--tests` | `-t` | Path to test cases JSON file (required) |
| `--schema` | `-s` | Path to JSON schema file (required) |
| `--prompt` | `-p` | System prompt for the LLM |
| `--prompt-file` | | Path to file containing system prompt |
| `--model` | `-m` | Model to test against (required, can be repeated) |
| `--parallel` | `-P` | Number of parallel requests per model (default: 1) |
| `--output` | `-o` | Output format: `terminal`, `json`, `html`, or `github` (default: `terminal`) |
| `--provider` | | LLM provider: `openrouter` (default), `cloudflare`, `openai`, `google`, `xai`, or `anthropic` |
| `--api-key` | | Provider API key. OpenRouter: `OPENROUTER_API_KEY`. Cloudflare: the downstream provider key, or `CLOUDFLARE_API_KEY`. OpenAI: `OPENAI_API_KEY`. Google: `GEMINI_API_KEY`. xAI: `XAI_API_KEY`. Anthropic: `ANTHROPIC_API_KEY` |
| `--cf-account-id` | | Cloudflare account ID (or `CLOUDFLARE_ACCOUNT_ID`), used with `--provider cloudflare` |
| `--cf-gateway` | | Cloudflare AI Gateway ID (or `CLOUDFLARE_GATEWAY_ID`), used with `--provider cloudflare` |
| `--cf-token` | | Cloudflare AI Gateway token for authenticated gateways (or `CF_AIG_TOKEN`) |

### Examples

**Single model:**

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano
```

**Multiple models for comparison:**

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt "Extract entities from the text" \
  --model openai/gpt-4.1-nano \
  --model mistralai/mistral-nemo
```

**Parallel execution:**

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --parallel 5
```

**JSON output for CI/CD:**

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --output json > results.json
```

**HTML report:**

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --output html > report.html
```

## Test File Format

The test file is a JSON array of test cases:

```json
[
  {
    "name": "Test name (for display)",
    "input": "The input text to send to the LLM",
    "expected": {
      "field1": "expected value",
      "field2": 123
    }
  }
]
```

- `name`: A human-readable name for the test case
- `input`: The user message sent to the LLM
- `expected`: The expected JSON output (must match the schema)

## JSON Schema

The schema file should be a valid [JSON Schema](https://json-schema.org/). It is passed to the provider's `response_format` parameter to enforce structured output from the LLM.

Example schema:

```json
{
  "type": "object",
  "properties": {
    "sentiment": {
      "type": "string",
      "enum": ["positive", "negative", "neutral"]
    },
    "confidence": {
      "type": "number",
      "minimum": 0,
      "maximum": 1
    }
  },
  "required": ["sentiment", "confidence"],
  "additionalProperties": false
}
```

## Output

Litmus supports four output formats via the `--output` flag:

- `terminal` (default): Colored, formatted output for the terminal
- `json`: Machine-readable JSON for CI/CD pipelines
- `html`: Self-contained HTML report for sharing and archiving
- `github`: GitHub Actions workflow commands with inline annotations and a job summary

### Terminal Output

The terminal output includes:

- Provider used for each model
- Summary metrics (pass/fail counts, accuracy %)
- Token usage and throughput (tokens/second)
- Latency percentiles (P50, P95, P99)
- Detailed test results table
- Field-level diff for failures
- Model comparison table (when testing multiple models)

### JSON Output

Use `--output json` to get machine-readable output:

```json
{
  "timestamp": "2025-12-27T16:19:30Z",
  "prompt": "Extract entities...",
  "schema_file": "schema.json",
  "test_file": "tests.json",
  "models": [
    {
      "model": "openai/gpt-4.1-nano",
      "results": [...],
      "metrics": {
        "total_tests": 10,
        "passed": 9,
        "failed": 1,
        "accuracy": 90.0,
        "latency_p50_ms": 450,
        "throughput_tps": 25.5
      }
    }
  ]
}
```

### HTML Output

Use `--output html` to generate a self-contained HTML report:

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --output html > report.html
```

The HTML report includes all the same information as the terminal output, formatted for viewing in a browser. It's self-contained with no external dependencies, making it easy to share or archive.

![HTML Report Screenshot](https://github.com/user-attachments/assets/0f2ba956-de27-42fa-9e06-42bda13412b0)

### GitHub Actions Output

Use `--output github` inside a GitHub Actions workflow:

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --output github
```

Each failed or errored test becomes an inline annotation on the test file, at the line where the test is defined, and Litmus appends a results table to the run's job summary. `litmus run` exits non-zero when any test fails, so the step fails on a regression. See [Output Formats](https://lukecarr.github.io/litmus/output/formats/) for details.

## Exit Codes

- `0`: All tests passed
- `1`: One or more tests failed or errored

## Supported Models

With OpenRouter, Litmus works with any model in the [OpenRouter catalog](https://openrouter.ai/models). With Cloudflare AI Gateway, it works with any model your gateway routes to, named in the same `provider/model` form. See the [Cloudflare AI Gateway docs](https://developers.cloudflare.com/ai-gateway/) for the providers it supports.

## License

Litmus is licensed under the [MIT License](LICENSE).
