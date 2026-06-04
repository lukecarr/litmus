// Package openrouter constructs an OpenAI-compatible client targeting the
// OpenRouter API.
package openrouter

import "go.carr.sh/litmus/internal/provider"

const baseURL = "https://openrouter.ai/api/v1"

// New returns a Provider configured for OpenRouter. Extra options are applied
// after the OpenRouter defaults, so callers can override them (for example, in
// tests).
func New(apiKey string, opts ...provider.Option) provider.Provider {
	base := []provider.Option{
		provider.WithBaseURL(baseURL),
		provider.WithHeader("HTTP-Referer", "https://github.com/lukecarr/litmus"),
		provider.WithHeader("X-Title", "Litmus CLI"),
	}
	return provider.New(apiKey, append(base, opts...)...)
}
