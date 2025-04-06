package migrations

import (
	"encoding/json"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
)

// V4_0_0 performs the DB migrations.
func V4_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("IMPORTANT: this upgrade might take a while if you have a large database. Please be patient ...")

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_subs_id_status ON subscribers(id, status);`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE EXTENSION IF NOT EXISTS pgcrypto;

		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_type') THEN
			CREATE TYPE user_type AS ENUM ('user', 'api');
			END IF;

			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
			CREATE TYPE user_status AS ENUM ('enabled', 'disabled');
			END IF;

			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'role_type') THEN
			CREATE TYPE role_type AS ENUM ('user', 'list');
			END IF;
		END$$;

		CREATE TABLE IF NOT EXISTS roles (
		    id               SERIAL PRIMARY KEY,
		    type             role_type NOT NULL DEFAULT 'user',
		    parent_id        INTEGER NULL REFERENCES roles(id) ON DELETE CASCADE ON UPDATE CASCADE,
		    list_id          INTEGER NULL REFERENCES lists(id) ON DELETE CASCADE ON UPDATE CASCADE,
		    permissions      TEXT[] NOT NULL DEFAULT '{}',
		    name             TEXT NULL,
		    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_roles ON roles (parent_id, list_id);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_name ON roles (type, name) WHERE name IS NOT NULL;

		CREATE TABLE IF NOT EXISTS users (
		    id               SERIAL PRIMARY KEY,
		    username         TEXT NOT NULL UNIQUE,
		    password_login   BOOLEAN NOT NULL DEFAULT false,
		    password         TEXT NULL,
		    email            TEXT NOT NULL UNIQUE,
		    name             TEXT NOT NULL,
		    avatar           TEXT NULL,
		    type             user_type NOT NULL DEFAULT 'user',
		    user_role_id     INTEGER NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
		    list_role_id     INTEGER NULL REFERENCES roles(id) ON DELETE CASCADE,
		    status           user_status NOT NULL DEFAULT 'disabled',
		    loggedin_at      TIMESTAMP WITH TIME ZONE NULL,
		    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS sessions (
		    id TEXT NOT NULL PRIMARY KEY,
		    data JSONB DEFAULT '{}'::jsonb NOT NULL,
		    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now() NOT NULL
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

	// Create super admin.
	var (
		user     = os.Getenv("LISTMONK_ADMIN_USER")
		password = os.Getenv("LISTMONK_ADMIN_PASSWORD")
		typ      = "env"
	)

	if user != "" {
		// If the env vars are set, use those values
		if len(user) < 2 || len(password) < 8 {
			lo.Fatal("LISTMONK_ADMIN_USER should be min 3 chars and LISTMONK_ADMIN_PASSWORD should be min 8 chars")
		}
	} else if ko.Exists("app.admin_username") {
		// Legacy admin/password are set in the config or env var. Use those.
		user = ko.String("app.admin_username")
		password = ko.String("app.admin_password")

		if len(user) < 2 || len(password) < 8 {
			lo.Fatal("admin_username should be min 3 chars and admin_password should be min 8 chars in the TOML config")
		}
		typ = "TOML config"
	}

	if user != "" && password != "" {
		lo.Printf("creating admin user '%s'. Credential source is '%s'", user, typ)

		perms := []string{}
		for _, group := range permGroups {
			perms = append(perms, group.Permissions...)
		}
		if _, err := db.Exec(`INSERT INTO roles (type, name, permissions) VALUES('user', 'Super Admin', $1) ON CONFLICT DO NOTHING`, pq.Array(perms)); err != nil {
			return err
		}

		if _, err := db.Exec(`
		INSERT INTO users (username, password_login, password, email, name, type, user_role_id, status) VALUES($1, true, CRYPT($2, GEN_SALT('bf')), $3, $4, 'user', 1, 'enabled') ON CONFLICT DO NOTHING;
	`, user, password, user+"@listmonk", user); err != nil {
			return err
		}
	} else {
		lo.Printf("no Super Admin user created. Visit webpage to create user.")
	}

	return nil
}
