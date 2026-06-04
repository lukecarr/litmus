package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
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

// Client is a generic OpenAI-compatible chat-completions client. Concrete
// providers configure it with a base URL and any provider-specific headers.
type Client struct {
	httpClient       *http.Client
	apiKey           string
	baseURL          string
	headers          map[string]string
	maxRetries       int
	retryDelay       time.Duration
	providerFallback func(model string) string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets the base URL for the API. The client appends
// "/chat/completions" to it.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithRetry configures retry behavior.
func WithRetry(maxRetries int, retryDelay time.Duration) Option {
	return func(c *Client) {
		c.maxRetries = maxRetries
		c.retryDelay = retryDelay
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHeader sets a static header sent with every request.
func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithProviderFallback sets a function that derives a provider name from the
// model when the API response does not include one.
func WithProviderFallback(fn func(model string) string) Option {
	return func(c *Client) {
		c.providerFallback = fn
	}
}

// New creates a generic OpenAI-compatible client. apiKey may be empty to omit
// the Authorization header, for gateways that supply provider credentials
// themselves.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		apiKey:     apiKey,
		headers:    make(map[string]string),
		maxRetries: defaultMaxRetries,
		retryDelay: defaultRetryDelay,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

var _ Provider = (*Client)(nil)

// Message represents a chat message.
type Message struct {
	// Role is the role of the message, either "system" or "user".
	Role string `json:"role"`
	// Content is the plain text content of the message.
	Content string `json:"content"`
}

// ResponseFormat specifies the structured output format.
type ResponseFormat struct {
	// Type is the type of response format, e.g. "json_schema".
	Type string `json:"type"`
	// JSONSchema is the structured output schema for the response.
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	// Model is the name of the model to use.
	Model string `json:"model"`
	// Messages is the list of messages to send to the model.
	Messages []Message `json:"messages"`
	// ResponseFormat is the response format for the model.
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	// PromptTokens is the number of tokens in the prompt.
	PromptTokens int `json:"prompt_tokens"`
	// CompletionTokens is the number of tokens in the completion.
	CompletionTokens int `json:"completion_tokens"`
}

// Choice represents a single completion choice.
type Choice struct {
	// Index is the index of the choice, zero-based.
	Index int `json:"index"`
	// Message is the message for the choice.
	Message Message `json:"message"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	// ID is the ID of the response.
	ID string `json:"id"`
	// Model is the name of the model that completed the request.
	Model string `json:"model"`
	// Provider is the upstream provider that served the request. Not every
	// OpenAI-compatible backend sets it.
	Provider string `json:"provider"`
	// Choices is the list of choices from the model.
	Choices []Choice `json:"choices"`
	// Usage is the token usage for the request.
	Usage Usage `json:"usage"`
}

// Complete sends a chat completion request with structured output.
func (c *Client) Complete(ctx context.Context, model, systemPrompt, userInput string, schema json.RawMessage) (*CompletionResult, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userInput},
	}

	// Wrap the schema in the format expected by OpenAI-compatible APIs.
	wrappedSchema := map[string]any{
		"name":   "response",
		"strict": true,
		"schema": schema,
	}
	wrappedSchemaBytes, err := json.Marshal(wrappedSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap schema: %w", err)
	}

	req := ChatRequest{
		Model:    model,
		Messages: messages,
		ResponseFormat: &ResponseFormat{
			Type:       "json_schema",
			JSONSchema: wrappedSchemaBytes,
		},
	}

	var result *CompletionResult
	var lastErr error

	for attempt := range c.maxRetries {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		result, lastErr = c.doRequest(ctx, req)
		if lastErr == nil {
			return result, nil
		}

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Don't retry terminal client errors.
		var apiErr *apiError
		if errors.As(lastErr, &apiErr) && !retryableStatus(apiErr.statusCode) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", c.maxRetries, lastErr)
}

func (c *Client) doRequest(ctx context.Context, chatReq ChatRequest) (*CompletionResult, error) {
	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
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

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := chatResp.Choices[0].Message.Content

	result := &CompletionResult{
		Response:  json.RawMessage(content),
		Provider:  chatResp.Provider,
		TokensIn:  chatResp.Usage.PromptTokens,
		TokensOut: chatResp.Usage.CompletionTokens,
		Latency:   latency,
	}

	// Fall back to a derived provider name when the backend omits one.
	if result.Provider == "" && c.providerFallback != nil {
		result.Provider = c.providerFallback(chatReq.Model)
	}

	return result, nil
}
