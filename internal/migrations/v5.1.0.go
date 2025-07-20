package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V5_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Update OIDC settings to include auto_create_users and default_user_role_id fields if not present
	_, err := db.Exec(`
		UPDATE settings
		SET value = value::JSONB
			|| CASE WHEN NOT (value::JSONB ? 'auto_create_users') THEN '{"auto_create_users": false}'::JSONB ELSE '{}'::JSONB END
			|| CASE WHEN NOT (value::JSONB ? 'default_user_role_id') THEN '{"default_user_role_id": null}'::JSONB ELSE '{}'::JSONB END
			|| CASE WHEN NOT (value::JSONB ? 'default_list_role_id') THEN '{"default_list_role_id": null}'::JSONB ELSE '{}'::JSONB END
		WHERE key = 'security.oidc';
	`)
	if err != nil {
		return err
	}
	return nil
}
