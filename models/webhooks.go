package models

import (
	"time"
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
