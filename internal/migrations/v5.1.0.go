package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V5_1_0 performs the DB migrations.
func V5_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Insert new list creation permission.
	if _, err := db.Exec(`
		UPDATE roles SET permissions = permissions || '{lists:create}' WHERE permissions @> '{lists:manage_all}' AND NOT permissions @> '{lists:create}';
	`); err != nil {
		return err
	}

	// Insert new messenger sending permission.
	if _, err := db.Exec(`
		UPDATE roles SET permissions = permissions || '{messengers:get_all}' WHERE NOT permissions @> '{messengers:get_all}';
	`); err != nil {
		return err
	}

	return nil
}
