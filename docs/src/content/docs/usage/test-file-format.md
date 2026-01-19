---
title: Test File Format
description: Learn how to structure your Litmus test files.
---

The test file is a JSON array of test cases that define the inputs and expected outputs for your LLM tests.

## Structure

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

## Fields

### `name` (required)

A human-readable name for the test case. This is displayed in the test results.

```json
{
  "name": "Extract person info"
}
```

### `input` (required)

The user message sent to the LLM. This is the text that the model will process.

```json
{
  "input": "John Smith is 30 years old and works at Acme Corp"
}
```

### `expected` (required)

The expected JSON output from the LLM. This must match the schema defined in your schema file.

```json
{
  "expected": {
    "name": "John Smith",
    "age": 30,
    "company": "Acme Corp"
  }
}
```

## Complete Example

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

## Tips

- Keep test names descriptive and unique
- Make sure expected outputs match your JSON schema
- Include edge cases and variations in your test suite
- Use consistent formatting for readability
