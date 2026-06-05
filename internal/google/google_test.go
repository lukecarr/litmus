package google_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.carr.sh/litmus/internal/google"
	"go.carr.sh/litmus/internal/provider"
)

func TestNewSetsGoogleDefaults(t *testing.T) {
	var header http.Header
	// The response omits a provider field so the constant fallback fires.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		io.WriteString(w, `{"choices":[{"message":{"content":"{}"}}],"usage":{}}`)
	}))
	defer srv.Close()

	// Caller options apply after the Google defaults, so WithBaseURL here
	// redirects the request from the real Gemini URL to the test server.
	p := google.New("gemini-key",
		provider.WithBaseURL(srv.URL),
		provider.WithHTTPClient(srv.Client()),
	)

	res, err := p.Complete(context.Background(), "gemini-2.5-flash", "s", "u", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if got := header.Get("Authorization"); got != "Bearer gemini-key" {
		t.Errorf("Authorization = %q, want Bearer gemini-key", got)
	}
	if res.Provider != "google" {
		t.Errorf("Provider = %q, want google from the constant fallback", res.Provider)
	}
}
