package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V2_4_0 performs the DB migrations.
func V2_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Insert new preference settings.
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
 			('security.enable_captcha', 'false'),
 			('security.captcha_key', '""'),
 			('security.captcha_secret', '""')
 			ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
