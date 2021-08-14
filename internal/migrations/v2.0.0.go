package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/stuffbin"
)

// V2_0_0 performs the DB migrations for v.1.0.0.
func V2_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
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

	if _, err := db.Exec(`ALTER TABLE subscribers DROP COLUMN IF EXISTS campaigns; `); err != nil {
		return err
	}

	return nil
}
