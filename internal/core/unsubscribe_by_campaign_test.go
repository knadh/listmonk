package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	_ "github.com/lib/pq"
)

// dummyCampaignUUID is the campaign id opt-in mail puts in List-Unsubscribe.
// It matches cmd.dummyUUID. No campaign row uses it.
const dummyCampaignUUID = "00000000-0000-0000-0000-000000000000"

func TestUnsubscribeByCampaignOneClickCancelsPendingOptin(t *testing.T) {
	db, unsub := newUnsubscribeDB(t)

	subUUID := "11111111-1111-1111-1111-111111111111"
	pendingList := insertList(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Pending")
	otherPending := insertList(t, db, "cccccccc-cccc-cccc-cccc-cccccccccccc", "Other pending")
	keptList := insertList(t, db, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Already confirmed")
	subID := insertSubscriber(t, db, subUUID, "optin@example.com", "enabled")
	insertSubscription(t, db, subID, pendingList, "unconfirmed")
	insertSubscription(t, db, subID, otherPending, "unconfirmed")
	insertSubscription(t, db, subID, keptList, "confirmed")

	if _, err := db.Exec(unsub, dummyCampaignUUID, subUUID, false); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}

	if got := subscriptionStatus(t, db, subID, pendingList); got != "unsubscribed" {
		t.Fatalf("pending opt-in status = %q, want unsubscribed", got)
	}
	if got := subscriptionStatus(t, db, subID, otherPending); got != "unsubscribed" {
		t.Fatalf("other pending opt-in status = %q, want unsubscribed", got)
	}
	if got := subscriptionStatus(t, db, subID, keptList); got != "confirmed" {
		t.Fatalf("confirmed subscription status = %q, want confirmed", got)
	}
	if got := subscriberStatus(t, db, subID); got != "enabled" {
		t.Fatalf("subscriber status = %q, want enabled", got)
	}
}

func TestUnsubscribeByCampaignLeavesListsOutsideTheCampaign(t *testing.T) {
	db, unsub := newUnsubscribeDB(t)

	subUUID := "22222222-2222-2222-2222-222222222222"
	campUUID := "33333333-3333-3333-3333-333333333333"
	onCampaign := insertList(t, db, "dddddddd-dddd-dddd-dddd-dddddddddddd", "On campaign")
	notOnCampaign := insertList(t, db, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", "Not on campaign")
	subID := insertSubscriber(t, db, subUUID, "campaign@example.com", "enabled")
	insertSubscription(t, db, subID, onCampaign, "confirmed")
	insertSubscription(t, db, subID, notOnCampaign, "unconfirmed")
	insertCampaignList(t, db, campUUID, onCampaign)

	if _, err := db.Exec(unsub, campUUID, subUUID, false); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}

	if got := subscriptionStatus(t, db, subID, onCampaign); got != "unsubscribed" {
		t.Fatalf("campaign list status = %q, want unsubscribed", got)
	}
	if got := subscriptionStatus(t, db, subID, notOnCampaign); got != "unconfirmed" {
		t.Fatalf("list outside the campaign status = %q, want unconfirmed", got)
	}
	if got := subscriberStatus(t, db, subID); got != "enabled" {
		t.Fatalf("subscriber status = %q, want enabled", got)
	}
}

func TestUnsubscribeByCampaignBlocklistStillUnsubscribesEveryList(t *testing.T) {
	db, unsub := newUnsubscribeDB(t)

	subUUID := "44444444-4444-4444-4444-444444444444"
	pendingList := insertList(t, db, "ffffffff-ffff-ffff-ffff-ffffffffffff", "Pending")
	keptList := insertList(t, db, "99999999-9999-9999-9999-999999999999", "Confirmed")
	subID := insertSubscriber(t, db, subUUID, "block@example.com", "enabled")
	insertSubscription(t, db, subID, pendingList, "unconfirmed")
	insertSubscription(t, db, subID, keptList, "confirmed")

	if _, err := db.Exec(unsub, dummyCampaignUUID, subUUID, true); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}

	if got := subscriptionStatus(t, db, subID, pendingList); got != "unsubscribed" {
		t.Fatalf("pending list status = %q, want unsubscribed", got)
	}
	if got := subscriptionStatus(t, db, subID, keptList); got != "unsubscribed" {
		t.Fatalf("confirmed list status = %q, want unsubscribed", got)
	}
	if got := subscriberStatus(t, db, subID); got != "blocklisted" {
		t.Fatalf("subscriber status = %q, want blocklisted", got)
	}
}

func newUnsubscribeDB(t *testing.T) (*sqlx.DB, string) {
	t.Helper()

	dsn := os.Getenv("LISTMONK_TEST_DSN")
	if dsn == "" {
		dsn = "postgres://listmonk:listmonk@127.0.0.1:5432/listmonk_test?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("postgres not available (%v); set LISTMONK_TEST_DSN to run unsubscribe tests", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(unsubscribeTestSchema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DROP TABLE IF EXISTS campaign_lists, subscriber_lists, campaigns, lists, subscribers CASCADE;
			DROP TYPE IF EXISTS subscription_status, subscriber_status CASCADE;`)
	})

	path := filepath.Join("..", "..", "queries", "subscribers.sql")
	q, err := goyesql.ParseFile(path)
	if err != nil {
		t.Fatalf("parse queries: %v", err)
	}
	query := q["unsubscribe-by-campaign"]
	if query == nil || query.Query == "" {
		t.Fatal("unsubscribe-by-campaign query missing")
	}
	return db, query.Query
}

func insertList(t *testing.T, db *sqlx.DB, uuid, name string) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`INSERT INTO lists (uuid, name) VALUES ($1, $2) RETURNING id`, uuid, name).Scan(&id); err != nil {
		t.Fatalf("insert list: %v", err)
	}
	return id
}

func insertSubscriber(t *testing.T, db *sqlx.DB, uuid, email, status string) int {
	t.Helper()
	var id int
	if err := db.QueryRow(
		`INSERT INTO subscribers (uuid, email, name, status) VALUES ($1, $2, $3, $4) RETURNING id`,
		uuid, email, email, status,
	).Scan(&id); err != nil {
		t.Fatalf("insert subscriber: %v", err)
	}
	return id
}

func insertSubscription(t *testing.T, db *sqlx.DB, subID, listID int, status string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO subscriber_lists (subscriber_id, list_id, status) VALUES ($1, $2, $3)`,
		subID, listID, status,
	); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
}

func subscriptionStatus(t *testing.T, db *sqlx.DB, subID, listID int) string {
	t.Helper()
	var status string
	err := db.QueryRow(
		`SELECT status FROM subscriber_lists WHERE subscriber_id = $1 AND list_id = $2`,
		subID, listID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("subscription status: %v", err)
	}
	return status
}

func subscriberStatus(t *testing.T, db *sqlx.DB, subID int) string {
	t.Helper()
	var status string
	if err := db.QueryRow(`SELECT status FROM subscribers WHERE id = $1`, subID).Scan(&status); err != nil {
		t.Fatalf("subscriber status: %v", err)
	}
	return status
}

func insertCampaignList(t *testing.T, db *sqlx.DB, campUUID string, listID int) {
	t.Helper()
	var campID int
	if err := db.QueryRow(`INSERT INTO campaigns (uuid) VALUES ($1) RETURNING id`, campUUID).Scan(&campID); err != nil {
		t.Fatalf("insert campaign: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO campaign_lists (campaign_id, list_id, list_name) VALUES ($1, $2, 'list')`,
		campID, listID,
	); err != nil {
		t.Fatalf("insert campaign list: %v", err)
	}
}

const unsubscribeTestSchema = `
DROP TABLE IF EXISTS campaign_lists, subscriber_lists, campaigns, lists, subscribers CASCADE;
DROP TYPE IF EXISTS subscription_status, subscriber_status CASCADE;

CREATE TYPE subscriber_status AS ENUM ('enabled', 'disabled', 'blocklisted');
CREATE TYPE subscription_status AS ENUM ('unconfirmed', 'confirmed', 'unsubscribed');

CREATE TABLE subscribers (
    id     SERIAL PRIMARY KEY,
    uuid   uuid NOT NULL UNIQUE,
    email  TEXT NOT NULL UNIQUE,
    name   TEXT NOT NULL,
    status subscriber_status NOT NULL DEFAULT 'enabled'
);

CREATE TABLE lists (
    id   SERIAL PRIMARY KEY,
    uuid uuid NOT NULL UNIQUE,
    name TEXT NOT NULL
);

CREATE TABLE subscriber_lists (
    subscriber_id INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    list_id       INTEGER NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    status        subscription_status NOT NULL DEFAULT 'unconfirmed',
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (subscriber_id, list_id)
);

CREATE TABLE campaigns (
    id   SERIAL PRIMARY KEY,
    uuid uuid NOT NULL UNIQUE
);

CREATE TABLE campaign_lists (
    id          BIGSERIAL PRIMARY KEY,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    list_id     INTEGER REFERENCES lists(id) ON DELETE SET NULL,
    list_name   TEXT NOT NULL DEFAULT ''
);
`
