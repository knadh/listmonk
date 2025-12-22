package webhooks

import (
	"time"

	"github.com/knadh/listmonk/models"
)

// EventType represents the type of webhook event.
type EventType string

const (
	// EventSubscriptionConfirmed is fired when a subscription is confirmed
	// (either immediately for single opt-in, or after clicking confirmation link for double opt-in).
	EventSubscriptionConfirmed EventType = "subscription.confirmed"
)

// Event represents a webhook event payload.
type Event struct {
	Event     EventType `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Data      EventData `json:"data"`
}

// EventData contains the data payload for a webhook event.
type EventData struct {
	Subscriber models.Subscriber `json:"subscriber"`
	Lists      []models.List     `json:"lists"`
	Meta       models.JSON       `json:"meta,omitempty"`
}

// NewSubscriptionConfirmedEvent creates a new subscription confirmed event.
func NewSubscriptionConfirmedEvent(sub models.Subscriber, lists []models.List, meta models.JSON) Event {
	return Event{
		Event:     EventSubscriptionConfirmed,
		Timestamp: time.Now(),
		Data: EventData{
			Subscriber: sub,
			Lists:      lists,
			Meta:       meta,
		},
	}
}
