package labextract

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
