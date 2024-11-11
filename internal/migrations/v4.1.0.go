package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V4_1_0 performs the DB migrations.
func V4_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Insert new preference settings.
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES('bounce.forwardemail', '{"enabled": false, "key": ""}') ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
