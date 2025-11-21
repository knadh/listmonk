package models

import (
	"encoding/json"

	"github.com/jmoiron/sqlx/types"
	null "gopkg.in/volatiletech/null.v6"
)

const (
	SubscriptionStatusUnconfirmed  = "unconfirmed"
	SubscriptionStatusConfirmed    = "confirmed"
	SubscriptionStatusUnsubscribed = "unsubscribed"
)

// Subscription represents a list attached to a subscriber.
type Subscription struct {
	List
	SubscriptionStatus    null.String     `db:"subscription_status" json:"subscription_status"`
	SubscriptionCreatedAt null.String     `db:"subscription_created_at" json:"subscription_created_at"`
	Meta                  json.RawMessage `db:"meta" json:"meta"`
}

type subLists struct {
	SubscriberID int            `db:"subscriber_id"`
	Lists        types.JSONText `db:"lists"`
}
