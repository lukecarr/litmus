package anthropic

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

// okResponse forces a tool call whose input is the structured response. A
// leading text block exercises the toolInput scan past non-tool_use blocks.
const okResponse = `{"content":[{"type":"text","text":"thinking"},{"type":"tool_use","input":{"name":"Ada"}}],"usage":{"input_tokens":11,"output_tokens":7}}`

func TestCompleteRequestShaping(t *testing.T) {
	var rec captured
	srv := newServer(t, http.StatusOK, okResponse, &rec)

	p := New("secret-key", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	schema := json.RawMessage(`{"type":"object"}`)
	if _, err := p.Complete(context.Background(), "claude-opus-4-8", "system prompt", "user input", schema); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if rec.method != http.MethodPost {
		t.Errorf("method = %q, want POST", rec.method)
	}
	if rec.path != "/messages" {
		t.Errorf("path = %q, want /messages", rec.path)
	}
	if got := rec.header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := rec.header.Get("x-api-key"); got != "secret-key" {
		t.Errorf("x-api-key = %q, want secret-key", got)
	}
	if got := rec.header.Get("anthropic-version"); got != anthropicVersion {
		t.Errorf("anthropic-version = %q, want %q", got, anthropicVersion)
	}

	var req messagesRequest
	if err := json.Unmarshal(rec.body, &req); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	if req.Model != "claude-opus-4-8" {
		t.Errorf("model = %q, want claude-opus-4-8", req.Model)
	}
	if req.MaxTokens != maxTokens {
		t.Errorf("max_tokens = %d, want %d", req.MaxTokens, maxTokens)
	}
	if req.System != "system prompt" {
		t.Errorf("system = %q, want system prompt", req.System)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "user input" {
		t.Errorf("messages = %+v, want a single user message", req.Messages)
	}
	if len(req.Tools) != 1 || req.Tools[0].Name != toolName || string(req.Tools[0].InputSchema) != `{"type":"object"}` {
		t.Errorf("tools = %+v, want one tool carrying the schema", req.Tools)
	}
	if req.ToolChoice.Type != "tool" || req.ToolChoice.Name != toolName {
		t.Errorf("tool_choice = %+v, want the forced tool", req.ToolChoice)
	}
}

func TestCompleteResponseParsing(t *testing.T) {
	srv := newServer(t, http.StatusOK, okResponse, nil)
	p := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	res, err := p.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if string(res.Response) != `{"name":"Ada"}` {
		t.Errorf("Response = %s, want the tool_use input", res.Response)
	}
	if res.Provider != "anthropic" {
		t.Errorf("Provider = %q, want anthropic", res.Provider)
	}
	if res.TokensIn != 11 || res.TokensOut != 7 {
		t.Errorf("tokens in/out = %d/%d, want 11/7", res.TokensIn, res.TokensOut)
	}
	if res.Latency <= 0 {
		t.Errorf("Latency = %v, want > 0", res.Latency)
	}
}

func TestCompleteOmitsAPIKeyWhenEmpty(t *testing.T) {
	var rec captured
	srv := newServer(t, http.StatusOK, okResponse, &rec)
	p := New("", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	if _, err := p.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if got, ok := rec.header["X-Api-Key"]; ok {
		t.Errorf("x-api-key present (%v), want absent when key is empty", got)
	}
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

	p := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(3, time.Millisecond))
	res, err := p.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if calls != 2 {
		t.Errorf("server calls = %d, want 2 (one failure, one success)", calls)
	}
	if res.Provider != "anthropic" {
		t.Errorf("Provider = %q, want anthropic", res.Provider)
	}
}

func TestCompleteReturnsErrorOnNon200(t *testing.T) {
	srv := newServer(t, http.StatusBadRequest, "bad request detail", nil)
	p := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(1, time.Millisecond))

	_, err := p.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`))
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

	p := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(3, time.Millisecond))
	if _, err := p.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected an error, got nil")
	}
	if calls != 1 {
		t.Errorf("server calls = %d, want 1 (client error must not be retried)", calls)
	}
}

func TestCompleteRetriesExhausted(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "still broken")
	}))
	defer srv.Close()

	p := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(2, time.Millisecond))
	_, err := p.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected an error after retries are exhausted, got nil")
	}
	if calls != 2 {
		t.Errorf("server calls = %d, want 2 (all retries used)", calls)
	}
}

func TestCompleteMalformedBody(t *testing.T) {
	srv := newServer(t, http.StatusOK, `{not valid json`, nil)
	p := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(1, time.Millisecond))

	if _, err := p.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected a decode error for a malformed body, got nil")
	}
}

func TestCompleteNoToolUseBlock(t *testing.T) {
	// A response with only a text block has no structured output to return.
	srv := newServer(t, http.StatusOK, `{"content":[{"type":"text","text":"hi"}],"usage":{}}`, nil)
	p := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(1, time.Millisecond))

	_, err := p.Complete(context.Background(), "m", "s", "u", json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "no tool_use block") {
		t.Errorf("error = %v, want it to mention no tool_use block", err)
	}
}

func TestCompleteContextCancelledBeforeRequest(t *testing.T) {
	srv := newServer(t, http.StatusOK, okResponse, nil)
	p := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(3, time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Complete(ctx, "m", "s", "u", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected a context error, got nil")
	}
}

func TestCompleteContextCancelledDuringRetryWait(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // always retryable
	}))
	defer srv.Close()

	// A long retry delay parks the second attempt in the wait, where the
	// cancellation lands and aborts the retry loop.
	p := New("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithRetry(3, time.Hour))
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)

	if _, err := p.Complete(ctx, "m", "s", "u", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected a context error, got nil")
	}
}

func TestDoRequestMarshalError(t *testing.T) {
	p := New("k", WithBaseURL("http://example.com"))
	// An invalid raw schema fails when marshalled into the request payload.
	req := messagesRequest{Tools: []tool{{InputSchema: json.RawMessage(`{invalid`)}}}
	if _, err := p.doRequest(context.Background(), req); err == nil {
		t.Fatal("expected a marshal error, got nil")
	}
}

func TestDoRequestNewRequestError(t *testing.T) {
	p := New("k", WithBaseURL("http://\x7f")) // control char makes the URL invalid
	if _, err := p.doRequest(context.Background(), messagesRequest{Model: "m"}); err == nil {
		t.Fatal("expected a request-construction error, got nil")
	}
}

// errReadTransport returns a response whose body errors on read.
type errReadTransport struct{}

func (errReadTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(errReader{}),
		Header:     make(http.Header),
	}, nil
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestDoRequestBodyReadError(t *testing.T) {
	p := New("k", WithBaseURL("http://example.com"), WithHTTPClient(&http.Client{Transport: errReadTransport{}}))
	if _, err := p.doRequest(context.Background(), messagesRequest{Model: "m"}); err == nil {
		t.Fatal("expected a body-read error, got nil")
	}
}

func TestAPIErrorMessage(t *testing.T) {
	e := &apiError{statusCode: 503, body: "overloaded"}
	if got := e.Error(); !strings.Contains(got, "503") || !strings.Contains(got, "overloaded") {
		t.Errorf("Error() = %q, want it to include the status and body", got)
	}
}
