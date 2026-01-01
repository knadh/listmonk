package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V5_3_0 adds webhook_logs table for persistent webhook delivery with background workers.
func V5_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Create webhook_log_status enum type.
	_, err := db.Exec(`
		DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'webhook_log_status') THEN
				CREATE TYPE webhook_log_status AS ENUM ('triggered', 'processing', 'completed', 'failed');
			END IF;
		END $$;
	`)
	if err != nil {
		return err
	}

	// Create webhook_logs table.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS webhook_logs (
			id              SERIAL PRIMARY KEY,
			webhook_id      TEXT NOT NULL,
			event           TEXT NOT NULL,
			payload         JSONB NOT NULL DEFAULT '{}',
			status          webhook_log_status NOT NULL DEFAULT 'triggered',
			retries         INT NOT NULL DEFAULT 0,
			last_retried_at TIMESTAMP WITH TIME ZONE,
			response        JSONB NOT NULL DEFAULT '{}',
			note            TEXT,
			created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_webhook_logs_webhook_id ON webhook_logs(webhook_id);
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_status ON webhook_logs(status);
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_created_at ON webhook_logs(created_at);
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_status_created ON webhook_logs(status, created_at);
	`)
	if err != nil {
		return err
	}

	// Add webhook workers setting.
	_, err = db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES ('app.webhook_workers', '2', NOW()) ON CONFLICT (key) DO NOTHING;
		INSERT INTO settings (key, value, updated_at) VALUES ('app.webhook_batch_size', '50', NOW()) ON CONFLICT (key) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	return nil
}
