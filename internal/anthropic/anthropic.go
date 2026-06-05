// Package anthropic implements the provider.Provider interface against
// Anthropic's native Messages API. Anthropic is not OpenAI-compatible, so this
// client talks to /v1/messages directly. Structured output is obtained by
// forcing a single tool call whose input_schema is the caller's JSON schema and
// reading the resulting tool_use block's input.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.carr.sh/litmus/internal/provider"
)

const (
	defaultBaseURL   = "https://api.anthropic.com/v1"
	anthropicVersion = "2023-06-01"
	// toolName is the forced tool whose input carries the structured response.
	toolName = "respond"
	// maxTokens caps the completion. Structured responses are small; this leaves
	// generous headroom without risking truncation.
	maxTokens = 8192

	defaultMaxRetries = 3
	defaultRetryDelay = time.Second
	defaultTimeout    = 120 * time.Second
)

// apiError is returned when the API responds with a non-200 status.
type apiError struct {
	statusCode int
	body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.statusCode, e.body)
}

// retryableStatus reports whether an HTTP status code represents a transient
// failure worth retrying.
func retryableStatus(code int) bool {
	return code >= 500 || code == http.StatusTooManyRequests || code == http.StatusRequestTimeout
}

// Provider talks to the Anthropic Messages API.
type Provider struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	maxRetries int
	retryDelay time.Duration
}

// Option configures a Provider.
type Option func(*Provider)

// WithBaseURL sets the base URL for the API. The client appends "/messages".
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(p *Provider) {
		p.httpClient = httpClient
	}
}

// WithRetry configures retry behavior.
func WithRetry(maxRetries int, retryDelay time.Duration) Option {
	return func(p *Provider) {
		p.maxRetries = maxRetries
		p.retryDelay = retryDelay
	}
}

// New returns a Provider configured for the Anthropic Messages API. Models are
// named without a provider prefix, for example "claude-opus-4-8".
func New(apiKey string, opts ...Option) *Provider {
	p := &Provider{
		httpClient: &http.Client{Timeout: defaultTimeout},
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		maxRetries: defaultMaxRetries,
		retryDelay: defaultRetryDelay,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

var _ provider.Provider = (*Provider)(nil)

// message is a single chat message in the Messages API.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// tool describes a tool the model may call. input_schema is the caller's JSON
// schema, which the forced tool call must satisfy.
type tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// toolChoice forces the model to call a specific tool.
type toolChoice struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// messagesRequest is the body of a /messages request.
type messagesRequest struct {
	Model      string     `json:"model"`
	MaxTokens  int        `json:"max_tokens"`
	System     string     `json:"system,omitempty"`
	Messages   []message  `json:"messages"`
	Tools      []tool     `json:"tools"`
	ToolChoice toolChoice `json:"tool_choice"`
}

// contentBlock is one block of the response content array. Only tool_use blocks
// carry an input payload.
type contentBlock struct {
	Type  string          `json:"type"`
	Input json.RawMessage `json:"input"`
}

// usage holds token counts for the request.
type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// messagesResponse is the body of a /messages response.
type messagesResponse struct {
	Content []contentBlock `json:"content"`
	Usage   usage          `json:"usage"`
}

// Complete sends a chat completion that forces a tool call to obtain structured
// output, then returns the tool input as the response.
func (p *Provider) Complete(ctx context.Context, model, systemPrompt, userInput string, schema json.RawMessage) (*provider.CompletionResult, error) {
	req := messagesRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  []message{{Role: "user", Content: userInput}},
		Tools: []tool{{
			Name:        toolName,
			Description: "Return the structured response.",
			InputSchema: schema,
		}},
		ToolChoice: toolChoice{Type: "tool", Name: toolName},
	}

	var result *provider.CompletionResult
	var lastErr error

	for attempt := range p.maxRetries {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(p.retryDelay * time.Duration(attempt)):
			}
		}

		result, lastErr = p.doRequest(ctx, req)
		if lastErr == nil {
			return result, nil
		}

		// Don't retry on context cancellation.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Don't retry terminal client errors.
		var apiErr *apiError
		if errors.As(lastErr, &apiErr) && !retryableStatus(apiErr.statusCode) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", p.maxRetries, lastErr)
}

func (p *Provider) doRequest(ctx context.Context, msgReq messagesRequest) (*provider.CompletionResult, error) {
	body, err := json.Marshal(msgReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicVersion)
	if p.apiKey != "" {
		req.Header.Set("x-api-key", p.apiKey)
	}

	start := time.Now()
	resp, err := p.httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &apiError{statusCode: resp.StatusCode, body: string(respBody)}
	}

	var msgResp messagesResponse
	if err := json.Unmarshal(respBody, &msgResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	input := toolInput(msgResp.Content)
	if input == nil {
		return nil, fmt.Errorf("no tool_use block in response")
	}

	return &provider.CompletionResult{
		Response:  input,
		Provider:  "anthropic",
		TokensIn:  msgResp.Usage.InputTokens,
		TokensOut: msgResp.Usage.OutputTokens,
		Latency:   latency,
	}, nil
}

// toolInput returns the input of the first tool_use block, or nil when none is
// present.
func toolInput(blocks []contentBlock) json.RawMessage {
	for _, b := range blocks {
		if b.Type == "tool_use" {
			return b.Input
		}
	}
	return nil
}
