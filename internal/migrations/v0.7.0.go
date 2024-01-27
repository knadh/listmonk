package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_7_0 performs the DB migrations for v.0.7.0.
func V0_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Check if the subscriber_status.blocklisted enum value exists. If not,
	// it has to be created (for the change from blacklisted -> blocklisted).
	var bl bool
	if err := db.Get(&bl, `SELECT 'blocklisted' = ANY(ENUM_RANGE(NULL::subscriber_status)::TEXT[])`); err != nil {
		return err
	}

	// If `blocklist` doesn't exist, add it to the subscriber_status enum,
	// and update existing statuses to this value. Unfortunately, it's not possible
	// to remove the enum value `blacklisted` (until PG10).
	if !bl {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err := tx.Exec(`
			-- Change the status column to text.
			ALTER TABLE subscribers ALTER COLUMN status TYPE TEXT;

			-- Change all statuses from 'blacklisted' to 'blocklisted'.
			UPDATE subscribers SET status='blocklisted' WHERE status='blacklisted';
	
			-- Remove the old enum.
			DROP TYPE subscriber_status CASCADE;

			-- Create new enum with the new values.
			CREATE TYPE subscriber_status AS ENUM ('enabled', 'disabled', 'blocklisted');

			-- Change the text status column to the new enum.
			ALTER TABLE subscribers ALTER COLUMN status TYPE subscriber_status
				USING (status::subscriber_status);
		`); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	_, err := db.Exec(`
	ALTER TABLE media DROP COLUMN IF EXISTS width,
					  DROP COLUMN IF EXISTS height,
					  ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT '';

	-- 'blacklisted' to 'blocklisted' ENUM rename is not possible (until pg10),
	-- so just add the new value and ignore the old one.
	
	
	CREATE TABLE IF NOT EXISTS settings (
		key             TEXT NOT NULL UNIQUE,
		value           JSONB NOT NULL DEFAULT '{}',
		updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_settings_key ON settings(key);
	
	-- Insert default settings if the table is empty.
	INSERT INTO settings (key, value) SELECT k, v::JSONB FROM (VALUES
		('app.root_url', '"http://localhost:9000"'),
		('app.favicon_url', '""'),
		('app.from_email', '"listmonk <noreply@listmonk.yoursite.com>"'),
		('app.logo_url', '"http://localhost:9000/public/static/logo.png"'),
		('app.concurrency', '10'),
		('app.message_rate', '10'),
		('app.batch_size', '1000'),
		('app.max_send_errors', '1000'),
		('app.notify_emails', '["admin1@mysite.com", "admin2@mysite.com"]'),
		('privacy.unsubscribe_header', 'true'),
		('privacy.allow_blocklist', 'true'),
		('privacy.allow_export', 'true'),
		('privacy.allow_wipe', 'true'),
		('privacy.exportable', '["profile", "subscriptions", "campaign_views", "link_clicks"]'),
		('upload.provider', '"filesystem"'),
		('upload.filesystem.upload_path', '"uploads"'),
		('upload.filesystem.upload_uri', '"/uploads"'),
		('upload.s3.aws_access_key_id', '""'),
		('upload.s3.aws_secret_access_key', '""'),
		('upload.s3.aws_default_region', '"ap-south-1"'),
		('upload.s3.bucket', '""'),
		('upload.s3.bucket_domain', '""'),
		('upload.s3.bucket_path', '"/"'),
		('upload.s3.bucket_type', '"public"'),
		('upload.s3.expiry', '"14d"'),
		('smtp',
			'[{"enabled":true, "host":"smtp.yoursite.com","port":25,"auth_protocol":"cram","username":"username","password":"password","hello_hostname":"","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","max_msg_retries":2,"tls_enabled":true,"tls_skip_verify":false,"email_headers":[]},
			  {"enabled":false, "host":"smtp2.yoursite.com","port":587,"auth_protocol":"plain","username":"username","password":"password","hello_hostname":"","max_conns":10,"idle_timeout":"15s","wait_timeout":"5s","max_msg_retries":2,"tls_enabled":false,"tls_skip_verify":false,"email_headers":[]}]'),
		('messengers', '[]')) vals(k, v) WHERE NOT EXISTS(SELECT * FROM settings LIMIT 1);
	
	`)
	if err != nil {
		return err
	}

	// `provider` in the media table is a new field. If there's provider config available
	// and no provider value exists in the media table, set it.
	prov := ko.String("upload.provider")
	if prov != "" {
		if _, err := db.Exec(`UPDATE media SET provider=$1 WHERE provider=''`, prov); err != nil {
			return err
		}
	}

	return nil
}
