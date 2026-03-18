// Package notify provides Web Push notification functionality for LittleLiver.
package notify

import (
	"encoding/json"
	"fmt"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// Payload is the JSON structure sent as the push notification body.
type Payload struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	URL   string            `json:"url,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
}

// Subscription holds the Web Push subscription details from the client.
type Subscription struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

// Pusher sends Web Push notifications. Implementations can be swapped for testing.
type Pusher interface {
	// Send delivers a push notification to the given subscription.
	// Returns the HTTP response from the push service and any error.
	Send(sub Subscription, payload Payload) (*http.Response, error)
}

// VAPIDConfig holds the VAPID keys and subscriber contact.
type VAPIDConfig struct {
	PublicKey  string
	PrivateKey string
	Subscriber string // mailto: or https: URL
}

// WebPusher implements Pusher using the Web Push protocol with VAPID.
type WebPusher struct {
	config VAPIDConfig
}

// NewWebPusher creates a new WebPusher with the given VAPID configuration.
func NewWebPusher(cfg VAPIDConfig) *WebPusher {
	return &WebPusher{config: cfg}
}

// Send delivers a push notification using the Web Push protocol.
func (wp *WebPusher) Send(sub Subscription, payload Payload) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	s := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	resp, err := webpush.SendNotification(body, s, &webpush.Options{
		Subscriber:      wp.config.Subscriber,
		VAPIDPublicKey:  wp.config.PublicKey,
		VAPIDPrivateKey: wp.config.PrivateKey,
		TTL:             60,
	})
	if err != nil {
		return nil, fmt.Errorf("send push: %w", err)
	}

	return resp, nil
}

// GenerateVAPIDKeys generates a new VAPID key pair (base64url-encoded).
func GenerateVAPIDKeys() (privateKey, publicKey string, err error) {
	return webpush.GenerateVAPIDKeys()
}
