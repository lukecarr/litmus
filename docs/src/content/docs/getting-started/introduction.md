---
title: Introduction
description: Learn about Litmus - specification testing for structured LLM outputs.
---

Litmus is a command-line tool for specification testing of structured LLM outputs. It lets you define test cases with input strings and expected JSON outputs, run them against LLM models via OpenRouter, and compare accuracy, latency, and throughput across models.

## What is Litmus?

Litmus helps you:

- **Define test cases** with input strings and expected JSON outputs
- **Run tests** against LLM models via OpenRouter
- **Compare models** by accuracy, latency, and throughput
- **Generate reports** in terminal, JSON, or HTML format

## Why use Litmus?

When working with LLMs for structured data extraction, you need to:

1. Validate that the model produces correct outputs
2. Compare different models to find the best fit for your use case
3. Monitor latency and throughput for production readiness
4. Automate testing in CI/CD pipelines

Litmus makes all of this easy with a simple CLI interface.

## Example Output

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

Model Comparison
──────────────────────────────────────────────────
┌────────────────────────┬──────────┬──────────┬──────────────┬─────────┬────────┐
│         MODEL          │ PROVIDER │ ACCURACY │ P 50 LATENCY │ TOK / S │ TOKENS │
├────────────────────────┼──────────┼──────────┼──────────────┼─────────┼────────┤
│ openai/gpt-4.1-nano    │ OpenAI   │ 100.0%   │ 363ms        │ 16.1    │ 182    │
│ mistralai/mistral-nemo │ Mistral  │ 100.0%   │ 254ms        │ 73.4    │ 120    │
└────────────────────────┴──────────┴──────────┴──────────────┴─────────┴────────┘
```

## Next Steps

- [Installation](/getting-started/installation/) - Install Litmus on your system
- [Quick Start](/getting-started/quick-start/) - Run your first test
