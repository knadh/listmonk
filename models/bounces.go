package models

import (
	"encoding/json"
	"time"
)

const (
	BounceTypeHard      = "hard"
	BounceTypeSoft      = "soft"
	BounceTypeComplaint = "complaint"
)

// Bounce represents a single bounce event.
type Bounce struct {
	ID        int             `db:"id" json:"id"`
	Type      string          `db:"type" json:"type"`
	Source    string          `db:"source" json:"source"`
	Meta      json.RawMessage `db:"meta" json:"meta"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`

	// One of these should be provided.
	Email            string `db:"email" json:"email,omitempty"`
	SubscriberUUID   string `db:"subscriber_uuid" json:"subscriber_uuid,omitempty"`
	SubscriberID     int    `db:"subscriber_id" json:"subscriber_id,omitempty"`
	SubscriberStatus string `db:"subscriber_status" json:"subscriber_status"`

	CampaignUUID string           `db:"campaign_uuid" json:"campaign_uuid,omitempty"`
	Campaign     *json.RawMessage `db:"campaign" json:"campaign"`

	// Pseudofield for getting the total number of bounces
	// in searches and queries.
	Total int `db:"total" json:"-"`
}
