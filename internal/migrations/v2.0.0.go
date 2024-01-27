package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_0_0 performs the DB migrations for v.1.0.0.
func V2_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'bounce_type') THEN
				CREATE TYPE bounce_type AS ENUM ('soft', 'hard', 'complaint');
			END IF;
		END$$;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS bounces (
		    id               SERIAL PRIMARY KEY,
		    subscriber_id    INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE ON UPDATE CASCADE,
		    campaign_id      INTEGER NULL REFERENCES campaigns(id) ON DELETE SET NULL ON UPDATE CASCADE,
		    type             bounce_type NOT NULL DEFAULT 'hard',
		    source           TEXT NOT NULL DEFAULT '',
		    meta             JSONB NOT NULL DEFAULT '{}',
		    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_bounces_sub_id ON bounces(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_bounces_camp_id ON bounces(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_bounces_source ON bounces(source);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
	    ('app.send_optin_confirmation', 'true'),
	    ('privacy.domain_blocklist', '[]'),
	    ('bounce.enabled', 'false'),
	    ('bounce.webhooks_enabled', 'false'),
	    ('bounce.count', '2'),
	    ('bounce.action', '"blocklist"'),
	    ('bounce.ses_enabled', 'false'),
	    ('bounce.sendgrid_enabled', 'false'),
	    ('bounce.sendgrid_key', '""'),
	    ('bounce.mailboxes', '[{"enabled":false, "type": "pop", "host":"pop.yoursite.com","port":995,"auth_protocol":"userpass","username":"username","password":"password","return_path": "bounce@listmonk.yoursite.com","scan_interval":"15m","tls_enabled":true,"tls_skip_verify":false}]')
	    ON CONFLICT DO NOTHING;`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE subscribers DROP COLUMN IF EXISTS campaigns`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'campaign_views_pkey') THEN
				ALTER TABLE campaign_views ADD COLUMN IF NOT EXISTS id BIGSERIAL PRIMARY KEY;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'link_clicks_pkey') THEN
				ALTER TABLE link_clicks ADD COLUMN IF NOT EXISTS id BIGSERIAL PRIMARY KEY;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'campaign_lists_pkey') THEN
				ALTER TABLE campaign_lists ADD COLUMN IF NOT EXISTS id BIGSERIAL PRIMARY KEY;
			END IF;
		END$$;

		CREATE INDEX IF NOT EXISTS idx_views_date ON campaign_views((TIMEZONE('UTC', created_at)::DATE));
		CREATE INDEX IF NOT EXISTS idx_clicks_date ON link_clicks((TIMEZONE('UTC', created_at)::DATE));
	`); err != nil {
		return err
	}

	// S3 URL i snow a settings field. Prepare S3 URL based on region and bucket.
	if _, err := db.Exec(`
		WITH region AS (
			SELECT value#>>'{}' AS value FROM settings WHERE key='upload.s3.aws_default_region'
		), s3url AS (
			SELECT FORMAT('https://s3.%s.amazonaws.com', (SELECT value FROM region)) AS value
		)

		INSERT INTO settings (key, value) VALUES ('upload.s3.url', TO_JSON((SELECT * FROM s3url))) ON CONFLICT DO NOTHING;`); err != nil {
		return err
	}

	return nil
}
