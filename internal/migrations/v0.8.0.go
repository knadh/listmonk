package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V0_8_0 performs the DB migrations for v.0.8.0.
func V0_8_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
	INSERT INTO settings (key, value) VALUES ('privacy.individual_tracking', 'false')
		ON CONFLICT DO NOTHING;
	INSERT INTO settings (key, value) VALUES ('messengers', '[]')
		ON CONFLICT DO NOTHING;
	`)
	return err
}
