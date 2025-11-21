package models

import "encoding/json"

// SubscriberExport represents a subscriber record that is exported to raw data.
type SubscriberExport struct {
	Base

	UUID    string `db:"uuid" json:"uuid"`
	Email   string `db:"email" json:"email"`
	Name    string `db:"name" json:"name"`
	Attribs string `db:"attribs" json:"attribs"`
	Status  string `db:"status" json:"status"`
}

// SubscriberExportProfile represents a subscriber's collated data in JSON for export.
type SubscriberExportProfile struct {
	Email         string          `db:"email" json:"-"`
	Profile       json.RawMessage `db:"profile" json:"profile,omitempty"`
	Subscriptions json.RawMessage `db:"subscriptions" json:"subscriptions,omitempty"`
	CampaignViews json.RawMessage `db:"campaign_views" json:"campaign_views,omitempty"`
	LinkClicks    json.RawMessage `db:"link_clicks" json:"link_clicks,omitempty"`
}
