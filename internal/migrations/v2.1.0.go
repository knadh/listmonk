package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V0_2_1 performs the DB migrations for v.2.1.0.
func V2_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'template_type') THEN
			CREATE TYPE template_type AS ENUM ('html', 'mjml');
		END IF;
	END$$;

	ALTER TYPE content_type ADD VALUE IF NOT EXISTS 'mjml';

	ALTER TABLE templates ADD COLUMN IF NOT EXISTS type template_type NOT NULL DEFAULT 'html';
	`)
	return err
}
