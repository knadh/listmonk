package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V3_0_0 performs the DB migrations.
func V3_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("IMPORTANT: this upgrade might take a while if you have a large database. Please be patient ...")

	// Insert new preference settings.
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
		('bounce.postmark', '{"enabled": false, "username": "", "password": ""}'),
		('app.cache_slow_queries', 'false'),
		('app.cache_slow_queries_interval', '"0 3 * * *"')
		ON CONFLICT DO NOTHING;
	`); err != nil {
		return err
	}

	// Fix incorrect "d" (day) time prefix in S3 expiry settings.
	if _, err := db.Exec(`UPDATE settings SET value = '"167h"'  WHERE key = 'upload.s3.expiry' AND value = '"14d"'`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS archive_slug TEXT NULL UNIQUE`); err != nil {
		return err
	}

	// Add indexes that make sorting faster on large tables.
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_subs_created_at ON subscribers(created_at);
		CREATE INDEX IF NOT EXISTS idx_subs_updated_at ON subscribers(updated_at);

		CREATE INDEX IF NOT EXISTS idx_camps_status ON campaigns(status);
		CREATE INDEX IF NOT EXISTS idx_camps_name ON campaigns(name);
		CREATE INDEX IF NOT EXISTS idx_camps_created_at ON campaigns(created_at);
		CREATE INDEX IF NOT EXISTS idx_camps_updated_at ON campaigns(updated_at);

		CREATE INDEX IF NOT EXISTS idx_lists_type ON lists(type);
		CREATE INDEX IF NOT EXISTS idx_lists_optin ON lists(optin);
		CREATE INDEX IF NOT EXISTS idx_lists_name ON lists(name);
		CREATE INDEX IF NOT EXISTS idx_lists_created_at ON lists(created_at);
		CREATE INDEX IF NOT EXISTS idx_lists_updated_at ON lists(updated_at);
	`); err != nil {
		return err
	}

	// Create materialized views for slow aggregate queries.
	if _, err := db.Exec(`
		-- dashboard stats
		CREATE MATERIALIZED VIEW IF NOT EXISTS mat_dashboard_counts AS
		    WITH subs AS (
		        SELECT COUNT(*) AS num, status FROM subscribers GROUP BY status
		    )
		    SELECT NOW() AS updated_at,
		        JSON_BUILD_OBJECT(
		            'subscribers', JSON_BUILD_OBJECT(
		                'total', (SELECT SUM(num) FROM subs),
		                'blocklisted', (SELECT num FROM subs WHERE status='blocklisted'),
		                'orphans', (
		                    SELECT COUNT(id) FROM subscribers
		                    LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id)
		                    WHERE subscriber_lists.subscriber_id IS NULL
		                )
		            ),
		            'lists', JSON_BUILD_OBJECT(
		                'total', (SELECT COUNT(*) FROM lists),
		                'private', (SELECT COUNT(*) FROM lists WHERE type='private'),
		                'public', (SELECT COUNT(*) FROM lists WHERE type='public'),
		                'optin_single', (SELECT COUNT(*) FROM lists WHERE optin='single'),
		                'optin_double', (SELECT COUNT(*) FROM lists WHERE optin='double')
		            ),
		            'campaigns', JSON_BUILD_OBJECT(
		                'total', (SELECT COUNT(*) FROM campaigns),
		                'by_status', (
		                    SELECT JSON_OBJECT_AGG (status, num) FROM
		                    (SELECT status, COUNT(*) AS num FROM campaigns GROUP BY status) r
		                )
		            ),
		            'messages', (SELECT SUM(sent) AS messages FROM campaigns)
		        ) AS data;
		CREATE UNIQUE INDEX IF NOT EXISTS mat_dashboard_stats_idx ON mat_dashboard_counts (updated_at);

		CREATE MATERIALIZED VIEW IF NOT EXISTS mat_dashboard_charts AS
		    WITH clicks AS (
		        SELECT JSON_AGG(ROW_TO_JSON(row))
		        FROM (
		            WITH viewDates AS (
		              SELECT TIMEZONE('UTC', created_at)::DATE AS to_date,
		                     TIMEZONE('UTC', created_at)::DATE - INTERVAL '30 DAY' AS from_date
		                     FROM link_clicks ORDER BY id DESC LIMIT 1
		            )
		            SELECT COUNT(*) AS count, created_at::DATE as date FROM link_clicks
		              -- use > between < to force the use of the date index.
		              WHERE TIMEZONE('UTC', created_at)::DATE BETWEEN (SELECT from_date FROM viewDates) AND (SELECT to_date FROM viewDates)
		              GROUP by date ORDER BY date
		        ) row
		    ),
		    views AS (
		        SELECT JSON_AGG(ROW_TO_JSON(row))
		        FROM (
		            WITH viewDates AS (
		              SELECT TIMEZONE('UTC', created_at)::DATE AS to_date,
		                     TIMEZONE('UTC', created_at)::DATE - INTERVAL '30 DAY' AS from_date
		                     FROM campaign_views ORDER BY id DESC LIMIT 1
		            )
		            SELECT COUNT(*) AS count, created_at::DATE as date FROM campaign_views
		              -- use > between < to force the use of the date index.
		              WHERE TIMEZONE('UTC', created_at)::DATE BETWEEN (SELECT from_date FROM viewDates) AND (SELECT to_date FROM viewDates)
		              GROUP by date ORDER BY date
		        ) row
		    )
		    SELECT NOW() AS updated_at, JSON_BUILD_OBJECT('link_clicks', COALESCE((SELECT * FROM clicks), '[]'),
		                                  'campaign_views', COALESCE((SELECT * FROM views), '[]')
		                                ) AS data;
		CREATE UNIQUE INDEX IF NOT EXISTS mat_dashboard_charts_idx ON mat_dashboard_charts (updated_at);

		-- subscriber counts stats for lists
		CREATE MATERIALIZED VIEW IF NOT EXISTS mat_list_subscriber_stats AS
		    SELECT NOW() AS updated_at, lists.id AS list_id, subscriber_lists.status, COUNT(*) AS subscriber_count FROM lists
		    LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = lists.id)
		    GROUP BY lists.id, subscriber_lists.status
		    UNION ALL
		    SELECT NOW() AS updated_at, 0 AS list_id, NULL AS status, COUNT(*) AS subscriber_count FROM subscribers;
		CREATE UNIQUE INDEX IF NOT EXISTS mat_list_subscriber_stats_idx ON mat_list_subscriber_stats (list_id, status);
	`); err != nil {
		return err
	}

	return nil
}
