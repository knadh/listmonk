package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_3_0 adds per-list welcome e-mail configuration and the subscriber_welcomes
// dedup table used to guarantee a welcome is sent at most once per (subscriber, list).
func V6_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Per-list welcome e-mail columns. Idempotent.
	if _, err := db.Exec(`
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS welcome_enabled BOOLEAN NOT NULL DEFAULT false;
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS welcome_subject TEXT NOT NULL DEFAULT '';
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS welcome_content_type content_type NOT NULL DEFAULT 'richtext';
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS welcome_body TEXT NOT NULL DEFAULT '';
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS welcome_body_source TEXT NULL;
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS welcome_template_id INTEGER NULL REFERENCES templates(id) ON DELETE SET NULL;
	`); err != nil {
		return err
	}

	// Dedup table for welcome sends. Idempotent.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS subscriber_welcomes (
			subscriber_id INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE ON UPDATE CASCADE,
			list_id       INTEGER NOT NULL REFERENCES lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
			created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY(subscriber_id, list_id)
		);
	`); err != nil {
		return err
	}

	return nil
}
