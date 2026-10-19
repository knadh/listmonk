package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V7_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("IMPORTANT: this upgrade might take a while if you have a large database. Please be patient ...")

	// Settings.
	if _, err := db.Exec(`
        -- Migrate existing from_email to from_email[].
        UPDATE settings SET value = JSONB_BUILD_ARRAY(value)
        WHERE key = 'app.from_email' AND JSONB_TYPEOF(value) = 'string';

        -- Login session duration.
        INSERT INTO settings (key, value) VALUES ('security.session_duration', '"7d"')
        ON CONFLICT (key) DO NOTHING;

        -- Add exclude_from_default field to all SMTP servers.
        UPDATE settings
        SET value = (
            SELECT JSONB_AGG(
                JSONB_SET(v, '{exclude_from_default}', COALESCE(v->'exclude_from_default', 'false'::JSONB))
                ORDER BY ord
            )
            FROM JSONB_ARRAY_ELEMENTS(value) WITH ORDINALITY AS t(v, ord)
        )
        WHERE key = 'smtp' AND EXISTS (
            SELECT 1 FROM JSONB_ARRAY_ELEMENTS(value) AS v
            WHERE NOT (v ? 'exclude_from_default')
        );

        -- OIDC RP-initiated logout option.
        UPDATE settings SET value = JSONB_SET(value, '{logout}', 'false'::JSONB)
        WHERE key = 'security.oidc' AND NOT (value ? 'logout');

        -- OIDC claim mapping.
        UPDATE settings SET value = JSONB_SET(value, '{roles}', '[]'::JSONB)
        WHERE key = 'security.oidc' AND NOT (value ? 'roles');
	`); err != nil {
		return err
	}

	// List analytics.
	if _, err := db.Exec(`
CREATE INDEX IF NOT EXISTS idx_sub_lists_created_at ON subscriber_lists(created_at);
CREATE INDEX IF NOT EXISTS idx_sub_lists_unsubscribed_updated_at ON subscriber_lists(updated_at) WHERE status = 'unsubscribed';

DROP MATERIALIZED VIEW IF EXISTS mat_list_charts;
CREATE MATERIALIZED VIEW mat_list_charts AS
    WITH views AS (
        SELECT campaign_id, DATE_TRUNC('week', created_at)::DATE AS date, COUNT(*) AS count
        FROM campaign_views
        WHERE created_at >= CURRENT_DATE - INTERVAL '3 months' AND created_at <= NOW()
        GROUP BY campaign_id, date
    ), counts AS (
        SELECT list_id, DATE_TRUNC('week', created_at)::DATE AS date,
               COUNT(*) AS subscriptions, 0::BIGINT AS unsubscriptions, 0::BIGINT AS campaign_views
        FROM subscriber_lists
        WHERE created_at >= CURRENT_DATE - INTERVAL '3 months' AND created_at <= NOW()
        GROUP BY list_id, date
        UNION ALL
        SELECT list_id, DATE_TRUNC('week', updated_at)::DATE AS date,
               0::BIGINT, COUNT(*), 0::BIGINT
        FROM subscriber_lists
        WHERE status = 'unsubscribed'
            AND updated_at >= CURRENT_DATE - INTERVAL '3 months' AND updated_at <= NOW()
        GROUP BY list_id, date
        UNION ALL
        SELECT cl.list_id, v.date, 0::BIGINT, 0::BIGINT, SUM(v.count)::BIGINT
        FROM views v JOIN campaign_lists cl ON cl.campaign_id = v.campaign_id
        WHERE cl.list_id IS NOT NULL
        GROUP BY cl.list_id, v.date
    )
    SELECT list_id, date, SUM(subscriptions)::BIGINT AS subscriptions,
           SUM(unsubscriptions)::BIGINT AS unsubscriptions,
           SUM(campaign_views)::BIGINT AS campaign_views
    FROM counts
    GROUP BY list_id, date;
CREATE UNIQUE INDEX IF NOT EXISTS mat_list_charts_idx ON mat_list_charts (list_id, date);
	`); err != nil {
		return err
	}

	return nil
}
