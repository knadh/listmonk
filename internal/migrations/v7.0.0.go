package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V7_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// OIDC claim mapping.
	if _, err := db.Exec(`UPDATE settings SET value = JSONB_SET(value, '{roles}', '[]'::JSONB) WHERE key = 'security.oidc' AND NOT (value ? 'roles');`); err != nil {
		return err
	}

	return nil
}
