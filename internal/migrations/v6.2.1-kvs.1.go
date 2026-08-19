package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_2_1_KVS_1 adds provider delivery lifecycle tracking for campaigns.
func V6_2_1_KVS_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS campaign_delivery_events (
			id                    BIGSERIAL PRIMARY KEY,
			campaign_id           INTEGER NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,
			subscriber_id         INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL ON UPDATE CASCADE,
			provider              TEXT NOT NULL,
			provider_event_id     TEXT NOT NULL,
			provider_message_id   TEXT NOT NULL DEFAULT '',
			event_type            TEXT NOT NULL,
			meta                  JSONB NOT NULL DEFAULT '{}',
			occurred_at           TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			UNIQUE(provider, provider_event_id)
		);

		ALTER TABLE campaign_delivery_events
			ALTER COLUMN campaign_id DROP NOT NULL;

		CREATE INDEX IF NOT EXISTS idx_delivery_events_camp_type
			ON campaign_delivery_events(campaign_id, event_type);
		CREATE INDEX IF NOT EXISTS idx_delivery_events_sub_id
			ON campaign_delivery_events(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_delivery_events_date
			ON campaign_delivery_events(occurred_at);

		INSERT INTO settings (key, value)
		VALUES ('delivery.sendgrid_tracking_started_at', TO_JSONB(NOW()))
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
