package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V1_0_1 performs the DB migrations for v.1.2.0.
func V1_0_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('appearance.custom_css', '"/* custom.css */"'),
			('activeTab', '""')
			ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
