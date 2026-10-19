-- lists
-- name: get-lists
SELECT * FROM lists WHERE (CASE WHEN $1 = '' THEN 1=1 ELSE type=$1::list_type END)
    AND (CASE WHEN $2 = '' THEN 1=1 ELSE status=$2::list_status END)
    AND CASE
        -- Optional list IDs based on user permission.
        WHEN $4 = TRUE THEN TRUE ELSE id = ANY($5::INT[])
    END
    ORDER BY CASE WHEN $3 = 'id' THEN id END, CASE WHEN $3 = 'name' THEN name END;

-- name: query-lists
WITH statuses AS (
    SELECT
        list_id,
        COALESCE(JSONB_OBJECT_AGG(status, subscriber_count) FILTER (WHERE status IS NOT NULL), '{}') AS subscriber_statuses,
        SUM(subscriber_count) AS subscriber_count
    FROM mat_list_subscriber_stats
    GROUP BY list_id
),
ls AS (
    SELECT
        COUNT(*) OVER () AS total,
        lists.*,
        COALESCE(statuses.subscriber_statuses, '{}') AS subscriber_statuses,
        COALESCE(statuses.subscriber_count, 0) AS subscriber_count
    FROM lists
    LEFT JOIN statuses ON (lists.id = statuses.list_id)
    WHERE
    CASE
        WHEN $1 > 0 THEN lists.id = $1
        WHEN $2 != '' THEN uuid = $2::UUID
        WHEN $3 != '' THEN (TO_TSVECTOR(name) @@ PLAINTO_TSQUERY($3) OR name ILIKE ('%' || $3 || '%'))
        ELSE TRUE
    END
    AND ($4 = '' OR type = $4::list_type)
    AND ($5 = '' OR optin = $5::list_optin)
    AND ($6 = '' OR status = $6::list_status)
    AND (CARDINALITY($7::VARCHAR(100)[]) = 0 OR $7 <@ tags)
    AND CASE
        -- Optional list IDs based on user permission.
        WHEN $8 = TRUE THEN TRUE ELSE lists.id = ANY($9::INT[])
    END
    ORDER BY %order% OFFSET $10 LIMIT (CASE WHEN $11 < 1 THEN NULL ELSE $11 END)
)
SELECT * FROM ls ORDER BY %order%;

-- name: get-lists-by-optin
-- Can have a list of IDs or a list of UUIDs.
SELECT * FROM lists WHERE (CASE WHEN $1 != '' THEN optin=$1::list_optin ELSE TRUE END) AND
    (CASE WHEN $2::INT[] IS NOT NULL THEN id = ANY($2::INT[])
          WHEN $3::UUID[] IS NOT NULL THEN uuid = ANY($3::UUID[])
    END) ORDER BY name;

-- name: get-list-types
-- Retrieves the private|public type of lists by ID or uuid. Used for filtering.
SELECT id, uuid, type FROM lists WHERE
    (CASE WHEN $1::INT[] IS NOT NULL THEN id = ANY($1::INT[])
          WHEN $2::UUID[] IS NOT NULL THEN uuid = ANY($2::UUID[])
    END);

-- name: create-list
INSERT INTO lists (uuid, name, type, optin, status, tags, description) VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id;

-- name: update-list
WITH l AS (
    UPDATE lists SET
        name=(CASE WHEN $2 != '' THEN $2 ELSE name END),
        type=(CASE WHEN $3 != '' THEN $3::list_type ELSE type END),
        optin=(CASE WHEN $4 != '' THEN $4::list_optin ELSE optin END),
        status=(CASE WHEN $5 != '' THEN $5::list_status ELSE status END),
        tags=$6::VARCHAR(100)[],
        description=(CASE WHEN $7 != '' THEN $7 ELSE description END),
        updated_at=NOW()
    WHERE id = $1
    RETURNING id, name
),
c AS (
    UPDATE campaign_lists SET list_name = l.name FROM l WHERE campaign_lists.list_id = l.id RETURNING 1
)
SELECT COUNT(*) FROM l, c;

-- name: update-lists-date
UPDATE lists SET updated_at=NOW() WHERE id = ANY($1);

-- name: delete-lists
DELETE FROM lists
WHERE CASE
    WHEN CARDINALITY($1::INT[]) > 0 THEN id = ANY($1)
    ELSE ($2 = '' OR to_tsvector(name) @@ plainto_tsquery($2) OR name ILIKE ('%' || $2 || '%'))
END
AND (
    CARDINALITY($1::INT[]) > 0 OR (
        ($3 = '' OR type = $3::list_type)
        AND ($4 = '' OR optin = $4::list_optin)
        AND ($5 = '' OR status = $5::list_status)
        AND (CARDINALITY($6::VARCHAR(100)[]) = 0 OR $6 <@ tags)
    )
)
AND CASE
    -- Optional list IDs based on user permission.
    WHEN $7 = TRUE THEN TRUE ELSE id = ANY($8::INT[])
END;

-- name: get-list-charts
-- Fill missing weeks, including empty/new lists absent from the cached view.
WITH weeks AS (
    SELECT GENERATE_SERIES(DATE_TRUNC('week', CURRENT_DATE - INTERVAL '3 months'),
                           DATE_TRUNC('week', CURRENT_DATE), INTERVAL '1 week')::DATE AS date
)
SELECT JSON_BUILD_OBJECT(
    'unsubscriptions', JSON_AGG(JSON_BUILD_OBJECT('date', w.date, 'count', COALESCE(c.unsubscriptions, 0)) ORDER BY w.date),
    'subscriptions', JSON_AGG(JSON_BUILD_OBJECT('date', w.date, 'count', COALESCE(c.subscriptions, 0)) ORDER BY w.date),
    'campaign_views', CASE WHEN $2::BOOLEAN THEN
        JSON_AGG(JSON_BUILD_OBJECT('date', w.date, 'count', COALESCE(c.campaign_views, 0)) ORDER BY w.date)
        ELSE '[]'::JSON END
)
FROM weeks w LEFT JOIN mat_list_charts c ON c.list_id = $1 AND c.date = w.date;

-- name: get-list-charts-live
-- raw: true
-- Compute the requested list live when slow-query caching is disabled.
-- Executed as a raw unprepared query so that different intervals don't end up using
-- the same cached query plan which may be suboptimal. This was discovered through
-- a bunch of profile exercises.
WITH subscriptions AS (
    SELECT DATE_TRUNC('week', created_at)::DATE AS date, COUNT(*) AS count
    FROM subscriber_lists
    WHERE
        list_id = $1
        AND created_at >= CURRENT_DATE - INTERVAL '3 months'
        AND created_at <= NOW()
    GROUP BY date
), unsubscriptions AS (
    SELECT DATE_TRUNC('week', updated_at)::DATE AS date, COUNT(*) AS count
    FROM subscriber_lists
    WHERE
        list_id = $1
        AND status = 'unsubscribed'
        AND updated_at >= CURRENT_DATE - INTERVAL '3 months'
        AND updated_at <= NOW()
    GROUP BY date
), views AS (
    SELECT DATE_TRUNC('week', v.created_at)::DATE AS date, COUNT(*) AS count
    FROM campaign_views v JOIN campaign_lists cl ON cl.campaign_id = v.campaign_id
    WHERE
        $2::BOOLEAN AND cl.list_id = $1
        AND v.created_at >= CURRENT_DATE - INTERVAL '3 months'
        AND v.created_at <= NOW()
    GROUP BY date
), weeks AS (
    SELECT GENERATE_SERIES(DATE_TRUNC('week', CURRENT_DATE - INTERVAL '3 months'),
                           DATE_TRUNC('week', CURRENT_DATE), INTERVAL '1 week')::DATE AS date
)
SELECT JSON_BUILD_OBJECT(
    'unsubscriptions', JSON_AGG(JSON_BUILD_OBJECT('date', w.date, 'count', COALESCE(u.count, 0)) ORDER BY w.date),
    'subscriptions', JSON_AGG(JSON_BUILD_OBJECT('date', w.date, 'count', COALESCE(s.count, 0)) ORDER BY w.date),
    'campaign_views',
        CASE WHEN $2::BOOLEAN
            THEN JSON_AGG(JSON_BUILD_OBJECT('date', w.date, 'count', COALESCE(v.count, 0)) ORDER BY w.date)
            ELSE '[]'::JSON
        END
)
FROM weeks w
LEFT JOIN subscriptions s ON s.date = w.date
LEFT JOIN unsubscriptions u ON u.date = w.date
LEFT JOIN views v ON v.date = w.date;

-- name: get-list-analytics-counts
-- raw: true
WITH counts AS (
    SELECT campaign_id, COUNT(*) AS "count", DATE_TRUNC($4::TEXT, created_at) AS "timestamp"
    FROM %s
    WHERE campaign_id IN (SELECT campaign_id FROM campaign_lists WHERE list_id = ANY($1))
        AND created_at >= $2 AND created_at <= $3
    GROUP BY campaign_id, "timestamp"
)
SELECT cl.list_id, SUM(c.count)::BIGINT AS "count", c.timestamp
FROM counts c JOIN campaign_lists cl ON cl.campaign_id = c.campaign_id
WHERE cl.list_id = ANY($1)
GROUP BY cl.list_id, c.timestamp ORDER BY c.timestamp;

-- name: get-list-subscription-counts
SELECT list_id, COUNT(*) AS "count", DATE_TRUNC($4::TEXT, created_at) AS "timestamp"
FROM subscriber_lists
WHERE list_id = ANY($1) AND created_at >= $2 AND created_at <= $3
GROUP BY list_id, "timestamp" ORDER BY "timestamp" ASC;

-- name: get-list-unsubscription-counts
-- Memberships retain only their current status and last update, not event history.
SELECT list_id, COUNT(*) AS "count", DATE_TRUNC($4::TEXT, updated_at) AS "timestamp"
FROM subscriber_lists
WHERE list_id = ANY($1) AND status = 'unsubscribed' AND updated_at >= $2 AND updated_at <= $3
GROUP BY list_id, "timestamp" ORDER BY "timestamp" ASC;

-- name: get-list-link-counts
-- raw: true
SELECT COUNT(%s) AS "count", url
FROM link_clicks LEFT JOIN links ON link_clicks.link_id = links.id
WHERE campaign_id IN (SELECT campaign_id FROM campaign_lists WHERE list_id = ANY($1))
    AND link_clicks.created_at >= $2 AND link_clicks.created_at <= $3
GROUP BY links.url ORDER BY "count" DESC LIMIT 50;

-- name: get-list-analytics-unique-counts
-- raw: true
WITH uniq AS (
    SELECT DISTINCT ON (subscriber_id, campaign_id)
        campaign_id, DATE_TRUNC($4::TEXT, created_at) AS "timestamp"
    FROM %s
    WHERE campaign_id IN (SELECT campaign_id FROM campaign_lists WHERE list_id = ANY($1))
        AND created_at >= $2 AND created_at <= $3
    ORDER BY subscriber_id, campaign_id, "timestamp"
), counts AS (
    SELECT campaign_id, COUNT(*) AS "count", "timestamp"
    FROM uniq GROUP BY campaign_id, "timestamp"
)
SELECT cl.list_id, SUM(c.count)::BIGINT AS "count", c.timestamp
FROM counts c JOIN campaign_lists cl ON cl.campaign_id = c.campaign_id
WHERE cl.list_id = ANY($1)
GROUP BY cl.list_id, c.timestamp ORDER BY c.timestamp;

-- name: get-list-analytics-items
SELECT id, name FROM lists
WHERE id = ANY($1::INT[]) AND ($2 OR id = ANY($3::INT[]))
ORDER BY ARRAY_POSITION($1::INT[], id);
