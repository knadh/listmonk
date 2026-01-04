package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES ('security.cors_origins', '[]', NOW()) ON CONFLICT (key) DO NOTHING
	`)
	if err != nil {
		return err
	}

	// Add 2FA fields to users table.
	_, err = db.Exec(`
		DO $$ BEGIN
			-- Create twofa_type enum if it doesn't exist
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'twofa_type') THEN
				CREATE TYPE twofa_type AS ENUM ('none', 'totp');
			END IF;
		END $$;

		ALTER TABLE users ADD COLUMN IF NOT EXISTS twofa_type twofa_type NOT NULL DEFAULT 'none';
		ALTER TABLE users ADD COLUMN IF NOT EXISTS twofa_key TEXT NULL;
	`)
	if err != nil {
		return err
	}

	// Add status field to lists table.
	_, err = db.Exec(`
		DO $$ BEGIN
			-- Create list_status enum if it doesn't exist
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'list_status') THEN
				CREATE TYPE list_status AS ENUM ('active', 'archived');
			END IF;
		END $$;

		ALTER TABLE lists ADD COLUMN IF NOT EXISTS status list_status NOT NULL DEFAULT 'active';
		CREATE INDEX IF NOT EXISTS idx_lists_status ON lists(status);
	`)
	if err != nil {
		return err
	}

	// Add attribs field to campaigns table.
	_, err = db.Exec(`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS attribs JSONB NOT NULL DEFAULT '{}'`)
	if err != nil {
		return err
	}

	return nil
}
