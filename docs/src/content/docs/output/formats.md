---
title: Output Formats
description: Understanding the different output formats supported by Litmus.
---

Litmus supports four output formats via the `--output` flag:

- `terminal` (default): Colored, formatted output for the terminal
- `json`: Machine-readable JSON for CI/CD pipelines
- `html`: Self-contained HTML report for sharing and archiving
- `github`: GitHub Actions workflow commands with inline annotations and a job summary

## Terminal Output

The default terminal output includes:

- Provider used for each model
- Summary metrics (pass/fail counts, accuracy %)
- Token usage and throughput (tokens/second)
- Latency percentiles (P50, P95, P99)
- Detailed test results table
- Field-level diff for failures
- Model comparison table (when testing multiple models)

Example:

```plain
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
```

## JSON Output

Use `--output json` to get machine-readable output for CI/CD pipelines:

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --output json > results.json
```

### JSON Schema

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

## HTML Output

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

### Features

- Self-contained with no external dependencies
- Responsive design for desktop and mobile
- Collapsible sections for detailed results
- Color-coded pass/fail indicators
- Interactive model comparison

## GitHub Actions Output

Use `--output github` when you run Litmus inside a GitHub Actions workflow:

```bash
litmus run \
  --tests tests.json \
  --schema schema.json \
  --prompt-file prompt.txt \
  --model openai/gpt-4.1-nano \
  --output github
```

For each failed or errored test, Litmus prints a [workflow command](https://docs.github.com/en/actions/reference/workflow-commands-for-github-actions) that GitHub turns into an inline annotation on the test file, at the line where the test is defined:

```plain
::error file=tests.json,line=11,title=litmus%3A openai/gpt-4.1-nano::Test "Extract another person" failed:%0Aage: expected 25, got 24
```

Special characters in the message are URL-encoded, so newlines appear as `%0A`. GitHub decodes them when it renders the annotation.

The `file=` value is the path you passed to `--tests`. GitHub only attaches the inline annotation to the diff when that path is relative to the repository root, so run Litmus from the repo root and pass a repo-relative path. An absolute or subdirectory-relative path still appears in the log but will not show up on the changed files.

GitHub also limits how many annotations it surfaces per step (10 of each level), so a run with many failures will not show every annotation inline. The job summary below lists every model's totals, so use it to see the full picture.

When `$GITHUB_STEP_SUMMARY` is set, which is the case inside any job, Litmus also appends a Markdown table to the run's summary:

| Model | Passed | Failed | Errors | Accuracy |
|-------|--------|--------|--------|----------|
| openai/gpt-4.1-nano | 9 | 1 | 0 | 90.0% |

`litmus run` exits non-zero when any test fails or errors, so the workflow step fails when a model regresses.

## Choosing the Right Format

| Use Case | Recommended Format |
|----------|-------------------|
| Local development | `terminal` |
| GitHub Actions | `github` |
| Other CI/CD pipelines | `json` |
| Sharing with stakeholders | `html` |
| Archiving results | `html` or `json` |
| Automated processing | `json` |
