package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gopkg.in/volatiletech/null.v6"
)

// Webhook event types.
const (
	// Subscriber events.
	EventSubscriberCreated     = "subscriber.created"
	EventSubscriberUpdated     = "subscriber.updated"
	EventSubscriberDeleted     = "subscriber.deleted"
	EventSubscriberOptinStart  = "subscriber.optin_start"
	EventSubscriberOptinFinish = "subscriber.optin_finish"

	// Subscription events.
	EventSubscriberAddedToList     = "subscriber.added_to_list"
	EventSubscriberRemovedFromList = "subscriber.removed_from_list"
	EventSubscriberUnsubscribed    = "subscriber.unsubscribed"

	// Bounce events.
	EventSubscriberBounced = "subscriber.bounced"

	// Campaign events.
	EventCampaignStarted   = "campaign.started"
	EventCampaignPaused    = "campaign.paused"
	EventCampaignCancelled = "campaign.cancelled"
	EventCampaignFinished  = "campaign.finished"
)

// Webhook auth types.
const (
	WebhookAuthTypeNone  = "none"
	WebhookAuthTypeBasic = "basic"
	WebhookAuthTypeHMAC  = "hmac"
)

// Webhook log status types.
const (
	WebhookLogStatusTriggered  = "triggered"
	WebhookLogStatusProcessing = "processing"
	WebhookLogStatusCompleted  = "completed"
	WebhookLogStatusFailed     = "failed"
)

// Webhook is the configured endpoint to send events to.
type Webhook struct {
	UUID           string   `json:"uuid"`
	Enabled        bool     `json:"enabled"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	Events         []string `json:"events"`
	AuthType       string   `json:"auth_type"`
	AuthBasicUser  string   `json:"auth_basic_user"`
	AuthBasicPass  string   `json:"auth_basic_pass,omitempty"`
	AuthHMACSecret string   `json:"auth_hmac_secret,omitempty"`
	MaxRetries     int      `json:"max_retries"`
	Timeout        string   `json:"timeout"`
}

// WebhookEvent represents an event payload to be sent to webhooks.
type WebhookEvent struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// WebhookLog represents a webhook delivery log entry.
type WebhookLog struct {
	ID            int             `db:"id" json:"id"`
	WebhookID     string          `db:"webhook_id" json:"webhook_id"`
	Event         string          `db:"event" json:"event"`
	Payload       JSON            `db:"payload" json:"payload"`
	Status        string          `db:"status" json:"status"`
	Retries       int             `db:"retries" json:"retries"`
	LastRetriedAt null.Time       `db:"last_retried_at" json:"last_retried_at"`
	Response      WebhookResponse `db:"response" json:"response"`
	Note          null.String     `db:"note" json:"note"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updated_at"`
}

// WebhookResponse stores the HTTP response details from webhook delivery.
type WebhookResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Body       string `json:"body,omitempty"`
}

// Scan implements the sql.Scanner interface for WebhookResponse.
func (r *WebhookResponse) Scan(src interface{}) error {
	if src == nil {
		*r = WebhookResponse{}
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(b, r)
}

// Value implements the driver.Valuer interface for WebhookResponse.
func (r WebhookResponse) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// AllWebhookEvents returns a list of all available webhook events.
func AllWebhookEvents() []string {
	return []string{
		EventSubscriberCreated,
		EventSubscriberUpdated,
		EventSubscriberDeleted,
		EventSubscriberOptinStart,
		EventSubscriberOptinFinish,
		EventSubscriberAddedToList,
		EventSubscriberRemovedFromList,
		EventSubscriberUnsubscribed,
		EventSubscriberBounced,
		EventCampaignStarted,
		EventCampaignPaused,
		EventCampaignCancelled,
		EventCampaignFinished,
	}
}
