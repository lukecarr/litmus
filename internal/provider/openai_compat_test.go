package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// captured records details of a request the test server received.
type captured struct {
	method string
	path   string
	header http.Header
	body   []byte
}

// newServer starts a test server that records the request into rec (when
// non-nil) and replies with the given status and body.
func newServer(t *testing.T, status int, respBody string, rec *captured) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rec != nil {
			b, _ := io.ReadAll(r.Body)
			rec.method = r.Method
			rec.path = r.URL.Path
			rec.header = r.Header.Clone()
			rec.body = b
		}
		w.WriteHeader(status)
		io.WriteString(w, respBody)
	}))
	t.Cleanup(srv.Close)
	return srv
}

const okResponse = `{"id":"resp_1","model":"openai/gpt-4o","provider":"openai","choices":[{"index":0,"message":{"role":"assistant","content":"{\"name\":\"Ada\"}"}}],"usage":{"prompt_tokens":11,"completion_tokens":7}}`

func TestCompleteRequestShaping(t *testing.T) {
	var rec captured
	srv := newServer(t, http.StatusOK, okResponse, &rec)

	c := New("secret-key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithHeader("X-Title", "Litmus CLI"),
	)

	schema := json.RawMessage(`{"type":"object"}`)
	if _, err := c.Complete(context.Background(), "openai/gpt-4o", "system prompt", "user input", schema); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if rec.method != http.MethodPost {
		t.Errorf("method = %q, want POST", rec.method)
	}
	if rec.path != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", rec.path)
	}
	if got := rec.header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := rec.header.Get("Authorization"); got != "Bearer secret-key" {
		t.Errorf("Authorization = %q, want Bearer secret-key", got)
	}
	if got := rec.header.Get("X-Title"); got != "Litmus CLI" {
		t.Errorf("X-Title = %q, want Litmus CLI", got)
	}

	var req ChatRequest
	if err := json.Unmarshal(rec.body, &req); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	if req.Model != "openai/gpt-4o" {
		t.Errorf("model = %q, want openai/gpt-4o", req.Model)
	}
	if len(req.Messages) != 2 ||
		req.Messages[0].Role != "system" || req.Messages[0].Content != "system prompt" ||
		req.Messages[1].Role != "user" || req.Messages[1].Content != "user input" {
		t.Errorf("messages = %+v, want system then user", req.Messages)
	}
	if req.ResponseFormat == nil || req.ResponseFormat.Type != "json_schema" {
		t.Fatalf("response_format = %+v, want type json_schema", req.ResponseFormat)
	}

	var wrapped struct {
		Name   string          `json:"name"`
		Strict bool            `json:"strict"`
		Schema json.RawMessage `json:"schema"`
	}
	if err := json.Unmarshal(req.ResponseFormat.JSONSchema, &wrapped); err != nil {
		t.Fatalf("failed to decode wrapped schema: %v", err)
	}
	if wrapped.Name != "response" || !wrapped.Strict {
		t.Errorf("wrapped schema = %+v, want name=response strict=true", wrapped)
	}
	if string(wrapped.Schema) != `{"type":"object"}` {
		t.Errorf("wrapped schema body = %s, want the original schema", wrapped.Schema)
	}
}

func TestCompleteResponseParsing(t *testing.T) {
	srv := newServer(t, http.StatusOK, okResponse, nil)
	c := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	res, err := c.Complete(context.Background(), "openai/gpt-4o", "s", "u", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if string(res.Response) != `{"name":"Ada"}` {
		t.Errorf("Response = %s, want the message content", res.Response)
	}
	if res.Provider != "openai" {
		t.Errorf("Provider = %q, want openai", res.Provider)
	}
	if res.TokensIn != 11 || res.TokensOut != 7 {
		t.Errorf("tokens in/out = %d/%d, want 11/7", res.TokensIn, res.TokensOut)
	}
	if res.Latency <= 0 {
		t.Errorf("Latency = %v, want > 0", res.Latency)
	}
}

func TestCompleteOmitsAuthorizationWhenNoKey(t *testing.T) {
	var rec captured
	srv := newServer(t, http.StatusOK, okResponse, &rec)
	c := New("", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	if _, err := c.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if got, ok := rec.header["Authorization"]; ok {
		t.Errorf("Authorization header present (%v), want absent when key is empty", got)
	}
}

func TestProviderFallback(t *testing.T) {
	const noProvider = `{"choices":[{"index":0,"message":{"content":"{}"}}],"usage":{}}`

	t.Run("fires when response omits provider", func(t *testing.T) {
		srv := newServer(t, http.StatusOK, noProvider, nil)
		c := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()),
			WithProviderFallback(func(string) string { return "derived" }))
		res, err := c.Complete(context.Background(), "openai/gpt-4o", "s", "u", json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("Complete returned error: %v", err)
		}
		if res.Provider != "derived" {
			t.Errorf("Provider = %q, want derived", res.Provider)
		}
	})

	t.Run("does not override a non-empty provider", func(t *testing.T) {
		srv := newServer(t, http.StatusOK, okResponse, nil)
		c := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()),
			WithProviderFallback(func(string) string { return "derived" }))
		res, err := c.Complete(context.Background(), "openai/gpt-4o", "s", "u", json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("Complete returned error: %v", err)
		}
		if res.Provider != "openai" {
			t.Errorf("Provider = %q, want openai (fallback must not override)", res.Provider)
		}
	})
}

func TestCompleteRetriesThenSucceeds(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "boom")
			return
		}
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, okResponse)
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(3, time.Millisecond))
	res, err := c.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if calls != 2 {
		t.Errorf("server calls = %d, want 2 (one failure, one success)", calls)
	}
	if res.Provider != "openai" {
		t.Errorf("Provider = %q, want openai", res.Provider)
	}
}

func TestCompleteReturnsErrorOnNon200(t *testing.T) {
	srv := newServer(t, http.StatusBadRequest, "bad request detail", nil)
	c := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(1, time.Millisecond))

	_, err := c.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "bad request detail") {
		t.Errorf("error = %q, want it to include status 400 and the response body", err.Error())
	}
}

func TestRetryableStatus(t *testing.T) {
	retryable := []int{500, 502, 503, http.StatusTooManyRequests, http.StatusRequestTimeout}
	for _, code := range retryable {
		if !retryableStatus(code) {
			t.Errorf("retryableStatus(%d) = false, want true", code)
		}
	}

	notRetryable := []int{400, 401, 403, 404, 422}
	for _, code := range notRetryable {
		if retryableStatus(code) {
			t.Errorf("retryableStatus(%d) = true, want false", code)
		}
	}
}

func TestCompleteDoesNotRetryClientError(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "bad request detail")
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(3, time.Millisecond))
	if _, err := c.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected an error, got nil")
	}
	if calls != 1 {
		t.Errorf("server calls = %d, want 1 (client error must not be retried)", calls)
	}
}

func TestCompleteNoChoices(t *testing.T) {
	srv := newServer(t, http.StatusOK, `{"choices":[],"usage":{}}`, nil)
	c := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(1, time.Millisecond))

	_, err := c.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "no choices") {
		t.Errorf("error = %v, want it to mention no choices", err)
	}
}
