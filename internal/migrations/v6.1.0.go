package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES ('privacy.disable_tracking', 'false', NOW()) ON CONFLICT (key) DO NOTHING
	`); err != nil {
		return err
	}

	// Drop the old UTC-based date indexes and simply use local time with zone consistent
	// with the rest of the schema.
	if _, err := db.Exec(`
		DROP INDEX IF EXISTS idx_views_date; CREATE INDEX IF NOT EXISTS idx_views_date ON campaign_views(created_at);
		DROP INDEX IF EXISTS idx_clicks_date; CREATE INDEX IF NOT EXISTS idx_clicks_date ON link_clicks(created_at);
		DROP INDEX IF EXISTS idx_bounces_date; CREATE INDEX IF NOT EXISTS idx_bounces_date ON bounces(created_at);
	`); err != nil {
		return err
	}

	// Recreate the materialized views to use server local time instead of UTC.
	// Create new views first, let them populate, then drop the old ones and rename the new ones.
	lo.Println("IMPORTANT: recreating analytics materialized views. This might take a while if you have a large database. Please be patient ...")
	if _, err := db.Exec(`
		CREATE MATERIALIZED VIEW IF NOT EXISTS mat_dashboard_charts_v6_1_0 AS
		WITH clicks AS (
			SELECT JSON_AGG(ROW_TO_JSON(row))
			FROM (
				WITH viewDates AS (
					SELECT created_at::DATE AS to_date,
						   created_at::DATE - INTERVAL '30 DAY' AS from_date
						   FROM link_clicks ORDER BY id DESC LIMIT 1
				)
				SELECT COUNT(*) AS count, created_at::DATE as date FROM link_clicks
					WHERE created_at >= (SELECT from_date FROM viewDates)
					AND created_at < (SELECT to_date FROM viewDates) + INTERVAL '1 day'
					GROUP by date ORDER BY date
			) row
		),
		views AS (
			SELECT JSON_AGG(ROW_TO_JSON(row))
			FROM (
				WITH viewDates AS (
					SELECT created_at::DATE AS to_date,
						   created_at::DATE - INTERVAL '30 DAY' AS from_date
						   FROM campaign_views ORDER BY id DESC LIMIT 1
				)
				SELECT COUNT(*) AS count, created_at::DATE as date FROM campaign_views
					WHERE created_at >= (SELECT from_date FROM viewDates)
					AND created_at < (SELECT to_date FROM viewDates) + INTERVAL '1 day'
					GROUP by date ORDER BY date
			) row
		)
		SELECT NOW() AS updated_at, JSON_BUILD_OBJECT('link_clicks', COALESCE((SELECT * FROM clicks), '[]'),
								  'campaign_views', COALESCE((SELECT * FROM views), '[]')
								) AS data;

		DROP INDEX IF EXISTS mat_dashboard_charts_idx;
		DROP MATERIALIZED VIEW IF EXISTS mat_dashboard_charts;

		ALTER MATERIALIZED VIEW mat_dashboard_charts_v6_1_0 RENAME TO mat_dashboard_charts;
		CREATE UNIQUE INDEX IF NOT EXISTS mat_dashboard_charts_idx ON mat_dashboard_charts (updated_at);
	`); err != nil {
		return err
	}

	return nil
}
