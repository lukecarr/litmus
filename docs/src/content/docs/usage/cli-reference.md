---
title: CLI Reference
description: Complete reference for all Litmus CLI commands and flags.
---

This page documents all available commands and flags for the Litmus CLI.

## Basic Command

```bash
litmus run --tests <test-file> --schema <schema-file> --prompt <prompt> --model <model>
```

## Providers

Select the provider with `--provider`. The default is `openrouter`.

### OpenRouter

Set your key with `--api-key` or the `OPENROUTER_API_KEY` environment variable.

### Cloudflare AI Gateway

Pass `--provider cloudflare` with `--cf-account-id` and `--cf-gateway`. Models use the same `provider/model` names as OpenRouter.

Supply credentials in either or both of these ways:

- `--api-key` (or `CLOUDFLARE_API_KEY`) sets the downstream provider key, sent as the `Authorization` header. This is the key for the model's own provider, for example your OpenAI key.
- `--cf-token` (or `CF_AIG_TOKEN`) sets the gateway token, sent as the `cf-aig-authorization` header. It is required for authenticated gateways and is sufficient on its own when the gateway stores provider keys for you.

A single `--api-key` is sent as the upstream `Authorization` header on every request, so it only works when all the models you compare share one upstream provider. To compare models from different upstream providers in one run, store the provider keys in the gateway and authenticate with `--cf-token` alone.

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
| `--provider` | | LLM provider: `openrouter` (default) or `cloudflare` |
| `--api-key` | | Provider API key. OpenRouter: `OPENROUTER_API_KEY`. Cloudflare: the downstream provider key, or `CLOUDFLARE_API_KEY` |
| `--cf-account-id` | | Cloudflare account ID (or `CLOUDFLARE_ACCOUNT_ID`), used with `--provider cloudflare` |
| `--cf-gateway` | | Cloudflare AI Gateway ID (or `CLOUDFLARE_GATEWAY_ID`), used with `--provider cloudflare` |
| `--cf-token` | | Cloudflare AI Gateway token for authenticated gateways (or `CF_AIG_TOKEN`) |

## Examples

### Single Model

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano
```

### Cloudflare AI Gateway

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

With OpenRouter, Litmus works with any model in the [OpenRouter catalog](https://openrouter.ai/models). With Cloudflare AI Gateway, it works with any model your gateway routes to, named in the same `provider/model` form. See the [Cloudflare AI Gateway docs](https://developers.cloudflare.com/ai-gateway/) for the providers it supports.
