package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES('bounce.azure', '{"enabled": false, "shared_secret": "", "shared_secret_header": ""}')
		ON CONFLICT (key) DO UPDATE
		SET value = jsonb_build_object(
			'enabled', COALESCE((settings.value->>'enabled')::boolean, false),
			'shared_secret', COALESCE(settings.value->>'shared_secret', ''),
			'shared_secret_header', COALESCE(settings.value->>'shared_secret_header', '')
		);
	`); err != nil {
		return err
	}

	return nil
}
