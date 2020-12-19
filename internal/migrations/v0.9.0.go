package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V0_9_0 performs the DB migrations for v.0.9.0.
func V0_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
	INSERT INTO settings (key, value) VALUES ('app.lang', '"en"')
		ON CONFLICT DO NOTHING;
	`)
	return err
}
