package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_5_0 performs the DB migrations.
func V2_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Insert new preference settings.
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
 			('app.enable_public_archive_rss_content', 'false')
 			ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
