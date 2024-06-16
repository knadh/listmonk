package migrations

import (
	"encoding/json"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
)

// V3_1_0 performs the DB migrations.
func V3_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		CREATE EXTENSION IF NOT EXISTS pgcrypto;

		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_type') THEN
			CREATE TYPE user_type AS ENUM ('user', 'super', 'api');
			END IF;

			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
			CREATE TYPE user_status AS ENUM ('enabled', 'disabled');
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

		CREATE TABLE IF NOT EXISTS user_roles (
		    id               SERIAL PRIMARY KEY,
		    name             TEXT NOT NULL DEFAULT '',
		    permissions      TEXT[] NOT NULL DEFAULT '{}',
		    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_name ON user_roles(LOWER(name));

		CREATE TABLE IF NOT EXISTS sessions (
		    id TEXT NOT NULL PRIMARY KEY,
		    data jsonb DEFAULT '{}'::jsonb NOT NULL,
		    created_at timestamp without time zone DEFAULT now() NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_sessions ON sessions (id, created_at);
	`); err != nil {
		return err
	}

	// Insert new preference settings.
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES('security.oidc', '{"enabled": false, "provider_url": "", "client_id": "", "client_secret": ""}') ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	// Insert superuser role.
	pmRaw, err := fs.Read("/permissions.json")
	if err != nil {
		lo.Fatalf("error reading permissions file: %v", err)
	}
	permGroups := []struct {
		Group       string   `json:"group"`
		Permissions []string `json:"permissions"`
	}{}
	if err := json.Unmarshal(pmRaw, &permGroups); err != nil {
		lo.Fatalf("error loading permissions file: %v", err)
	}

	perms := []string{}
	for _, group := range permGroups {
		for _, p := range group.Permissions {
			perms = append(perms, p)
		}
	}
	if _, err := db.Exec(`INSERT INTO roles (id, name, permissions) VALUES(1, 'Super Admin', $1) ON CONFLICT DO NOTHING`, pq.Array(perms)); err != nil {
		return err
	}

	// Create super admin.
	var (
		user     = ko.String("app.admin_username")
		password = ko.String("app.admin_password")
	)
	if len(user) < 2 || len(password) < 8 {
		lo.Fatal("admin_username should be min 3 chars and admin_password should be min 8 chars in the config file")
	}

	if _, err := db.Exec(`
		INSERT INTO users (username, password_login, password, email, name, type, status) VALUES($1, true, CRYPT($2, GEN_SALT('bf')), $3, $4, 'super', 'enabled') ON CONFLICT DO NOTHING;
	`, user, password, user+"@listmonk", user); err != nil {
		return err
	}

	return nil
}
