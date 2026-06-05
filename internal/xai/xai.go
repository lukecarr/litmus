// Package xai constructs an OpenAI-compatible client targeting the xAI (Grok)
// API directly, without a gateway in the request path.
package xai

import "go.carr.sh/litmus/internal/provider"

const baseURL = "https://api.x.ai/v1"

// New returns a Provider configured for the xAI API. Models are named without a
// provider prefix, for example "grok-4". Extra options are applied after the
// xAI defaults, so callers can override them (for example, in tests).
func New(apiKey string, opts ...provider.Option) provider.Provider {
	base := []provider.Option{
		provider.WithBaseURL(baseURL),
		// xAI responses do not carry a provider field, so report a constant.
		provider.WithProviderFallback(func(string) string { return "xai" }),
	}
	return provider.New(apiKey, append(base, opts...)...)
}
