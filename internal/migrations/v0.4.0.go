package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_4_0 performs the DB migrations for v.0.4.0.
func V0_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'list_optin') THEN
			CREATE TYPE list_optin AS ENUM ('single', 'double');
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'campaign_type') THEN
			CREATE TYPE campaign_type AS ENUM ('regular', 'optin');
		END IF;
	END$$;

	ALTER TABLE lists ADD COLUMN IF NOT EXISTS optin list_optin NOT NULL DEFAULT 'single';
	ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS type campaign_type DEFAULT 'regular';
	`)
	return err
}
