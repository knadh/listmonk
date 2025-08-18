package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_2_0 performs the DB migrations for v6.2.0.
func V6_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
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

	// Add MJML to content_type enum if not exists.
	if _, err := db.Exec(`ALTER TYPE content_type ADD VALUE IF NOT EXISTS 'mjml';`); err != nil {
		return err
	}

	// Insert sample MJML template.
	tpl, err := fs.Get("/static/email-templates/sample-mjml.tpl")
	if err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT INTO templates (name, type, subject, body)
		SELECT $1, $2::template_type, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM templates WHERE name = $1 AND type = $2::template_type);`,
		"Sample MJML template", "campaign", "", tpl.ReadBytes()); err != nil {
		return err
	}

	return nil
}
