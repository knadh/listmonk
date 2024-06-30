package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V3_1_0 performs the DB migrations.
func V3_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {

	if _, err := db.Exec(`
		ALTER TABLE lists ADD COLUMN IF NOT EXISTS welcome_template_id INTEGER NULL
			REFERENCES templates(id) ON DELETE SET NULL ON UPDATE CASCADE;
	`); err != nil {
		return err
	}

	return nil
}
