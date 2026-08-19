package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
	_ "github.com/lib/pq"
)

const deliveryEventsTestSchema = `
CREATE TABLE campaigns (
	id SERIAL PRIMARY KEY,
	uuid UUID NOT NULL UNIQUE
);
CREATE TABLE subscribers (
	id SERIAL PRIMARY KEY,
	uuid UUID NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	status TEXT NOT NULL
);
CREATE TABLE subscriber_lists (
	subscriber_id INTEGER NOT NULL REFERENCES subscribers(id),
	status TEXT NOT NULL,
	updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE TABLE bounces (
	id SERIAL PRIMARY KEY,
	subscriber_id INTEGER NOT NULL REFERENCES subscribers(id),
	campaign_id INTEGER NULL REFERENCES campaigns(id),
	type TEXT NOT NULL,
	source TEXT NOT NULL,
	meta JSONB NOT NULL DEFAULT '{}',
	created_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE TABLE campaign_delivery_events (
	id BIGSERIAL PRIMARY KEY,
	campaign_id INTEGER NULL REFERENCES campaigns(id),
	subscriber_id INTEGER NULL REFERENCES subscribers(id),
	provider TEXT NOT NULL,
	provider_event_id TEXT NOT NULL,
	provider_message_id TEXT NOT NULL DEFAULT '',
	event_type TEXT NOT NULL,
	meta JSONB NOT NULL DEFAULT '{}',
	occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
	UNIQUE(provider, provider_event_id)
);`

func loadNamedTestQuery(t *testing.T, filename, name string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "..", "..", "queries", filename))
	if err != nil {
		t.Fatalf("read query file: %v", err)
	}

	marker := "-- name: " + name
	contents := string(b)
	start := strings.Index(contents, marker)
	if start < 0 {
		t.Fatalf("query %q not found in %s", name, filename)
	}
	contents = contents[start+len(marker):]
	if end := strings.Index(contents, "\n-- name: "); end >= 0 {
		contents = contents[:end]
	}
	return strings.TrimSpace(contents)
}

func newDeliveryEventsTestCore(t *testing.T) (*Core, *sqlx.DB) {
	t.Helper()

	dsn := os.Getenv("LISTMONK_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("LISTMONK_TEST_DB_DSN is not set")
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}
	db.SetMaxOpenConns(1)

	schema := fmt.Sprintf("delivery_events_test_%d", time.Now().UnixNano())
	if _, err := db.Exec(`CREATE SCHEMA ` + schema); err != nil {
		db.Close()
		t.Fatalf("create test schema: %v", err)
	}
	if _, err := db.Exec(`SET search_path TO ` + schema); err != nil {
		db.Close()
		t.Fatalf("select test schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`)
		_ = db.Close()
	})

	if _, err := db.Exec(deliveryEventsTestSchema); err != nil {
		t.Fatalf("create delivery event test tables: %v", err)
	}

	eventStmt, err := db.Preparex(loadNamedTestQuery(t, "campaigns.sql", "record-campaign-delivery-event"))
	if err != nil {
		t.Fatalf("prepare delivery event query: %v", err)
	}
	bounceStmt, err := db.Preparex(loadNamedTestQuery(t, "bounces.sql", "record-bounce"))
	if err != nil {
		t.Fatalf("prepare bounce query: %v", err)
	}

	queries := &models.Queries{
		RecordCampaignDeliveryEvent: eventStmt,
		RecordBounce:                bounceStmt,
	}
	co := New(&Opt{
		Constants: Constants{BounceActions: map[string]struct {
			Count  int
			Action string
		}{
			models.BounceTypeHard: {Count: 1, Action: "blocklist"},
		}},
		DB:      db,
		Queries: queries,
		Log:     log.New(io.Discard, "", 0),
	}, &Hooks{})

	return co, db
}

func TestRecordCampaignDeliveryEventsTransactionalAndIdempotent(t *testing.T) {
	co, db := newDeliveryEventsTestCore(t)

	const (
		campaignUUID   = "d6da0074-1084-4aa1-9c62-65fde375d33c"
		subscriberUUID = "07f04382-c46a-4f36-839c-9a8cb907ff22"
	)
	if _, err := db.Exec(`INSERT INTO campaigns (uuid) VALUES ($1)`, campaignUUID); err != nil {
		t.Fatalf("insert campaign: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO subscribers (uuid, email, status) VALUES ($1, $2, 'enabled')`,
		subscriberUUID, "bounce@example.com"); err != nil {
		t.Fatalf("insert subscriber: %v", err)
	}

	bounce := models.CampaignDeliveryEvent{
		Provider:        "sendgrid",
		ProviderEventID: "bounce-event-1",
		EventType:       models.DeliveryEventBounce,
		CampaignUUID:    campaignUUID,
		SubscriberUUID:  subscriberUUID,
		Email:           "bounce@example.com",
		BounceType:      models.BounceTypeHard,
		Meta:            json.RawMessage(`{"reason":"invalid mailbox"}`),
		OccurredAt:      time.Now().UTC(),
	}

	result, err := co.RecordCampaignDeliveryEvents([]models.CampaignDeliveryEvent{bounce})
	if err != nil {
		t.Fatalf("record bounce event: %v", err)
	}
	if result.Inserted != 1 || result.Duplicates != 0 {
		t.Fatalf("first result = %#v, want one insertion", result)
	}

	result, err = co.RecordCampaignDeliveryEvents([]models.CampaignDeliveryEvent{bounce})
	if err != nil {
		t.Fatalf("record duplicate bounce event: %v", err)
	}
	if result.Inserted != 0 || result.Duplicates != 1 {
		t.Fatalf("duplicate result = %#v, want one duplicate", result)
	}

	var bounceCount int
	if err := db.Get(&bounceCount, `SELECT COUNT(*) FROM bounces`); err != nil {
		t.Fatalf("count bounces: %v", err)
	}
	if bounceCount != 1 {
		t.Fatalf("bounce workflow ran %d times, want once", bounceCount)
	}
	var status string
	if err := db.Get(&status, `SELECT status FROM subscribers WHERE uuid = $1`, subscriberUUID); err != nil {
		t.Fatalf("get subscriber status: %v", err)
	}
	if status != "blocklisted" {
		t.Fatalf("subscriber status = %q, want blocklisted", status)
	}

	rollbackBatch := []models.CampaignDeliveryEvent{
		{
			Provider: "sendgrid", ProviderEventID: "rollback-delivered",
			EventType: models.DeliveryEventDelivered, CampaignUUID: campaignUUID,
			SubscriberUUID: subscriberUUID, Email: "bounce@example.com",
			Meta: json.RawMessage(`{}`), OccurredAt: time.Now().UTC(),
		},
		{
			Provider: "sendgrid", ProviderEventID: "rollback-invalid-bounce",
			EventType: models.DeliveryEventBounce, CampaignUUID: campaignUUID,
			SubscriberUUID: subscriberUUID, Email: "bounce@example.com",
			BounceType: "invalid", Meta: json.RawMessage(`{}`), OccurredAt: time.Now().UTC(),
		},
	}
	if _, err := co.RecordCampaignDeliveryEvents(rollbackBatch); err == nil {
		t.Fatal("invalid bounce batch unexpectedly succeeded")
	}
	var rollbackCount int
	if err := db.Get(&rollbackCount, `
		SELECT COUNT(*) FROM campaign_delivery_events
		WHERE provider_event_id IN ('rollback-delivered', 'rollback-invalid-bounce')`); err != nil {
		t.Fatalf("count rolled back events: %v", err)
	}
	if rollbackCount != 0 {
		t.Fatalf("transaction retained %d rolled back events", rollbackCount)
	}
}

func TestRecordCampaignDeliveryEventsUnattributedBounceFallback(t *testing.T) {
	co, db := newDeliveryEventsTestCore(t)

	const subscriberUUID = "332b7d36-2d0e-44a6-a099-9a26cb5ff32e"
	if _, err := db.Exec(`INSERT INTO subscribers (uuid, email, status) VALUES ($1, $2, 'enabled')`,
		subscriberUUID, "fallback@example.com"); err != nil {
		t.Fatalf("insert subscriber: %v", err)
	}

	event := models.CampaignDeliveryEvent{
		Provider:        "sendgrid",
		ProviderEventID: "unattributed-bounce-1",
		EventType:       models.DeliveryEventBounce,
		Email:           "fallback@example.com",
		BounceType:      models.BounceTypeHard,
		Meta:            json.RawMessage(`{}`),
		OccurredAt:      time.Now().UTC(),
	}
	result, err := co.RecordCampaignDeliveryEvents([]models.CampaignDeliveryEvent{event})
	if err != nil {
		t.Fatalf("record unattributed bounce: %v", err)
	}
	if result.Inserted != 1 || result.Unattributed != 1 {
		t.Fatalf("unattributed result = %#v", result)
	}

	var campaignIsNull bool
	if err := db.Get(&campaignIsNull, `
		SELECT campaign_id IS NULL FROM campaign_delivery_events
		WHERE provider_event_id = 'unattributed-bounce-1'`); err != nil {
		t.Fatalf("get unattributed event: %v", err)
	}
	if !campaignIsNull {
		t.Fatal("unattributed event unexpectedly has a campaign")
	}
	var status string
	if err := db.Get(&status, `SELECT status FROM subscribers WHERE uuid = $1`, subscriberUUID); err != nil {
		t.Fatalf("get fallback subscriber status: %v", err)
	}
	if status != "blocklisted" {
		t.Fatalf("fallback subscriber status = %q, want blocklisted", status)
	}
}
