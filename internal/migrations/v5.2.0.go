package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V5_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES ('security.cors_origins', '[]', NOW()) ON CONFLICT (key) DO NOTHING
	`)
	if err != nil {
		return err
	}

	return nil
}
