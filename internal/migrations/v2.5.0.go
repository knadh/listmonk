package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_5_0 performs the DB migrations.
func V2_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Insert new preference settings.
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
 			('upload.extensions', '["jpg","jpeg","png","gif","svg","*"]'),
 			('app.enable_public_archive_rss_content', 'false'),
 			('bounce.actions', '{"soft": {"count": 2, "action": "none"}, "hard": {"count": 2, "action": "blocklist"}, "complaint" : {"count": 2, "action": "blocklist"}}'),
			('privacy.record_optin_ip', 'false')
 			ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		DELETE FROM settings WHERE key IN ('bounce.count', 'bounce.action');

		-- Add the content_type column.
		ALTER TABLE media ADD COLUMN IF NOT EXISTS content_type TEXT NOT NULL DEFAULT 'application/octet-stream';

		-- Add meta column to subscriptions.
		ALTER TABLE subscriber_lists ADD COLUMN IF NOT EXISTS meta JSONB NOT NULL DEFAULT '{}';

		-- Fill the content type column for existing files (which would only be images at this point).
		UPDATE media SET content_type = CASE
			WHEN LOWER(SUBSTRING(filename FROM '.([^.]+)$')) = 'svg' THEN 'image/svg+xml'
				ELSE 'image/' || LOWER(SUBSTRING(filename FROM '.([^.]+)$'))
			END;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS campaign_media (
		    campaign_id  INTEGER REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,

		    -- Media items may be deleted, so media_id is nullable
		    -- and a copy of the original name is maintained here.
		    media_id     INTEGER NULL REFERENCES media(id) ON DELETE SET NULL ON UPDATE CASCADE,

		    filename     TEXT NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_camp_media_id ON campaign_media (campaign_id, media_id);
		CREATE INDEX IF NOT EXISTS idx_camp_media_camp_id ON campaign_media(campaign_id);
	`); err != nil {
		return err
	}

	return nil
}
