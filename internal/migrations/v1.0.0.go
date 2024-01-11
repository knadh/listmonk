package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_0_0 performs the DB migrations for v.1.0.0.
func V1_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`ALTER TYPE content_type ADD VALUE IF NOT EXISTS 'markdown'`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('app.check_updates', 'true')
			ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
