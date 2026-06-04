---
title: GitHub Actions
description: Run Litmus in a GitHub Actions workflow with inline annotations on failures.
---

Litmus ships a GitHub Action that runs your tests and annotates failures inline on the test file. The action downloads the release binary that matches the runner, so it works on Linux, macOS, and Windows runners.

## Quick start

```yaml
name: litmus
on: pull_request
jobs:
  litmus:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: lukecarr/litmus@v0.3.0
        with:
          tests: example/tests.json
          schema: example/schema.json
          prompt-file: example/prompt.txt
          model: openai/gpt-4.1-nano
          api-key: ${{ secrets.OPENROUTER_API_KEY }}
```

The step fails when any test fails or errors. Each failure shows up as an inline annotation on the test file, and the run gets a results table in its job summary.

## Inputs

Each input maps to a `litmus run` flag. `tests`, `schema`, and `model` are required.

| Input | Flag | Default | Notes |
|-------|------|---------|-------|
| `tests` | `--tests` | | Required. Path to the test cases JSON file. |
| `schema` | `--schema` | | Required. Path to the JSON schema file. |
| `model` | `--model` | | Required. One model per line, or comma-separated. |
| `prompt` | `--prompt` | | System prompt. Mutually exclusive with `prompt-file`. |
| `prompt-file` | `--prompt-file` | | Path to a file containing the system prompt. |
| `parallel` | `--parallel` | `1` | Parallel requests per model. |
| `output` | `--output` | `github` | Output format. Defaults to `github` for inline annotations. |
| `provider` | `--provider` | `openrouter` | `openrouter` or `cloudflare`. |
| `api-key` | env | | Provider API key, passed through the environment. |
| `cf-account-id` | env | | Cloudflare account ID. |
| `cf-gateway` | env | | Cloudflare AI Gateway ID. |
| `cf-token` | env | | Cloudflare AI Gateway token. |
| `version` | | action ref | Litmus release to download. Defaults to the pinned tag, else the latest release. See [Versions](#versions). |
| `working-directory` | | `.` | Directory to run litmus from. |

## Credentials

Pass secrets through the `api-key` and `cf-token` inputs, wired from repository secrets. The action sets the provider's environment variable (`OPENROUTER_API_KEY` or `CLOUDFLARE_API_KEY`) from `api-key`, so the key never appears on the command line. If you already export the key in the job environment, leave `api-key` empty and the action keeps your value.

To test several models in one run, list them one per line:

```yaml
      - uses: lukecarr/litmus@v0.3.0
        with:
          tests: tests.json
          schema: schema.json
          prompt-file: prompt.txt
          model: |
            openai/gpt-4.1-nano
            anthropic/claude-3.5-sonnet
          api-key: ${{ secrets.OPENROUTER_API_KEY }}
```

### Cloudflare AI Gateway

```yaml
      - uses: lukecarr/litmus@v0.3.0
        with:
          tests: tests.json
          schema: schema.json
          prompt-file: prompt.txt
          model: openai/gpt-4.1-nano
          provider: cloudflare
          cf-account-id: ${{ vars.CLOUDFLARE_ACCOUNT_ID }}
          cf-gateway: my-gateway
          api-key: ${{ secrets.OPENAI_API_KEY }}
          cf-token: ${{ secrets.CF_AIG_TOKEN }}
```

## Versions

The action runs the Litmus release that matches the tag you pin, so there is one version to think about:

- `uses: lukecarr/litmus@v0.3.0` downloads and runs Litmus v0.3.0.
- `uses: lukecarr/litmus@main` (or a branch or commit SHA) runs the latest release.

Pin an exact release tag for reproducible runs. To run a binary different from the pinned ref, set `version` explicitly:

```yaml
      - uses: lukecarr/litmus@main
        with:
          version: v0.3.0
          tests: tests.json
          schema: schema.json
          prompt-file: prompt.txt
          model: openai/gpt-4.1-nano
          api-key: ${{ secrets.OPENROUTER_API_KEY }}
```

## Output

The action defaults `output` to `github`, which prints workflow-command annotations and appends a job summary. See [Output Formats](/output/formats/) for what the annotations look like. Set `output` to `json` or `terminal` for a different format in the log.
