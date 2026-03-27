package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_4_0 adds the 'emailmd' value to the content_type enum.
// NOTE: The --install flow uses schema.sql and then records the latest
// migration version without running migrations. Ensure that schema.sql
// defines content_type with the 'emailmd' value (or adjust install to run
// pending migrations) so fresh installs match the migrated schema.
func V6_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`ALTER TYPE content_type ADD VALUE IF NOT EXISTS 'emailmd'`); err != nil {
		return err
	}

	return nil
}
