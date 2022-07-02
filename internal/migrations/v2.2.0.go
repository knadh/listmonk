package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V2_2_0 performs the DB migrations for v.2.2.0.
func V2_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'template_type') THEN
				CREATE TYPE template_type AS ENUM ('campaign', 'tx');
			END IF;
		END$$;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE templates ADD COLUMN IF NOT EXISTS "type" template_type NOT NULL DEFAULT 'campaign'`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE templates ADD COLUMN IF NOT EXISTS "subject" TEXT NOT NULL`); err != nil {
		return err
	}

	return nil
}
