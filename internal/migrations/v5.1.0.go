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

	// Migrate old captcha settings to new JSON structure.
	_, err = db.Exec(`
		WITH old AS (
			SELECT 
				COALESCE((SELECT (value#>>'{}')::BOOLEAN FROM settings WHERE key = 'security.enable_captcha'), false) AS enable_captcha,
				COALESCE((SELECT value#>>'{}' FROM settings WHERE key = 'security.captcha_key'), '') AS captcha_key,
				COALESCE((SELECT value#>>'{}' FROM settings WHERE key = 'security.captcha_secret'), '') AS captcha_secret
		)
		INSERT INTO settings (key, value, updated_at) 
		SELECT 
			'security.captcha',
			JSON_BUILD_OBJECT(
				'altcha', JSON_BUILD_OBJECT('enabled', false, 'complexity', 300000),
				'hcaptcha', JSON_BUILD_OBJECT('enabled', enable_captcha, 'key', captcha_key, 'secret', captcha_secret)
			),
			NOW()
		FROM old
		ON CONFLICT (key) DO NOTHING
	`)
	if err != nil {
		return err
	}

	// Remove old captcha settings.
	if _, err = db.Exec(`DELETE FROM settings WHERE key IN ('security.enable_captcha', 'security.captcha_key', 'security.captcha_secret')`); err != nil {
		return err
	}

	return nil
}
