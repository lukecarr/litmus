// Package openrouter provides an HTTP client for the OpenRouter API.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL    = "https://openrouter.ai/api/v1"
	defaultMaxRetries = 3
	defaultRetryDelay = time.Second
	defaultTimeout    = 120 * time.Second
)

// Client is an HTTP client for the OpenRouter API.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	maxRetries int
	retryDelay time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets a custom base URL for the API.
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

// NewClient creates a new OpenRouter API client.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		maxRetries: defaultMaxRetries,
		retryDelay: defaultRetryDelay,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Message represents a chat message.
type Message struct {
	// Role is the role of the message, either "system" or "user".
	Role string `json:"role"`
	// Content is the plain text content of the message.
	Content string `json:"content"`
}

// ResponseFormat specifies the structured output format.
type ResponseFormat struct {
	// Type is the type of response format, either "json_schema".
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
	// Provider is the provider that OpenRouter used to complete the request.
	Provider string `json:"provider"`
	// Choices is the list of choices from the model.
	Choices []Choice `json:"choices"`
	// Usage is the token usage for the request.
	Usage Usage `json:"usage"`
}

// CompletionResult contains the response and timing information.
type CompletionResult struct {
	// Response is the raw JSON response from the model.
	Response json.RawMessage
	// Provider is the provider that OpenRouter used to complete the request.
	Provider string
	// TokensIn is the number of tokens in the prompt.
	TokensIn int
	// TokensOut is the number of tokens in the completion.
	TokensOut int
	// Latency is the latency of the request.
	Latency time.Duration
}

// Complete sends a chat completion request with structured output.
func (c *Client) Complete(ctx context.Context, model, systemPrompt, userInput string, schema json.RawMessage) (*CompletionResult, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userInput},
	}

	// Wrap the schema in the required format for OpenRouter
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
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/litmus-cli/litmus")
	req.Header.Set("X-Title", "Litmus CLI")

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
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := chatResp.Choices[0].Message.Content

	return &CompletionResult{
		Response:  json.RawMessage(content),
		Provider:  chatResp.Provider,
		TokensIn:  chatResp.Usage.PromptTokens,
		TokensOut: chatResp.Usage.CompletionTokens,
		Latency:   latency,
	}, nil
}
