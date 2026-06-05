package xai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.carr.sh/litmus/internal/provider"
	"go.carr.sh/litmus/internal/xai"
)

func TestNewSetsXAIDefaults(t *testing.T) {
	var header http.Header
	// The response omits a provider field so the constant fallback fires.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		io.WriteString(w, `{"choices":[{"message":{"content":"{}"}}],"usage":{}}`)
	}))
	defer srv.Close()

	// Caller options apply after the xAI defaults, so WithBaseURL here redirects
	// the request from the real xAI URL to the test server.
	p := xai.New("xai-key",
		provider.WithBaseURL(srv.URL),
		provider.WithHTTPClient(srv.Client()),
	)

	res, err := p.Complete(context.Background(), "grok-4", "s", "u", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if got := header.Get("Authorization"); got != "Bearer xai-key" {
		t.Errorf("Authorization = %q, want Bearer xai-key", got)
	}
	if res.Provider != "xai" {
		t.Errorf("Provider = %q, want xai from the constant fallback", res.Provider)
	}
}
