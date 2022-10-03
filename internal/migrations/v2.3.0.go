package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V2_2_0 performs the DB migrations for v.2.2.0.
func V2_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`ALTER TABLE media ADD COLUMN IF NOT EXISTS "meta" JSONB NOT NULL DEFAULT '{}'`); err != nil {
		return err
	}

	return nil
}
