---
title: Quick Start
description: Get started with Litmus in 5 minutes.
---

This guide will help you run your first Litmus test in just a few minutes.

## Prerequisites

- Litmus installed on your system (see [Installation](/getting-started/installation/))
- An OpenRouter API key

## Step 1: Set Your API Key

Set your OpenRouter API key as an environment variable:

```bash
export OPENROUTER_API_KEY="your-api-key"
```

## Step 2: Create a Test File

Create a test file called `tests.json`:

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

## Step 3: Create a JSON Schema

Create a schema file called `schema.json`:

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

## Step 4: Create a Prompt File

Create a prompt file called `prompt.txt`:

```plain
Extract the person's name, age, and company from the given text.
```

## Step 5: Run Tests

Run your tests against a model:

```bash
litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt --model openai/gpt-4.1-nano
```

## Step 6: Compare Models

Run tests against multiple models to compare them:

```bash
litmus run --tests tests.json --schema schema.json --prompt-file prompt.txt --model openai/gpt-4.1-nano --model mistralai/mistral-nemo
```

## Next Steps

- [CLI Reference](/usage/cli-reference/) - Learn about all available options
- [Test File Format](/usage/test-file-format/) - Understand the test file structure
- [JSON Schema](/usage/json-schema/) - Learn about JSON schema requirements
