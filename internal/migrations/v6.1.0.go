package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES ('privacy.disable_tracking', 'false', NOW()) ON CONFLICT (key) DO NOTHING
	`)
	return err
}
