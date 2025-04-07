package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V5_0_0 performs the DB migrations.
func V5_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("IMPORTANT: this upgrade might take a while if you have a large database. Please be patient ...")
	if _, err := db.Exec(`
		-- Create a new temp materialized view with the fixed query (removing COUNT(*) that returns 1 for NULLs) 
		CREATE MATERIALIZED VIEW IF NOT EXISTS mat_list_subscriber_stats_v5_0_0 AS
		SELECT NOW() AS updated_at, lists.id AS list_id, subscriber_lists.status, COUNT(subscriber_lists.status) AS subscriber_count FROM lists
		LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = lists.id)
		GROUP BY lists.id, subscriber_lists.status
		UNION ALL
		SELECT NOW() AS updated_at, 0 AS list_id, NULL AS status, COUNT(id) AS subscriber_count FROM subscribers;
	
		-- Drop the old view and index.
		DROP INDEX IF EXISTS mat_list_subscriber_stats_idx;
		DROP MATERIALIZED VIEW IF EXISTS mat_list_subscriber_stats;
		
		-- Rename the temp view and create an index.
		ALTER MATERIALIZED VIEW mat_list_subscriber_stats_v5_0_0 RENAME TO mat_list_subscriber_stats;
		CREATE UNIQUE INDEX IF NOT EXISTS mat_list_subscriber_stats_idx ON mat_list_subscriber_stats (list_id, status);
	`); err != nil {
		return err
	}

	// Index of media filename lookup.
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_media_filename ON media(provider, filename);
	`); err != nil {
		return err
	}

	// Insert new preference settings.
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES('privacy.domain_allowlist', '[]') ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	// Insert new default super admin permissions.
	if _, err := db.Exec(`
		UPDATE roles SET permissions = permissions || '{campaigns:get_all}' WHERE id = 1 AND NOT permissions @> '{campaigns:get_all}';
		UPDATE roles SET permissions = permissions || '{campaigns:manage_all}' WHERE id = 1 AND NOT permissions @> '{campaigns:manage_all}';
	`); err != nil {
		return err
	}

	// Visual editor changes.
	if _, err := db.Exec(`
		ALTER TYPE content_type ADD VALUE IF NOT EXISTS 'visual';
		ALTER TYPE template_type ADD VALUE IF NOT EXISTS 'campaign_visual';
		ALTER TABLE templates ADD COLUMN IF NOT EXISTS body_source TEXT NULL;
		ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS body_source TEXT NULL;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_template_id_fkey,
			ADD FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE SET NULL,
			ALTER COLUMN template_id DROP DEFAULT;

		ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_archive_template_id_fkey,
			ADD FOREIGN KEY (archive_template_id) REFERENCES templates(id) ON DELETE SET NULL,
			ALTER COLUMN archive_template_id DROP DEFAULT;
	`); err != nil {
		return err
	}

	return nil
}
