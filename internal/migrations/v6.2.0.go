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

	// Add `msg_retry_delay` to each SMTP server entry in the `smtp` settings JSON array.
	// Idempotent: only updates rows where at least one entry is missing the key.
	if _, err := db.Exec(`
		UPDATE settings SET value = s.updated
		FROM (
			SELECT JSONB_AGG(
				CASE WHEN v ? 'msg_retry_delay' THEN v
				     ELSE JSONB_SET(v, '{msg_retry_delay}', '"10ms"'::JSONB)
				END
			) AS updated FROM settings, JSONB_ARRAY_ELEMENTS(value) v WHERE key = 'smtp'
		) s WHERE key = 'smtp'
		AND EXISTS (
			SELECT 1 FROM JSONB_ARRAY_ELEMENTS(value) v WHERE NOT (v ? 'msg_retry_delay')
		);
	`); err != nil {
		return err
	}

	// Update app language settings that used incorrect locale codes.
	if _, err := db.Exec(`
		UPDATE settings SET value = langs.new_value
		FROM (VALUES
			('"cs-cz"'::JSONB, '"cs"'::JSONB),
			('"jp"'::JSONB, '"ja"'::JSONB),
			('"se"'::JSONB, '"sv"'::JSONB)
		) AS langs(old_value, new_value)
		WHERE key = 'app.lang' AND value = langs.old_value;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`INSERT INTO settings (key, value) VALUES ('app.show_optin_page', 'true') ON CONFLICT (key) DO NOTHING	`); err != nil {
		return err
	}

	// Rename `security.cors_origins` to `security.trusted_urls`.
	if _, err := db.Exec(`UPDATE settings SET key = 'security.trusted_urls' WHERE key = 'security.cors_origins'`); err != nil {
		return err
	}

	return nil
}
