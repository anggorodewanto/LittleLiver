package labextract

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPClaudeClient is a real implementation of ClaudeClient using the Anthropic Messages API.
type HTTPClaudeClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

// DefaultModel is the Claude model used for lab extraction.
const DefaultModel = "claude-sonnet-4-6"

// NewHTTPClaudeClient creates a new Claude API client.
func NewHTTPClaudeClient(apiKey string) *HTTPClaudeClient {
	return &HTTPClaudeClient{
		apiKey:     apiKey,
		model:      DefaultModel,
		httpClient: http.DefaultClient,
		baseURL:    "https://api.anthropic.com",
	}
}

// NewHTTPClaudeClientWithBaseURL creates a Claude API client with a custom base URL.
// Useful for testing with a mock API server.
func NewHTTPClaudeClientWithBaseURL(apiKey, baseURL string) *HTTPClaudeClient {
	return &HTTPClaudeClient{
		apiKey:     apiKey,
		model:      DefaultModel,
		httpClient: http.DefaultClient,
		baseURL:    baseURL,
	}
}

// claudeRequest is the request body for the Anthropic Messages API.
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type   string       `json:"type"`
	Text   string       `json:"text,omitempty"`
	Source *imageSource `json:"source,omitempty"`
}

type imageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// claudeResponse is the response body from the Anthropic Messages API.
type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ExtractLabResults sends images to Claude and returns the raw text response.
func (c *HTTPClaudeClient) ExtractLabResults(ctx context.Context, images []ImageData, prompt string) (string, error) {
	// Build content parts: images first, then a user message asking to extract
	parts := make([]contentPart, 0, len(images)+1)
	for _, img := range images {
		parts = append(parts, contentPart{
			Type: "image",
			Source: &imageSource{
				Type:      "base64",
				MediaType: img.ContentType,
				Data:      base64.StdEncoding.EncodeToString(img.Data),
			},
		})
	}
	parts = append(parts, contentPart{
		Type: "text",
		Text: "Please extract all lab test results from these images.",
	})

	reqBody := claudeRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    prompt,
		Messages: []claudeMessage{
			{Role: "user", Content: parts},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if claudeResp.Error != nil {
		return "", fmt.Errorf("claude API error: %s", claudeResp.Error.Message)
	}

	// Extract text from the first text content block
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}
