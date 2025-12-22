// Package webhooks implements outgoing webhook functionality for listmonk events.
package webhooks

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Config represents the configuration for a webhook endpoint.
type Config struct {
	Enabled     bool                `json:"enabled"`
	URL         string              `json:"url"`
	AuthType    string              `json:"auth_type"` // none, basic, bearer
	Username    string              `json:"username"`
	Password    string              `json:"password,omitempty"`
	BearerToken string              `json:"bearer_token,omitempty"`
	Headers     []map[string]string `json:"headers"`
	Timeout     string              `json:"timeout"`
	MaxRetries  int                 `json:"max_retries"`
}

// Client handles sending webhook requests.
type Client struct {
	httpClient *http.Client
	log        *log.Logger
	configs    map[EventType]Config
}

// New creates a new webhook client.
func New(lo *log.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   10,
				MaxConnsPerHost:       10,
				IdleConnTimeout:       30 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
			},
		},
		log:     lo,
		configs: make(map[EventType]Config),
	}
}

// SetConfig sets the configuration for a specific event type.
func (c *Client) SetConfig(eventType EventType, cfg Config) {
	c.configs[eventType] = cfg

	// Update HTTP client timeout based on config.
	if cfg.Timeout != "" {
		if d, err := time.ParseDuration(cfg.Timeout); err == nil {
			c.httpClient.Timeout = d
		}
	}
}

// IsEnabled checks if webhooks are enabled for a specific event type.
func (c *Client) IsEnabled(eventType EventType) bool {
	cfg, ok := c.configs[eventType]
	return ok && cfg.Enabled && cfg.URL != ""
}

// Fire sends a webhook for the given event asynchronously.
// It fires in a goroutine and logs any errors.
func (c *Client) Fire(event Event) {
	cfg, ok := c.configs[event.Event]
	if !ok || !cfg.Enabled || cfg.URL == "" {
		return
	}

	go func() {
		if err := c.send(cfg, event); err != nil {
			c.log.Printf("webhook error (%s): %v", event.Event, err)
		}
	}()
}

// FireSync sends a webhook synchronously and returns any error.
// Useful for testing webhooks.
func (c *Client) FireSync(event Event) error {
	cfg, ok := c.configs[event.Event]
	if !ok || !cfg.Enabled || cfg.URL == "" {
		return fmt.Errorf("webhook not configured for event type: %s", event.Event)
	}

	return c.send(cfg, event)
}

// send executes the HTTP POST with retries.
func (c *Client) send(cfg Config, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, 8s, ...
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			time.Sleep(backoff)
		}

		if err := c.doRequest(cfg, payload); err != nil {
			lastErr = err
			c.log.Printf("webhook attempt %d/%d failed for %s: %v", attempt+1, maxRetries, event.Event, err)
			continue
		}

		return nil // Success
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", maxRetries, lastErr)
}

// doRequest performs a single HTTP POST request.
func (c *Client) doRequest(cfg Config, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "listmonk")

	// Set authentication.
	switch cfg.AuthType {
	case "basic":
		if cfg.Username != "" {
			authStr := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))
			req.Header.Set("Authorization", "Basic "+authStr)
		}
	case "bearer":
		if cfg.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.BearerToken)
		}
	}

	// Set custom headers.
	for _, h := range cfg.Headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}

	// Execute request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	// Check response status.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("non-success status code: %d", resp.StatusCode)
	}

	return nil
}
