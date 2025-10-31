-- privacy
-- name: export-subscriber-data
WITH prof AS (
    SELECT id, uuid, email, name, attribs, status, created_at, updated_at FROM subscribers WHERE
    CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END
),
subs AS (
    SELECT subscriber_lists.status AS subscription_status,
            (CASE WHEN lists.type = 'private' THEN 'Private list' ELSE lists.name END) as name,
            lists.type, subscriber_lists.created_at
    FROM lists
    LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = lists.id)
    WHERE subscriber_lists.subscriber_id = (SELECT id FROM prof)
),
views AS (
    SELECT subject as campaign, COUNT(subscriber_id) as views FROM campaign_views
        LEFT JOIN campaigns ON (campaigns.id = campaign_views.campaign_id)
        WHERE subscriber_id = (SELECT id FROM prof)
        GROUP BY campaigns.id ORDER BY campaigns.id
),
clicks AS (
    SELECT url, COUNT(subscriber_id) as clicks FROM link_clicks
        LEFT JOIN links ON (links.id = link_clicks.link_id)
        WHERE subscriber_id = (SELECT id FROM prof)
        GROUP BY links.id ORDER BY links.id
)
SELECT (SELECT email FROM prof) as email,
        COALESCE((SELECT JSON_AGG(t) FROM prof t), '{}') AS profile,
        COALESCE((SELECT JSON_AGG(t) FROM subs t), '[]') AS subscriptions,
        COALESCE((SELECT JSON_AGG(t) FROM views t), '[]') AS campaign_views,
        COALESCE((SELECT JSON_AGG(t) FROM clicks t), '[]') AS link_clicks;
