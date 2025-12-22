package webhooks

import (
	"time"
	"github.com/knadh/listmonk/models"
)

type EventType string

const (
	// EventSubscriptionConfirmed is fired when a subscription is confirmed
	// (either immediately for single opt-in, or after clicking confirmation link for double opt-in).
	EventSubscriptionConfirmed EventType = "subscription.confirmed"
)

type Event struct {
	Event     EventType `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Data      EventData `json:"data"`
}

type EventData struct {
	Subscriber models.Subscriber `json:"subscriber"`
	Lists      []models.List     `json:"lists"`
	Meta       models.JSON       `json:"meta,omitempty"`
}

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
