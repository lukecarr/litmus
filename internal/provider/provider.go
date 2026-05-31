// Package provider defines the LLM provider abstraction used by litmus and
// implements a generic OpenAI-compatible chat-completions client that the
// concrete providers (OpenRouter, Cloudflare AI Gateway) build on.
package provider

import (
	"context"
	"encoding/json"
	"time"
)

// CompletionResult contains a provider's response and timing information for a
// single chat completion.
type CompletionResult struct {
	// Response is the raw JSON response from the model.
	Response json.RawMessage
	// Provider is the upstream provider that served the request.
	Provider string
	// TokensIn is the number of tokens in the prompt.
	TokensIn int
	// TokensOut is the number of tokens in the completion.
	TokensOut int
	// Latency is the latency of the request.
	Latency time.Duration
}

// Provider performs a structured chat completion against an LLM backend.
type Provider interface {
	// Complete sends a chat completion request with structured output and
	// returns the model's response.
	Complete(ctx context.Context, model, systemPrompt, userInput string, schema json.RawMessage) (*CompletionResult, error)
}
