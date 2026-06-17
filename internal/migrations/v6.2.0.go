package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_2_0 performs the DB migrations.
func V6_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Add a log column to subscribers to record the campaigns that were actually
	// sent to them along with the timestamp of the send.
	if _, err := db.Exec(`
		ALTER TABLE subscribers ADD COLUMN IF NOT EXISTS campaigns_sent JSONB NOT NULL DEFAULT '[]';
	`); err != nil {
		return err
	}

	return nil
}
