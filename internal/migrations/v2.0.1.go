package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V2_0_1 performs the DB migrations for v.2.0.1.
func V2_0_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
 			('admin.custom_css', '""')
 			ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
