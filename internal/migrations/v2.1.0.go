package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V2_1_0 performs the DB migrations for v.2.1.0.
func V2_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
 			('appearance.admin.custom_css', '""'),
 			('appearance.admin.custom_js', '""'),
 			('appearance.public.custom_css', '""'),
 			('appearance.public.custom_js', '""')
 			ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
