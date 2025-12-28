// Package webhooks implements an outgoing webhook delivery system for listmonk.
// It delivers events to configured webhook endpoints with retry logic and HMAC signatures.
package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/knadh/listmonk/models"
)

// Webhook represents a webhook configuration loaded from settings.
type Webhook struct {
	UUID           string
	Enabled        bool
	Name           string
	URL            string
	Events         map[string]struct{} // O(1) lookup
	AuthType       string
	AuthBasicUser  string
	AuthBasicPass  string
	AuthHMACSecret string
	MaxRetries     int
	Timeout        time.Duration
}

// Manager handles webhook event delivery.
type Manager struct {
	webhooks      []Webhook
	log           *log.Logger
	mu            sync.RWMutex
	versionString string
}

// New creates a new webhook manager.
func New(log *log.Logger, versionString string) *Manager {
	return &Manager{
		webhooks:      []Webhook{},
		log:           log,
		versionString: versionString,
	}
}

// Load loads webhooks from settings into memory.
func (m *Manager) Load(settings []models.Webhook) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.webhooks = make([]Webhook, 0, len(settings))
	for _, s := range settings {
		if !s.Enabled {
			continue
		}

		// Parse timeout with default.
		timeout, err := time.ParseDuration(s.Timeout)
		if err != nil || timeout <= 0 {
			timeout = 30 * time.Second
		}

		// Default max retries.
		maxRetries := s.MaxRetries
		if maxRetries <= 0 {
			maxRetries = 3
		}

		events := make(map[string]struct{})
		for _, ev := range s.Events {
			events[ev] = struct{}{}
		}

		m.webhooks = append(m.webhooks, Webhook{
			UUID:           s.UUID,
			Enabled:        s.Enabled,
			Name:           s.Name,
			URL:            s.URL,
			Events:         events,
			AuthType:       s.AuthType,
			AuthBasicUser:  s.AuthBasicUser,
			AuthBasicPass:  s.AuthBasicPass,
			AuthHMACSecret: s.AuthHMACSecret,
			MaxRetries:     maxRetries,
			Timeout:        timeout,
		})
	}

	numHooks := len(m.webhooks)
	label := "webhook"
	if numHooks > 1 {
		label = "webhooks"
	}
	m.log.Printf("loaded %d %s", numHooks, label)
}

// Trigger fires all webhooks subscribed to the given event.
// Delivery happens asynchronously in goroutines.
func (m *Manager) Trigger(event string, data any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Build the event payload once.
	payload := models.WebhookEvent{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		m.log.Printf("error marshaling webhook payload: %v", err)
		return err
	}

	// Fire webhooks that are subscribed to this event.
	for _, wh := range m.webhooks {
		if !m.isSubscribed(wh, event) {
			continue
		}

		// Deliver asynchronously.
		go m.deliver(wh, event, payloadBytes)
	}

	return nil
}

// isSubscribed checks if a webhook is subscribed to the given event.
func (m *Manager) isSubscribed(wh Webhook, event string) bool {
	_, exists := wh.Events[event]
	return exists
}

// deliver attempts to deliver a webhook with retries.
func (m *Manager) deliver(wh Webhook, event string, payload []byte) {
	var lastErr error

	for attempt := 0; attempt <= wh.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, 8s, ...
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			time.Sleep(backoff)
		}

		err := m.send(wh, event, payload)
		if err == nil {
			if attempt > 0 {
				m.log.Printf("webhook %s delivered after %d retries", wh.Name, attempt)
			}
			return
		}

		lastErr = err
		m.log.Printf("webhook %s delivery attempt %d failed: %v", wh.Name, attempt+1, err)
	}

	m.log.Printf("webhook %s delivery failed after %d attempts: %v", wh.Name, wh.MaxRetries+1, lastErr)
}

// send makes an HTTP request to deliver the webhook.
func (m *Manager) send(wh Webhook, event string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Set headers.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("listmonk/%s", m.versionString))
	req.Header.Set("X-Listmonk-Event", event)

	// Apply authentication.
	switch wh.AuthType {
	case models.WebhookAuthTypeBasic:
		req.SetBasicAuth(wh.AuthBasicUser, wh.AuthBasicPass)

	case models.WebhookAuthTypeHMAC:
		timestamp := time.Now().Unix()
		signature := m.computeHMAC(payload, wh.AuthHMACSecret, timestamp)
		req.Header.Set("X-Listmonk-Signature", signature)
		req.Header.Set("X-Listmonk-Timestamp", fmt.Sprintf("%d", timestamp))
	}

	// Create a client with the specific timeout.
	client := &http.Client{Timeout: wh.Timeout}

	// Make the request.
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	// Check if delivery was successful (2xx status).
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

// computeHMAC computes the HMAC-SHA256 signature for the payload.
func (m *Manager) computeHMAC(payload []byte, secret string, timestamp int64) string {
	// Signature is computed as HMAC-SHA256(timestamp.payload, secret)
	data := fmt.Sprintf("%d.%s", timestamp, string(payload))
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// Close is a no-op for the settings-based manager.
func (m *Manager) Close() {
	// No cleanup needed for settings-based manager.
}
