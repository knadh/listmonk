package migrations

import (
	"io"
	"log"
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/knadh/listmonk/internal/testdb"
)

// Upgrading a database without the campaigns.segment_query column adds it with an
// empty default, so that existing campaigns are sent to all their subscribers like before.
// Migrations are re-run on non-final builds and should be idempotent.
func TestV7_0_0CampaignSegment(t *testing.T) {
	db, _ := testdb.New(t)

	// Simulate a pre-v7 schema with an existing campaign.
	testdb.Exec(t, db, "ALTER TABLE campaigns DROP COLUMN segment_query",
		`INSERT INTO campaigns (uuid, name, subject, from_email, body, messenger)
			VALUES (gen_random_uuid(), 'old', 'subject', 'from@example.com', 'body', 'email')`)

	for range 2 {
		if err := V7_0_0(db, nil, koanf.New("."), log.New(io.Discard, "", 0)); err != nil {
			t.Fatalf("error running migration: %v", err)
		}
	}

	var q string
	if err := db.Get(&q, "SELECT segment_query FROM campaigns WHERE name='old'"); err != nil || q != "" {
		t.Errorf("existing campaign segment_query = %q (%v); want empty", q, err)
	}
}
