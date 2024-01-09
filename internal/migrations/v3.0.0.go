package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V3_0_0 performs the DB migrations.
func V3_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Insert new preference settings.
	if _, err := db.Exec(`INSERT INTO settings (key, value) VALUES ('bounce.postmark', '{"enabled": false, "username": "", "password": ""}') ON CONFLICT DO NOTHING;`); err != nil {
		return err
	}

	// Fix incorrect "d" (day) time prefix in S3 expiry settings.
	if _, err := db.Exec(`UPDATE settings SET value = '"167h"'  WHERE key = 'upload.s3.expiry' AND value = '"14d"'`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS archive_slug TEXT NULL UNIQUE`); err != nil {
		return err
	}

	return nil
}
