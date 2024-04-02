package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V3_1_0 performs the DB migrations.
func V3_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
			CREATE TYPE user_status AS ENUM ('enabled', 'disabled', 'super');
			END IF;
		END$$;

		CREATE TABLE IF NOT EXISTS users (
		    id               SERIAL PRIMARY KEY,
		    username         TEXT NOT NULL UNIQUE,
		    password_login   BOOLEAN NOT NULL DEFAULT false,
		    password         TEXT NULL,
		    email            TEXT NOT NULL UNIQUE,
		    name             TEXT NOT NULL,
		    status           user_status NOT NULL DEFAULT 'disabled',
		    loggedin_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Insert new preference settings.
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
		('security.oidc', '{"enabled": false, "provider_url": "", "client_id": "", "client_secret": ""}'),
		ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
