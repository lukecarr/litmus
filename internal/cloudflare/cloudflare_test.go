package cloudflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.carr.sh/litmus/internal/provider"
)

func TestBaseURL(t *testing.T) {
	got := baseURL("acct", "gw")
	want := "https://gateway.ai.cloudflare.com/v1/acct/gw/compat"
	if got != want {
		t.Errorf("baseURL = %q, want %q", got, want)
	}
}

func TestProviderFromModel(t *testing.T) {
	cases := map[string]string{
		"openai/gpt-4o":               "openai",
		"anthropic/claude-3.5-sonnet": "anthropic",
		"gpt-4o":                      "",
		"":                            "",
		"/leading":                    "",
	}
	for in, want := range cases {
		if got := providerFromModel(in); got != want {
			t.Errorf("providerFromModel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewValidation(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		ok   bool
	}{
		{"missing account", Config{GatewayID: "gw", APIKey: "k"}, false},
		{"missing gateway", Config{AccountID: "acct", APIKey: "k"}, false},
		{"no credentials", Config{AccountID: "acct", GatewayID: "gw"}, false},
		{"api key only", Config{AccountID: "acct", GatewayID: "gw", APIKey: "k"}, true},
		{"gateway token only", Config{AccountID: "acct", GatewayID: "gw", GatewayToken: "t"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.cfg)
			if tc.ok && err != nil {
				t.Errorf("New(%+v) returned error: %v", tc.cfg, err)
			}
			if !tc.ok && err == nil {
				t.Errorf("New(%+v) = nil error, want an error", tc.cfg)
			}
		})
	}
}

const noProviderResponse = `{"choices":[{"index":0,"message":{"content":"{}"}}],"usage":{}}`

func TestGatewayTokenSetsHeaderAndOmitsAuthorization(t *testing.T) {
	var header http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		io.WriteString(w, noProviderResponse)
	}))
	defer srv.Close()

	p, err := New(Config{AccountID: "a", GatewayID: "g", GatewayToken: "tok"},
		provider.WithBaseURL(srv.URL), provider.WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	res, err := p.Complete(context.Background(), "openai/gpt-4o", "s", "u", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if got := header.Get("cf-aig-authorization"); got != "Bearer tok" {
		t.Errorf("cf-aig-authorization = %q, want Bearer tok", got)
	}
	if got, ok := header["Authorization"]; ok {
		t.Errorf("Authorization present (%v), want absent when no API key is set", got)
	}
	if res.Provider != "openai" {
		t.Errorf("Provider = %q, want openai derived from the model prefix", res.Provider)
	}
}

func TestAPIKeySetsAuthorizationWithoutGatewayHeader(t *testing.T) {
	var header http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		io.WriteString(w, noProviderResponse)
	}))
	defer srv.Close()

	p, err := New(Config{AccountID: "a", GatewayID: "g", APIKey: "provkey"},
		provider.WithBaseURL(srv.URL), provider.WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if _, err := p.Complete(context.Background(), "anthropic/claude", "s", "u", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if got := header.Get("Authorization"); got != "Bearer provkey" {
		t.Errorf("Authorization = %q, want Bearer provkey", got)
	}
	if got, ok := header["Cf-Aig-Authorization"]; ok {
		t.Errorf("cf-aig-authorization present (%v), want absent when no gateway token is set", got)
	}
}
