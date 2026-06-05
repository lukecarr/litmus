package openai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.carr.sh/litmus/internal/openai"
	"go.carr.sh/litmus/internal/provider"
)

func TestNewSetsOpenAIDefaults(t *testing.T) {
	var header http.Header
	// The response omits a provider field so the constant fallback fires.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		io.WriteString(w, `{"choices":[{"message":{"content":"{}"}}],"usage":{}}`)
	}))
	defer srv.Close()

	// Caller options apply after the OpenAI defaults, so WithBaseURL here
	// redirects the request from the real OpenAI URL to the test server.
	p := openai.New("openai-key",
		provider.WithBaseURL(srv.URL),
		provider.WithHTTPClient(srv.Client()),
	)

	res, err := p.Complete(context.Background(), "gpt-4o", "s", "u", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if got := header.Get("Authorization"); got != "Bearer openai-key" {
		t.Errorf("Authorization = %q, want Bearer openai-key", got)
	}
	if res.Provider != "openai" {
		t.Errorf("Provider = %q, want openai from the constant fallback", res.Provider)
	}
}
