package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V5_1_0 performs the DB migrations.
func V5_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Update OIDC settings to include auto_create_users and default_user_role_id fields
	if _, err := db.Exec(`
		UPDATE settings 
		SET value = jsonb_set(
			jsonb_set(
				value::jsonb,
				'{auto_create_users}',
				'false'::jsonb
			),
			'{default_user_role_id}',
			'0'::jsonb
		)
		WHERE key = 'security.oidc' AND NOT value::jsonb ? 'auto_create_users';
	`); err != nil {
		return err
	}

	return nil
}