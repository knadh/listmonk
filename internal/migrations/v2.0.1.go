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
 			('appearance.admin.custom_css', '""'),
 			('appearance.public.custom_css', '""'),
 			('appearance.public.custom_js', '""')
 			ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
