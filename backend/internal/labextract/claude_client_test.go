package labextract

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewHTTPClaudeClientWithBaseURL(t *testing.T) {
	t.Parallel()
	client := NewHTTPClaudeClientWithBaseURL("test-key", "http://mock-server:9999")
	if client.baseURL != "http://mock-server:9999" {
		t.Errorf("expected base URL http://mock-server:9999, got %s", client.baseURL)
	}
	if client.apiKey != "test-key" {
		t.Errorf("expected api key test-key, got %s", client.apiKey)
	}
}

func TestHTTPClaudeClient_SuccessfulExtraction(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key=test-key, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version=2023-06-01, got %s", r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type=application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body structure
		body, _ := io.ReadAll(r.Body)
		var req claudeRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if req.System == "" {
			t.Error("expected system prompt to be set")
		}
		if len(req.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(req.Messages))
		}
		if len(req.Messages[0].Content) != 2 { // 1 image + 1 text
			t.Fatalf("expected 2 content parts, got %d", len(req.Messages[0].Content))
		}
		if req.Messages[0].Content[0].Type != "image" {
			t.Errorf("expected first part to be image, got %s", req.Messages[0].Content[0].Type)
		}
		if req.Messages[0].Content[0].Source.MediaType != "image/jpeg" {
			t.Errorf("expected media_type image/jpeg, got %s", req.Messages[0].Content[0].Source.MediaType)
		}

		// Return successful response
		resp := claudeResponse{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: `[{"test_name": "ALT", "value": "45", "unit": "U/L", "normal_range": "7-56", "confidence": "high"}]`},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &HTTPClaudeClient{
		apiKey:     "test-key",
		model:      "claude-sonnet-4-20250514",
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	result, err := client.ExtractLabResults(context.Background(), []ImageData{
		{Data: []byte("fake-image"), ContentType: "image/jpeg"},
	}, "test prompt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestHTTPClaudeClient_NonOKStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := &HTTPClaudeClient{
		apiKey:     "test-key",
		model:      "claude-sonnet-4-20250514",
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := client.ExtractLabResults(context.Background(), []ImageData{
		{Data: []byte("fake-image"), ContentType: "image/jpeg"},
	}, "test prompt")

	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestHTTPClaudeClient_APIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{
			Error: &struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				Type:    "invalid_request",
				Message: "bad request",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &HTTPClaudeClient{
		apiKey:     "test-key",
		model:      "claude-sonnet-4-20250514",
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := client.ExtractLabResults(context.Background(), []ImageData{
		{Data: []byte("fake-image"), ContentType: "image/jpeg"},
	}, "test prompt")

	if err == nil {
		t.Fatal("expected error for API error response")
	}
}

func TestNewHTTPClaudeClient(t *testing.T) {
	t.Parallel()
	client := NewHTTPClaudeClient("my-api-key")
	if client.baseURL != "https://api.anthropic.com" {
		t.Errorf("expected default base URL https://api.anthropic.com, got %s", client.baseURL)
	}
	if client.apiKey != "my-api-key" {
		t.Errorf("expected api key my-api-key, got %s", client.apiKey)
	}
	if client.model != "claude-sonnet-4-20250514" {
		t.Errorf("expected default model, got %s", client.model)
	}
}

func TestHTTPClaudeClient_ReadBodyError(t *testing.T) {
	t.Parallel()

	// Create a server that closes the connection after sending headers but before body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set Content-Length to a large value but don't send the body,
		// then hijack the connection to force a read error
		w.Header().Set("Content-Length", "99999")
		w.WriteHeader(http.StatusOK)
		// Flush headers then close connection to cause io.ReadAll error
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		// Hijack the connection and close it immediately
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	defer server.Close()

	client := &HTTPClaudeClient{
		apiKey:     "test-key",
		model:      "claude-sonnet-4-20250514",
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := client.ExtractLabResults(context.Background(), []ImageData{
		{Data: []byte("fake-image"), ContentType: "image/jpeg"},
	}, "test prompt")

	if err == nil {
		t.Fatal("expected error when response body read fails")
	}
}

func TestHTTPClaudeClient_NoTextContent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := claudeResponse{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &HTTPClaudeClient{
		apiKey:     "test-key",
		model:      "claude-sonnet-4-20250514",
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	_, err := client.ExtractLabResults(context.Background(), []ImageData{
		{Data: []byte("fake-image"), ContentType: "image/jpeg"},
	}, "test prompt")

	if err == nil {
		t.Fatal("expected error when no text content in response")
	}
}
