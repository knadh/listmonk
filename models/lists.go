package models

import (
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

const (
	ListTypePrivate    = "private"
	ListTypePublic     = "public"
	ListOptinSingle    = "single"
	ListOptinDouble    = "double"
	ListStatusActive   = "active"
	ListStatusArchived = "archived"
)

// List represents a mailing list.
type List struct {
	Base

	UUID             string         `db:"uuid" json:"uuid"`
	Name             string         `db:"name" json:"name"`
	Type             string         `db:"type" json:"type"`
	Optin            string         `db:"optin" json:"optin"`
	Status           string         `db:"status" json:"status"`
	Tags             pq.StringArray `db:"tags" json:"tags"`
	Description      string         `db:"description" json:"description"`

	// Welcome e-mail configuration. An optional message automatically sent to a subscriber
	// when they become an active member of the list (single opt-in: on subscribe; double
	// opt-in: on confirmation).
	WelcomeEnabled     bool        `db:"welcome_enabled" json:"welcome_enabled"`
	WelcomeSubject     string      `db:"welcome_subject" json:"welcome_subject"`
	WelcomeContentType string      `db:"welcome_content_type" json:"welcome_content_type"`
	WelcomeBody        string      `db:"welcome_body" json:"welcome_body"`
	WelcomeBodySource  null.String `db:"welcome_body_source" json:"welcome_body_source"`
	WelcomeTemplateID  null.Int    `db:"welcome_template_id" json:"welcome_template_id"`

	SubscriberCount  int            `db:"subscriber_count" json:"subscriber_count"`
	SubscriberCounts StringIntMap   `db:"subscriber_statuses" json:"subscriber_statuses"`
	SubscriberID     int            `db:"subscriber_id" json:"-"`

	// This is only relevant when querying the lists of a subscriber.
	SubscriptionStatus    string    `db:"subscription_status" json:"subscription_status,omitempty"`
	SubscriptionCreatedAt null.Time `db:"subscription_created_at" json:"subscription_created_at,omitempty"`
	SubscriptionUpdatedAt null.Time `db:"subscription_updated_at" json:"subscription_updated_at,omitempty"`

	// Pseudofield for getting the total number of subscribers
	// in searches and queries.
	Total int `db:"total" json:"-"`
}

// WelcomeList holds the welcome e-mail content for a single list, returned by the
// claim-welcomes query for lists whose welcome the subscriber just became eligible for.
type WelcomeList struct {
	ID           int         `db:"id"`
	Subject      string      `db:"welcome_subject"`
	ContentType  string      `db:"welcome_content_type"`
	Body         string      `db:"welcome_body"`
	TemplateID   null.Int    `db:"welcome_template_id"`
	TemplateBody null.String `db:"template_body"`
}
