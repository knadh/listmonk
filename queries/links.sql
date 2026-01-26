-- links
-- name: create-link
INSERT INTO links (uuid, url) VALUES($1, $2) ON CONFLICT (url) DO UPDATE SET url=EXCLUDED.url RETURNING uuid;

-- name: register-link-click
-- Returns (url, campaign_name) in one round-trip for link redirect (optionally with UTM).
WITH link AS (
    SELECT id, url FROM links WHERE uuid = $1
),
camp AS (
    SELECT id, name FROM campaigns WHERE uuid = $2
),
inserted AS (
    INSERT INTO link_clicks (campaign_id, subscriber_id, link_id)
    VALUES (
        (SELECT id FROM camp),
        (SELECT id FROM subscribers WHERE
            (CASE WHEN $3::TEXT != '' THEN subscribers.uuid = $3::UUID ELSE FALSE END)
        ),
        (SELECT id FROM link)
    )
    RETURNING campaign_id
)
SELECT link.url, COALESCE(camp.name, '') AS campaign_name
FROM link
CROSS JOIN inserted
LEFT JOIN camp ON camp.id = inserted.campaign_id;
