package notify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// MockPusher records push sends for verification in tests.
type MockPusher struct {
	mu    sync.Mutex
	Sends []mockSend
	Err   error // if non-nil, Send returns this error
}

type mockSend struct {
	Sub     Subscription
	Payload Payload
}

func (m *MockPusher) Send(sub Subscription, payload Payload) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	m.Sends = append(m.Sends, mockSend{Sub: sub, Payload: payload})
	return &http.Response{StatusCode: http.StatusCreated}, nil
}

func TestMockPusher_RecordsSends(t *testing.T) {
	t.Parallel()
	mock := &MockPusher{}

	sub := Subscription{
		Endpoint: "https://push.example.com/sub",
		P256dh:   "test-key",
		Auth:     "test-auth",
	}
	payload := Payload{
		Title: "Medication Reminder",
		Body:  "Time to give Ursodiol",
		URL:   "/log/med?medication_id=abc123",
	}

	resp, err := mock.Send(sub, payload)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	if len(mock.Sends) != 1 {
		t.Fatalf("expected 1 send, got %d", len(mock.Sends))
	}
	if mock.Sends[0].Sub.Endpoint != "https://push.example.com/sub" {
		t.Errorf("wrong endpoint: %q", mock.Sends[0].Sub.Endpoint)
	}
	if mock.Sends[0].Payload.Title != "Medication Reminder" {
		t.Errorf("wrong title: %q", mock.Sends[0].Payload.Title)
	}
	if mock.Sends[0].Payload.URL != "/log/med?medication_id=abc123" {
		t.Errorf("wrong URL: %q", mock.Sends[0].Payload.URL)
	}
}

func TestMockPusher_ReturnsError(t *testing.T) {
	t.Parallel()
	mock := &MockPusher{Err: io.ErrUnexpectedEOF}

	_, err := mock.Send(Subscription{}, Payload{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(mock.Sends) != 0 {
		t.Errorf("expected 0 sends on error, got %d", len(mock.Sends))
	}
}

func TestPusherInterface_Satisfied(t *testing.T) {
	t.Parallel()
	// Verify MockPusher and WebPusher both satisfy the Pusher interface at compile time.
	var _ Pusher = &MockPusher{}
	var _ Pusher = &WebPusher{}
}

func TestWebPusher_SendsToEndpoint(t *testing.T) {
	t.Parallel()

	// Start a test HTTP server that records the request
	var receivedBody []byte
	var receivedMethod string
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		receivedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	// Generate VAPID keys for the test
	privKey, pubKey, err := GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate VAPID keys: %v", err)
	}

	pusher := NewWebPusher(VAPIDConfig{
		PublicKey:  pubKey,
		PrivateKey: privKey,
		Subscriber: "mailto:test@example.com",
	})

	// Generate a client key pair for the subscription (simulating browser keys)
	clientPriv, clientPub, err := GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("generate client keys: %v", err)
	}
	_ = clientPriv // only need public key for subscription

	sub := Subscription{
		Endpoint: ts.URL,
		P256dh:   clientPub,
		Auth:     "dGVzdC1hdXRoLXNlY3JldA", // base64url of "test-auth-secret"
	}

	payload := Payload{
		Title: "Test",
		Body:  "Body",
		URL:   "/test",
	}

	resp, err := pusher.Send(sub, payload)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST, got %q", receivedMethod)
	}
	// The body is encrypted so we can't directly verify the payload content,
	// but we verified the endpoint was called with POST
	if len(receivedBody) == 0 {
		t.Error("expected non-empty encrypted body")
	}
}

func TestPayload_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	p := Payload{
		Title: "Med Reminder",
		Body:  "Time for Ursodiol",
		URL:   "/log/med?medication_id=xyz",
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Title != p.Title || decoded.Body != p.Body || decoded.URL != p.URL {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", decoded, p)
	}
}

func TestGenerateVAPIDKeys(t *testing.T) {
	t.Parallel()
	priv, pub, err := GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys: %v", err)
	}
	if priv == "" {
		t.Error("expected non-empty private key")
	}
	if pub == "" {
		t.Error("expected non-empty public key")
	}
	// Keys should be different
	if priv == pub {
		t.Error("private and public keys should differ")
	}
}

func TestNewWebPusher(t *testing.T) {
	t.Parallel()
	cfg := VAPIDConfig{
		PublicKey:  "pub",
		PrivateKey: "priv",
		Subscriber: "mailto:test@example.com",
	}
	pusher := NewWebPusher(cfg)
	if pusher == nil {
		t.Fatal("expected non-nil pusher")
	}
	if pusher.config.PublicKey != "pub" {
		t.Errorf("expected PublicKey=pub, got %q", pusher.config.PublicKey)
	}
}
