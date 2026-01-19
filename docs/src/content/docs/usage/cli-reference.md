---
title: CLI Reference
description: Complete reference for all Litmus CLI commands and flags.
---

This page documents all available commands and flags for the Litmus CLI.

## Basic Command

```bash
litmus run --tests <test-file> --schema <schema-file> --prompt <prompt> --model <model>
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--tests` | `-t` | Path to test cases JSON file (required) |
| `--schema` | `-s` | Path to JSON schema file (required) |
| `--prompt` | `-p` | System prompt for the LLM |
| `--prompt-file` | | Path to file containing system prompt |
| `--model` | `-m` | Model to test against (required, can be repeated) |
| `--parallel` | `-P` | Number of parallel requests per model (default: 1) |
| `--output` | `-o` | Output format: `terminal`, `json`, or `html` (default: `terminal`) |
| `--api-key` | | OpenRouter API key (or use OPENROUTER_API_KEY env var) |

## Examples

### Single Model

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano
```

### Multiple Models for Comparison

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt "Extract entities from the text" \
  --model openai/gpt-4.1-nano \
  --model mistralai/mistral-nemo
```

### Parallel Execution

Run tests in parallel for faster execution:

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --parallel 5
```

### JSON Output for CI/CD

Generate machine-readable JSON output:

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --output json > results.json
```

### HTML Report

Generate a self-contained HTML report:

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --output html > report.html
```

## Exit Codes

- `0`: All tests passed
- `1`: One or more tests failed or errored

## Supported Models

Litmus works with any model available on [OpenRouter](https://openrouter.ai/models).
