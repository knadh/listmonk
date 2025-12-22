package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V5_3_0 adds webhook settings for subscription events.
func V5_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("v5.3.0: adding webhook settings...")

	_, err := db.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES ('webhooks', '{"subscription_confirmed": {"enabled": false, "url": "", "timeout": "10s", "max_retries": 3}}', NOW())
		ON CONFLICT (key) DO NOTHING
	`)
	if err != nil {
		return err
	}

	return nil
}
