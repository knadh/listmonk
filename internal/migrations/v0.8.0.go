package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_8_0 performs the DB migrations for v.0.8.0.
func V0_8_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	_, err := db.Exec(`
	INSERT INTO settings (key, value) VALUES ('privacy.individual_tracking', 'false')
		ON CONFLICT DO NOTHING;
	INSERT INTO settings (key, value) VALUES ('messengers', '[]')
		ON CONFLICT DO NOTHING;

	-- Link clicks shouldn't exist if there's no corresponding link.
	-- links_clicks.link_id should have been NOT NULL originally.
	DELETE FROM link_clicks WHERE link_id is NULL;
	ALTER TABLE link_clicks ALTER COLUMN link_id SET NOT NULL;
	`)
	return err
}
