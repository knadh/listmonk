// Package webhooks implements an outgoing webhook delivery system for listmonk.
// It creates webhook log entries that are processed by background workers.
package webhooks

import (
	"encoding/json"
	"log"
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

// Manager handles webhook event triggering by creating log entries.
type Manager struct {
	webhooks      []Webhook
	log           *log.Logger
	mu            sync.RWMutex
	versionString string

	// Database query for creating webhook logs.
	createLogStmt *models.Queries
}

// New creates a new webhook manager.
func New(log *log.Logger, versionString string, queries *models.Queries) *Manager {
	return &Manager{
		webhooks:      []Webhook{},
		log:           log,
		versionString: versionString,
		createLogStmt: queries,
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

// Trigger creates webhook log entries for all webhooks subscribed to the given event.
// The logs are processed asynchronously by background workers.
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

	// Create webhook log entries for subscribed webhooks.
	for _, wh := range m.webhooks {
		if !m.isSubscribed(wh, event) {
			continue
		}

		// Create a webhook log entry.
		if _, err := m.createLogStmt.CreateWebhookLog.Exec(wh.UUID, event, payloadBytes); err != nil {
			m.log.Printf("error creating webhook log for %s: %v", wh.Name, err)
			continue
		}
	}

	return nil
}

// isSubscribed checks if a webhook is subscribed to the given event.
func (m *Manager) isSubscribed(wh Webhook, event string) bool {
	_, exists := wh.Events[event]
	return exists
}

// Close is a no-op for the settings-based manager.
func (m *Manager) Close() {
	// No cleanup needed for settings-based manager.
}
