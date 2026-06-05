// Package google constructs an OpenAI-compatible client targeting the Gemini
// API through Google's OpenAI-compatible endpoint.
package google

import "go.carr.sh/litmus/internal/provider"

const baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

// New returns a Provider configured for the Gemini API. Models are named
// without a provider prefix, for example "gemini-2.5-flash". Extra options are
// applied after the Google defaults, so callers can override them (for example,
// in tests).
func New(apiKey string, opts ...provider.Option) provider.Provider {
	base := []provider.Option{
		provider.WithBaseURL(baseURL),
		// Gemini's OpenAI-compatible responses omit a provider field, so report
		// a constant.
		provider.WithProviderFallback(func(string) string { return "google" }),
	}
	return provider.New(apiKey, append(base, opts...)...)
}
