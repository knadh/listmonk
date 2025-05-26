package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

// Mailgun event types
const (
	mailgunEventOpened         = "opened"
	mailgunEventClicked        = "clicked"
	mailgunEventUnsubscribed   = "unsubscribed"
	mailgunEventComplained     = "complained"
	mailgunEventBounced        = "bounced"       // Legacy bounce event (permanent and temporary)
	mailgunEventFailed         = "failed"        // Newer event for permanent and temporary failures
	mailgunEventDelivered      = "delivered"
)

// mailgunSignature represents the signature POSTed by Mailgun.
type mailgunSignature struct {
	Timestamp string `json:"timestamp"`
	Token     string `json:"token"`
	Signature string `json:"signature"`
}

// mailgunEventData contains the core event information.
type mailgunEventData struct {
	Event     string    `json:"event"`
	Timestamp float64   `json:"timestamp"` // Unix timestamp
	ID        string    `json:"id"`
	Recipient string    `json:"recipient"`
	Message   mailgunMessage `json:"message"`
	Severity  string    `json:"severity"` // "permanent" or "temporary" for "failed" event
	Reason    string    `json:"reason"`   // e.g., "generic", "suppress-bounce"
	Log       string    `json:"log-level"` // "info", "warn", "error"

	// For "bounced" (legacy) event
	Code        int    `json:"code"`
	Description string `json:"description"` // Bounce description
	Error       string `json:"error"`       // Bounce error message

	// For "complained" event
	ComplaintType string `json:"complaint-type"` // e.g. "spam"

	// Custom variables
	UserVariables map[string]string `json:"user-variables"`
}

// mailgunMessage contains message specific data, like headers.
type mailgunMessage struct {
	Headers mailgunMessageHeaders `json:"headers"`
}

// mailgunMessageHeaders contains the message headers.
type mailgunMessageHeaders struct {
	MessageID string `json:"message-id"`
	To        string `json:"to"`
	From      string `json:"from"`
	Subject   string `json:"subject"`
	// We expect X-Listmonk-Campaign-UUID to be here
}

// mailgunWebhookPayload is the top-level structure of the incoming webhook.
type mailgunWebhookPayload struct {
	Signature mailgunSignature `json:"signature"`
	EventData mailgunEventData `json:"event-data"`
}

// Mailgun handles Mailgun webhook notifications.
type Mailgun struct {
	webhookSigningKey string
}

// NewMailgun returns a new Mailgun instance.
func NewMailgun(webhookSigningKey string) *Mailgun {
	return &Mailgun{
		webhookSigningKey: webhookSigningKey,
	}
}

// VerifySignature verifies the signature of an incoming Mailgun webhook.
// Documentation: https://documentation.mailgun.com/en/latest/user_manual/webhooks.html#securing-webhooks
func (m *Mailgun) VerifySignature(timestamp, token, signature string) bool {
	if m.webhookSigningKey == "" {
		// If no key is configured, skip verification (useful for testing, but not recommended for production)
		return true
	}
	if timestamp == "" || token == "" || signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(m.webhookSigningKey))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(token))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}


// ProcessBounce processes Mailgun bounce notifications and returns one or more Bounce objects.
func (m *Mailgun) ProcessBounce(requestBody []byte) ([]models.Bounce, error) {
	var payload mailgunWebhookPayload
	if err := json.Unmarshal(requestBody, &payload); err != nil {
		return nil, fmt.Errorf("error unmarshalling mailgun notification: %w", err)
	}

	// Verify signature
	if !m.VerifySignature(payload.Signature.Timestamp, payload.Signature.Token, payload.Signature.Signature) {
		return nil, fmt.Errorf("invalid mailgun webhook signature")
	}
	
	eventData := payload.EventData

	// We are interested in 'failed' (permanent and temporary) and 'complained' events.
	// 'bounced' is a legacy event but good to handle if services still use it.
	isBounceEvent := eventData.Event == mailgunEventFailed || eventData.Event == mailgunEventBounced
	isComplaintEvent := eventData.Event == mailgunEventComplained

	if !isBounceEvent && !isComplaintEvent {
		// Not an event type we're interested in for bounce processing.
		return nil, nil
	}
	
	var bounceType models.BounceType
	if isComplaintEvent {
		bounceType = models.BounceTypeComplaint
	} else { // It's a bounce/failed event
		if eventData.Event == mailgunEventFailed {
			if eventData.Severity == "permanent" {
				bounceType = models.BounceTypeHard
			} else { // "temporary" or other
				bounceType = models.BounceTypeSoft
			}
		} else { // Legacy "bounced" event
			// Mailgun's legacy "bounced" event doesn't clearly distinguish soft/hard in a structured way.
			// We might need to infer from codes or descriptions if available, or default to soft.
			// For now, let's check for common hard bounce codes or default to soft.
			// Example: SMTP error code 550 is usually a hard bounce.
			// This part might need refinement based on actual legacy "bounced" event payloads.
			if strings.HasPrefix(strconv.Itoa(eventData.Code), "5") { // 5xx codes are usually hard
				bounceType = models.BounceTypeHard
			} else {
				bounceType = models.BounceTypeSoft
			}
		}
	}

	campUUID := ""
	if val, ok := eventData.UserVariables[models.EmailHeaderCampaignUUID]; ok {
		campUUID = val
	}

	// Timestamp from event data, fallback to current time
	ts := time.Unix(int64(eventData.Timestamp), 0)
	if eventData.Timestamp == 0 {
		ts = time.Now()
	}

	bounce := models.Bounce{
		Email:        strings.ToLower(eventData.Recipient),
		CampaignUUID: campUUID,
		Type:         bounceType,
		Source:       "mailgun",
		Meta:         json.RawMessage(requestBody), // Store the whole payload as meta
		CreatedAt:    ts,
	}

	return []models.Bounce{bounce}, nil
}
