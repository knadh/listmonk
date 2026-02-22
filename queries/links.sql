-- links
-- name: create-link
INSERT INTO links (uuid, url) VALUES($1, $2) ON CONFLICT (url) DO UPDATE SET url=EXCLUDED.url RETURNING uuid;

-- name: get-link-url
SELECT url FROM links WHERE uuid = $1;

-- name: register-link-click
WITH link AS(
    SELECT id, url FROM links WHERE uuid = $1
)
INSERT INTO link_clicks (campaign_id, subscriber_id, link_id) VALUES(
    (SELECT id FROM campaigns WHERE uuid = $2),
    (SELECT id FROM subscribers WHERE
        (CASE WHEN $3::TEXT != '' THEN subscribers.uuid = $3::UUID ELSE FALSE END)
    ),
    (SELECT id FROM link)
) RETURNING (SELECT url FROM link);
