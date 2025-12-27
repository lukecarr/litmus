# Litmus

Specification testing for structured LLM outputs.

Litmus lets you define test cases with input strings and expected JSON outputs, run them against LLM models via OpenRouter, and compare accuracy, latency, throughput, and cost across models.

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
litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt --model openai/gpt-4o
```

## Usage

### Basic Command

```bash
litmus run --tests <test-file> --schema <schema-file> --prompt <prompt> --model <model>
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--tests` | `-t` | Path to test cases JSON file (required) |
| `--schema` | `-s` | Path to JSON schema file (required) |
| `--prompt` | `-p` | System prompt for the LLM |
| `--prompt-file` | | Path to file containing system prompt |
| `--model` | `-m` | Model to test against (required, can be repeated) |
| `--parallel` | `-P` | Number of parallel requests per model (default: 1) |
| `--json` | | Output results as JSON |
| `--api-key` | | OpenRouter API key (or use OPENROUTER_API_KEY env var) |

### Examples

**Single model:**

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4o
```

**Multiple models for comparison:**

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt "Extract entities from the text" \
  --model openai/gpt-4o \
  --model anthropic/claude-3.5-sonnet \
  --model google/gemini-pro
```

**Parallel execution:**

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4o \
  --parallel 5
```

**JSON output for CI/CD:**

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4o \
  --json > results.json
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

The schema file should be a valid JSON Schema (draft-07 or later). It is passed to OpenRouter's `response_format` parameter to enforce structured output from the LLM.

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

Use `--json` to get machine-readable output:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "prompt": "Extract entities...",
  "schema_file": "schema.json",
  "test_file": "tests.json",
  "models": [
    {
      "model": "openai/gpt-4o",
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

## Exit Codes

- `0`: All tests passed
- `1`: One or more tests failed or errored

## Supported Models

Litmus works with any model available on [OpenRouter](https://openrouter.ai/models).

## License

Litmus is licensed under the [MIT License](LICENSE).
