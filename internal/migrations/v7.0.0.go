package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V7_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// OIDC claim mapping.
	if _, err := db.Exec(`UPDATE settings SET value = JSONB_SET(value, '{roles}', '[]'::JSONB) WHERE key = 'security.oidc' AND NOT (value ? 'roles');`); err != nil {
		return err
	}

	// List analytics.
	if _, err := db.Exec(`
CREATE MATERIALIZED VIEW IF NOT EXISTS mat_list_charts AS
    -- Aggregate campaign events before joining lists so multi-list campaigns do not
    -- multiply the work over the raw events. Views belong to campaigns, not lists.
    WITH views AS (
        SELECT campaign_id, DATE_TRUNC('week', created_at)::DATE AS date, COUNT(*) AS count
        FROM campaign_views
        WHERE created_at >= CURRENT_DATE - INTERVAL '12 months' AND created_at <= NOW()
        GROUP BY campaign_id, date
    ), counts AS (
        SELECT list_id, DATE_TRUNC('week', created_at)::DATE AS date,
               COUNT(*) AS subscriptions, 0::BIGINT AS campaign_views
        FROM subscriber_lists
        WHERE created_at >= CURRENT_DATE - INTERVAL '12 months' AND created_at <= NOW()
        GROUP BY list_id, date
        UNION ALL
        SELECT cl.list_id, v.date, 0::BIGINT, SUM(v.count)::BIGINT
        FROM views v JOIN campaign_lists cl ON cl.campaign_id = v.campaign_id
        WHERE cl.list_id IS NOT NULL
        GROUP BY cl.list_id, v.date
    )
    SELECT list_id, date, SUM(subscriptions)::BIGINT AS subscriptions,
           SUM(campaign_views)::BIGINT AS campaign_views
    FROM counts
    GROUP BY list_id, date;
CREATE UNIQUE INDEX IF NOT EXISTS mat_list_charts_idx ON mat_list_charts (list_id, date);
	`); err != nil {
		return err
	}

	return nil
}
