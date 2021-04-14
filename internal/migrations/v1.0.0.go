package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V1_0_0 performs the DB migrations for v.1.0.0.
func V1_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TYPE content_type ADD VALUE IF NOT EXISTS 'markdown'`)
	return err
}
