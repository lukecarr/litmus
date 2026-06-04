package openrouter_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.carr.sh/litmus/internal/openrouter"
	"go.carr.sh/litmus/internal/provider"
)

func TestNewSetsOpenRouterDefaults(t *testing.T) {
	var header http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		io.WriteString(w, `{"provider":"openai","choices":[{"message":{"content":"{}"}}],"usage":{}}`)
	}))
	defer srv.Close()

	// Caller options apply after the OpenRouter defaults, so WithBaseURL here
	// redirects the request from the real OpenRouter URL to the test server.
	p := openrouter.New("router-key",
		provider.WithBaseURL(srv.URL),
		provider.WithHTTPClient(srv.Client()),
	)

	res, err := p.Complete(context.Background(), "openai/gpt-4o", "s", "u", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if got := header.Get("Authorization"); got != "Bearer router-key" {
		t.Errorf("Authorization = %q, want Bearer router-key", got)
	}
	if got := header.Get("HTTP-Referer"); got != "https://github.com/lukecarr/litmus" {
		t.Errorf("HTTP-Referer = %q, want the litmus repo URL", got)
	}
	if got := header.Get("X-Title"); got != "Litmus CLI" {
		t.Errorf("X-Title = %q, want Litmus CLI", got)
	}
	if res.Provider != "openai" {
		t.Errorf("Provider = %q, want openai from the response (no fallback)", res.Provider)
	}
}
