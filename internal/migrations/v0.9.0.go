package migrations

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_9_0 performs the DB migrations for v.0.9.0.
func V0_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('app.lang', '"en"'),
			('app.message_sliding_window', 'false'),
			('app.message_sliding_window_duration', '"1h"'),
			('app.message_sliding_window_rate', '10000'),
			('app.enable_public_subscription_page', 'true')
			ON CONFLICT DO NOTHING;

		-- Add alternate (plain text) body field on campaigns.
		ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS altbody TEXT NULL DEFAULT NULL;
	`); err != nil {
		return err
	}

	// Until this version, the default template during installation was broken!
	// Check if there's a broken default template and if yes, override it with the
	// actual one.
	tplBody, err := fs.Get("/static/email-templates/default.tpl")
	if err != nil {
		return fmt.Errorf("error reading default e-mail template: %v", err)
	}

	if _, err := db.Exec(`UPDATE templates SET body=$1 WHERE body=$2`,
		tplBody.ReadBytes(), `{{ template "content" . }}`); err != nil {
		return err
	}
	return nil
}
