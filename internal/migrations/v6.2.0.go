package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES ('app.campaign_subject_prefix', '""', NOW()) ON CONFLICT (key) DO NOTHING
	`); err != nil {
		return err
	}

	return nil
}
