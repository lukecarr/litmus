// Package cloudflare constructs an OpenAI-compatible client targeting the
// Cloudflare AI Gateway "compat" endpoint.
package cloudflare

import (
	"fmt"
	"strings"

	"go.carr.sh/litmus/internal/provider"
)

// Config holds the Cloudflare AI Gateway connection parameters.
type Config struct {
	// AccountID is the Cloudflare account ID. Required.
	AccountID string
	// GatewayID is the AI Gateway name. Required.
	GatewayID string
	// APIKey is the downstream provider key, sent as the Authorization header.
	// It may be empty when the gateway holds the provider keys itself.
	APIKey string
	// GatewayToken sets the cf-aig-authorization header for authenticated
	// gateways. Optional.
	GatewayToken string
}

// New returns a Provider configured for the Cloudflare AI Gateway. It returns
// an error when AccountID or GatewayID is empty, or when neither APIKey nor
// GatewayToken is supplied. Extra options are applied after the Cloudflare
// defaults, so callers can override them (for example, in tests).
func New(cfg Config, opts ...provider.Option) (provider.Provider, error) {
	if cfg.AccountID == "" {
		return nil, fmt.Errorf("cloudflare: account ID required")
	}
	if cfg.GatewayID == "" {
		return nil, fmt.Errorf("cloudflare: gateway ID required")
	}
	if cfg.APIKey == "" && cfg.GatewayToken == "" {
		return nil, fmt.Errorf("cloudflare: a credential is required (provider API key or gateway token)")
	}

	o := []provider.Option{
		provider.WithBaseURL(baseURL(cfg.AccountID, cfg.GatewayID)),
		provider.WithProviderFallback(providerFromModel),
	}
	if cfg.GatewayToken != "" {
		o = append(o, provider.WithHeader("cf-aig-authorization", "Bearer "+cfg.GatewayToken))
	}

	return provider.New(cfg.APIKey, append(o, opts...)...), nil
}

// baseURL builds the compat endpoint base URL for an account and gateway. The
// generic client appends "/chat/completions" to it.
func baseURL(account, gateway string) string {
	return fmt.Sprintf("https://gateway.ai.cloudflare.com/v1/%s/%s/compat", account, gateway)
}

// providerFromModel derives a provider name from a "{provider}/{model}" id. It
// returns an empty string when the id has no provider prefix.
func providerFromModel(model string) string {
	if i := strings.IndexByte(model, '/'); i > 0 {
		return model[:i]
	}
	return ""
}
