// Package openai constructs an OpenAI-compatible client targeting the OpenAI
// API directly, without a gateway in the request path.
package openai

import "go.carr.sh/litmus/internal/provider"

const baseURL = "https://api.openai.com/v1"

// New returns a Provider configured for the OpenAI API. Models are named
// without a provider prefix, for example "gpt-4o". Extra options are applied
// after the OpenAI defaults, so callers can override them (for example, in
// tests).
func New(apiKey string, opts ...provider.Option) provider.Provider {
	base := []provider.Option{
		provider.WithBaseURL(baseURL),
		// OpenAI responses do not carry a provider field, so report a constant.
		provider.WithProviderFallback(func(string) string { return "openai" }),
	}
	return provider.New(apiKey, append(base, opts...)...)
}
