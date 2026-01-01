package webhooks

import (
	"bytes"
	"context"
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

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

// WorkerConfig holds the configuration for webhook workers.
type WorkerConfig struct {
	NumWorkers int
	BatchSize  int
}

// WorkerPool manages a pool of webhook delivery workers.
type WorkerPool struct {
	cfg           WorkerConfig
	db            *sqlx.DB
	queries       *models.Queries
	webhooks      map[string]Webhook // Webhook configs indexed by UUID
	webhooksMu    sync.RWMutex
	log           *log.Logger
	versionString string

	// Control channels
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewWorkerPool creates a new webhook worker pool.
func NewWorkerPool(cfg WorkerConfig, db *sqlx.DB, queries *models.Queries, log *log.Logger, versionString string) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	if cfg.NumWorkers < 1 {
		cfg.NumWorkers = 2
	}
	if cfg.BatchSize < 1 {
		cfg.BatchSize = 50
	}

	return &WorkerPool{
		cfg:           cfg,
		db:            db,
		queries:       queries,
		webhooks:      make(map[string]Webhook),
		log:           log,
		versionString: versionString,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// LoadWebhooks loads webhook configurations into the worker pool.
func (p *WorkerPool) LoadWebhooks(settings []models.Webhook) {
	p.webhooksMu.Lock()
	defer p.webhooksMu.Unlock()

	p.webhooks = make(map[string]Webhook, len(settings))
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

		p.webhooks[s.UUID] = Webhook{
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
		}
	}

	p.log.Printf("webhook worker pool loaded %d webhooks", len(p.webhooks))
}

// Run starts the worker pool. This is a blocking call.
func (p *WorkerPool) Run() {
	// Reset any stale processing logs on startup.
	if _, err := p.queries.ResetStaleProcessingLogs.Exec(); err != nil {
		p.log.Printf("error resetting stale webhook logs: %v", err)
	}

	// Start worker goroutines.
	for i := 0; i < p.cfg.NumWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	p.log.Printf("started %d webhook workers with batch size %d", p.cfg.NumWorkers, p.cfg.BatchSize)

	// Wait for all workers to complete.
	p.wg.Wait()
}

// Close gracefully shuts down the worker pool.
func (p *WorkerPool) Close() {
	p.cancel()
	p.wg.Wait()
	p.log.Printf("webhook worker pool stopped")
}

// worker is a single worker goroutine that processes webhook logs.
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.processBatch()
		}
	}
}

// processBatch fetches and processes a batch of pending webhook logs.
func (p *WorkerPool) processBatch() {
	// Fetch a batch of pending logs (already locked with SELECT FOR UPDATE SKIP LOCKED).
	var logs []models.WebhookLog
	if err := p.queries.GetPendingWebhookLogs.Select(&logs, p.cfg.BatchSize); err != nil {
		p.log.Printf("error fetching webhook logs: %v", err)
		return
	}

	if len(logs) == 0 {
		return
	}

	// Process each log.
	for _, wl := range logs {
		p.processLog(wl)
	}
}

// processLog processes a single webhook log entry.
func (p *WorkerPool) processLog(wl models.WebhookLog) {
	// Get the webhook configuration.
	p.webhooksMu.RLock()
	wh, exists := p.webhooks[wl.WebhookID]
	p.webhooksMu.RUnlock()

	// If webhook doesn't exist (deleted), mark as failed.
	if !exists {
		resp := models.WebhookResponse{}
		note := "webhook configuration not found (may have been deleted)"
		if _, err := p.queries.UpdateWebhookLogFailed.Exec(wl.ID, resp, note); err != nil {
			p.log.Printf("error marking webhook log %d as failed: %v", wl.ID, err)
		}
		return
	}

	// Attempt delivery with retries.
	p.deliverWithRetries(wl, wh)
}

// deliverWithRetries attempts to deliver a webhook with retry logic.
func (p *WorkerPool) deliverWithRetries(wl models.WebhookLog, wh Webhook) {
	// Get the payload bytes.
	payloadBytes, err := json.Marshal(wl.Payload)
	if err != nil {
		resp := models.WebhookResponse{}
		note := fmt.Sprintf("error marshaling payload: %v", err)
		if _, err := p.queries.UpdateWebhookLogFailed.Exec(wl.ID, resp, note); err != nil {
			p.log.Printf("error marking webhook log %d as failed: %v", wl.ID, err)
		}
		return
	}

	// Calculate remaining retries.
	remainingRetries := wh.MaxRetries - wl.Retries

	for attempt := 0; attempt <= remainingRetries; attempt++ {
		// Check if context is cancelled.
		select {
		case <-p.ctx.Done():
			// Reset the log to triggered so it can be picked up again.
			if _, err := p.queries.MarkWebhookLogTriggered.Exec(wl.ID); err != nil {
				p.log.Printf("error resetting webhook log %d: %v", wl.ID, err)
			}
			return
		default:
		}

		// Apply backoff for retries.
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			time.Sleep(backoff)
		}

		// Attempt delivery.
		resp, err := p.send(wh, wl.Event, payloadBytes)
		if err == nil {
			// Success - mark as completed.
			if _, err := p.queries.UpdateWebhookLogSuccess.Exec(wl.ID, resp); err != nil {
				p.log.Printf("error marking webhook log %d as success: %v", wl.ID, err)
			}
			if attempt > 0 {
				p.log.Printf("webhook %s (log %d) delivered after %d retries", wh.Name, wl.ID, attempt)
			}
			return
		}

		// Log the failure.
		p.log.Printf("webhook %s (log %d) delivery attempt %d failed: %v", wh.Name, wl.ID, wl.Retries+attempt+1, err)

		// Update retry count.
		note := fmt.Sprintf("attempt %d failed: %v", wl.Retries+attempt+1, err)
		if _, err := p.queries.UpdateWebhookLogRetry.Exec(wl.ID, resp, note); err != nil {
			p.log.Printf("error updating webhook log %d retry: %v", wl.ID, err)
		}
	}

	// All retries exhausted - mark as failed.
	resp := models.WebhookResponse{}
	note := fmt.Sprintf("delivery failed after %d attempts", wh.MaxRetries+1)
	if _, err := p.queries.UpdateWebhookLogFailed.Exec(wl.ID, resp, note); err != nil {
		p.log.Printf("error marking webhook log %d as failed: %v", wl.ID, err)
	}
	p.log.Printf("webhook %s (log %d) delivery failed after %d attempts", wh.Name, wl.ID, wh.MaxRetries+1)
}

// send makes an HTTP request to deliver the webhook.
func (p *WorkerPool) send(wh Webhook, event string, payload []byte) (models.WebhookResponse, error) {
	resp := models.WebhookResponse{}

	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewReader(payload))
	if err != nil {
		return resp, fmt.Errorf("creating request: %w", err)
	}

	// Set headers.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("listmonk/%s", p.versionString))
	req.Header.Set("X-Listmonk-Event", event)

	// Apply authentication.
	switch wh.AuthType {
	case models.WebhookAuthTypeBasic:
		req.SetBasicAuth(wh.AuthBasicUser, wh.AuthBasicPass)

	case models.WebhookAuthTypeHMAC:
		timestamp := time.Now().Unix()
		signature := p.computeHMAC(payload, wh.AuthHMACSecret, timestamp)
		req.Header.Set("X-Listmonk-Signature", signature)
		req.Header.Set("X-Listmonk-Timestamp", fmt.Sprintf("%d", timestamp))
	}

	// Create a client with the specific timeout.
	client := &http.Client{Timeout: wh.Timeout}

	// Make the request.
	httpResp, err := client.Do(req)
	if err != nil {
		return resp, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, httpResp.Body)
		httpResp.Body.Close()
	}()

	// Read response body (limit to 1KB to avoid memory issues).
	bodyBytes, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1024))
	resp.StatusCode = httpResp.StatusCode
	resp.Body = string(bodyBytes)

	// Check if delivery was successful (2xx status).
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return resp, fmt.Errorf("non-2xx status: %d", httpResp.StatusCode)
	}

	return resp, nil
}

// computeHMAC computes the HMAC-SHA256 signature for the payload.
func (p *WorkerPool) computeHMAC(payload []byte, secret string, timestamp int64) string {
	data := fmt.Sprintf("%d.%s", timestamp, string(payload))
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
