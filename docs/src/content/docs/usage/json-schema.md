---
title: JSON Schema
description: How to define JSON schemas for structured LLM outputs.
---

The schema file should be a valid [JSON Schema](https://json-schema.org/). It is passed to OpenRouter's `response_format` parameter to enforce structured output from the LLM.

## Basic Structure

```json
{
  "type": "object",
  "properties": {
    "field1": { "type": "string" },
    "field2": { "type": "integer" }
  },
  "required": ["field1", "field2"],
  "additionalProperties": false
}
```

## Supported Types

JSON Schema supports several data types:

- `string` - Text values
- `integer` - Whole numbers
- `number` - Decimal numbers
- `boolean` - True/false values
- `array` - Lists of values
- `object` - Nested objects

## Examples

### Person Extraction Schema

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

### Sentiment Analysis Schema

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

### Entity Extraction with Arrays

```json
{
  "type": "object",
  "properties": {
    "people": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "role": { "type": "string" }
        },
        "required": ["name", "role"]
      }
    },
    "organizations": {
      "type": "array",
      "items": { "type": "string" }
    }
  },
  "required": ["people", "organizations"],
  "additionalProperties": false
}
```

## Best Practices

1. **Use `additionalProperties: false`** - This ensures the LLM only outputs the fields you specify
2. **Mark required fields** - Use the `required` array to specify which fields must be present
3. **Use enums for constrained values** - When you need specific values, use `enum`
4. **Add constraints** - Use `minimum`, `maximum`, `minLength`, `maxLength` for validation

## Resources

- [JSON Schema Documentation](https://json-schema.org/)
- [JSON Schema Examples](https://json-schema.org/learn/miscellaneous-examples)
