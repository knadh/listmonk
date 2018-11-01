-- subscribers
-- name: get-subscriber
-- Get a single subscriber by id or UUID.
SELECT * FROM subscribers WHERE CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END;

-- subscribers
-- name: get-subscribers-by-emails
-- Get subscribers by emails.
SELECT * FROM subscribers WHERE email=ANY($1);

-- name: get-subscriber-lists
-- Get lists belonging to subscribers.
SELECT lists.*, subscriber_lists.subscriber_id, subscriber_lists.status AS subscription_status FROM lists
    LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = lists.id)
    WHERE subscriber_lists.subscriber_id = ANY($1::INT[]);

-- name: query-subscribers
-- raw: true
-- Unprepared statement for issuring arbitrary WHERE conditions.
SELECT * FROM subscribers WHERE 1=1 %s order by updated_at DESC OFFSET %d LIMIT %d;

-- name: query-subscribers-count
-- raw: true
SELECT COUNT(id) as num FROM subscribers WHERE 1=1 %s;

-- name: query-subscribers-by-list
-- raw: true
-- Unprepared statement for issuring arbitrary WHERE conditions.
SELECT subscribers.* FROM subscribers INNER JOIN subscriber_lists
    ON (subscriber_lists.subscriber_id = subscribers.id)
    WHERE subscriber_lists.list_id = %d
    %s
    ORDER BY id DESC OFFSET %d LIMIT %d;

-- name: query-subscribers-by-list-count
-- raw: true
SELECT COUNT(subscribers.id) as num FROM subscribers INNER JOIN subscriber_lists
    ON (subscriber_lists.subscriber_id = subscribers.id)
    WHERE subscriber_lists.list_id = %d
    %s;

-- name: upsert-subscriber
-- In case of updates, if $6 (override_status) is true, only then, the existing
-- value is overwritten with the incoming value. This is used for insertions and bulk imports.
WITH s AS (
    INSERT INTO subscribers (uuid, email, name, status, attribs)
    VALUES($1, $2, $3, $4, $5) ON CONFLICT (email) DO UPDATE
    SET name=$3, status=(CASE WHEN $6 = true THEN $4 ELSE subscribers.status END),
    attribs=$5, updated_at=NOW()
    RETURNING id
)  INSERT INTO subscriber_lists (subscriber_id, list_id)
    VALUES((SELECT id FROM s), UNNEST($7::INT[]) )
    ON CONFLICT (subscriber_id, list_id) DO NOTHING
    RETURNING subscriber_id;

-- name: update-subscriber
-- Updates a subscriber's data, and given a list of list_ids, inserts subscriptions
-- for them while deleting existing subscriptions not in the list.
WITH s AS (
    UPDATE subscribers SET
        email=(CASE WHEN $2 != '' THEN $2 ELSE email END),
        name=(CASE WHEN $3 != '' THEN $3 ELSE name END),
        status=(CASE WHEN $4 != '' THEN $4::subscriber_status ELSE status END),
        attribs=(CASE WHEN $5::TEXT != '' THEN $5::JSONB ELSE attribs END),
        updated_at=NOW()
    WHERE id = $1 RETURNING id
),
d AS (
    DELETE FROM subscriber_lists WHERE subscriber_id = $1 AND list_id != ALL($6)
)
INSERT INTO subscriber_lists (subscriber_id, list_id)
    VALUES( (SELECT id FROM s), UNNEST($6) )
    ON CONFLICT (subscriber_id, list_id) DO NOTHING;

-- name: delete-subscribers
-- Delete one or more subscribers.
DELETE FROM subscribers WHERE id = ALL($1);

-- name: unsubscribe
-- Unsubscribes a subscriber given a campaign UUID (from all the lists in the campaign) and the subscriber UUID.
-- If $3 is TRUE, then all subscriptions of the subscriber is blacklisted
-- and all existing subscriptions, irrespective of lists, unsubscribed.
WITH lists AS (
    SELECT list_id FROM campaign_lists
    LEFT JOIN campaigns ON (campaign_lists.campaign_id = campaigns.id)
    WHERE campaigns.uuid = $1
),
sub AS (
    UPDATE subscribers SET status = (CASE WHEN $3 IS TRUE THEN 'blacklisted' ELSE status END)
    WHERE uuid = $2 RETURNING id
)
UPDATE subscriber_lists SET status = 'unsubscribed' WHERE
    subscriber_id = (SELECT id FROM sub) AND status != 'unsubscribed' AND
    -- If $3 is false, unsubscribe from the campaign's lists, otherwise all lists.
    CASE WHEN $3 IS FALSE THEN list_id = ANY(SELECT list_id FROM lists) ELSE list_id != 0 END;

-- name: query-subscribers-into-lists
-- raw: true
-- Unprepared statement for issuring arbitrary WHERE conditions and getting
-- the resultant subscriber IDs into subscriber_lists.
WITH subs AS (
    SELECT id FROM subscribers WHERE status != 'blacklisted' %s
)
INSERT INTO subscriber_lists (subscriber_id, list_id)
    (SELECT id, UNNEST($1::INT[]) FROM subs)
    ON CONFLICT (subscriber_id, list_id) DO NOTHING;

-- lists
-- name: get-lists
SELECT lists.*, COUNT(subscriber_lists.subscriber_id) AS subscriber_count
    FROM lists LEFT JOIN subscriber_lists
	ON (subscriber_lists.list_id = lists.id AND subscriber_lists.status != 'unsubscribed')
    WHERE ($1 = 0 OR id = $1)
    GROUP BY lists.id ORDER BY lists.created_at;

-- name: create-list
INSERT INTO lists (uuid, name, type, tags) VALUES($1, $2, $3, $4) RETURNING id;

-- name: update-list
UPDATE lists SET
    name=(CASE WHEN $2 != '' THEN $2 ELSE name END),
    type=(CASE WHEN $3 != '' THEN $3::list_type ELSE type END),
    tags=(CASE WHEN ARRAY_LENGTH($4::VARCHAR(100)[], 1) > 0 THEN $4 ELSE tags END),
    updated_at=NOW()
WHERE id = $1;

-- name: delete-lists
DELETE FROM lists WHERE id = ALL($1);


-- campaigns
-- name: create-campaign
-- This creates the campaign and inserts campaign_lists relationships.
WITH counts AS (
    SELECT COALESCE(COUNT(id), 0) as to_send, COALESCE(MAX(id), 0) as max_sub_id
    FROM subscribers
    LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id)
    WHERE subscriber_lists.list_id=ANY($11::INT[])
    AND subscribers.status='enabled'
),
camp AS (
    INSERT INTO campaigns (uuid, name, subject, from_email, body, content_type, send_at, tags, messenger, template_id, to_send, max_subscriber_id)
        SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
                (SELECT to_send FROM counts),
                (SELECT max_sub_id FROM counts)
        RETURNING id
)
INSERT INTO campaign_lists (campaign_id, list_id, list_name)
    (SELECT (SELECT id FROM camp), id, name FROM lists WHERE id=ANY($11::INT[]))
    RETURNING (SELECT id FROM camp);

-- name: get-campaigns
-- Here, 'lists' is returned as an aggregated JSON array from campaign_lists because
-- the list reference may have been deleted.
SELECT campaigns.*, (
	SELECT COALESCE(ARRAY_TO_JSON(ARRAY_AGG(l)), '[]') FROM (
		SELECT COALESCE(campaign_lists.list_id, 0) AS id,
        campaign_lists.list_name AS name
        FROM campaign_lists WHERE campaign_lists.campaign_id = campaigns.id
	) l
) AS lists
FROM campaigns
WHERE ($1 = 0 OR id = $1) AND status=(CASE WHEN $2 != '' THEN $2::campaign_status ELSE status END)
ORDER BY created_at DESC OFFSET $3 LIMIT $4;

-- name: get-campaign-for-preview
SELECT campaigns.*, COALESCE(templates.body, (SELECT body FROM templates WHERE is_default = true LIMIT 1)) AS template_body,
(
	SELECT COALESCE(ARRAY_TO_JSON(ARRAY_AGG(l)), '[]') FROM (
		SELECT COALESCE(campaign_lists.list_id, 0) AS id,
        campaign_lists.list_name AS name
        FROM campaign_lists WHERE campaign_lists.campaign_id = campaigns.id
	) l
) AS lists
FROM campaigns
LEFT JOIN templates ON (templates.id = campaigns.template_id)
WHERE campaigns.id = $1;

-- name: get-campaign-stats
SELECT id, status, to_send, sent, started_at, updated_at
    FROM campaigns
    WHERE status=$1;

-- name: next-campaigns
-- Retreives campaigns that are running (or scheduled and the time's up) and need
-- to be processed. It updates the to_send count and max_subscriber_id of the campaign,
-- that is, the total number of subscribers to be processed across all lists of a campaign.
-- Thus, it has a sideaffect.
-- In addition, it finds the max_subscriber_id, the upper limit across all lists of
-- a campaign. This is used to fetch and slice subscribers for the campaign in next-subscriber-campaigns.
WITH camps AS (
    -- Get all running campaigns and their template bodies (if the template's deleted, the default template body instead)
    SELECT campaigns.*, COALESCE(templates.body, (SELECT body FROM templates WHERE is_default = true LIMIT 1)) AS template_body
    FROM campaigns
    LEFT JOIN templates ON (templates.id = campaigns.template_id)
    WHERE (status='running' OR (status='scheduled' AND campaigns.send_at >= NOW()))
    AND NOT(campaigns.id = ANY($1::INT[]))
),
counts AS (
    -- For each campaign above, get the total number of subscribers and the max_subscriber_id across all its lists.
    SELECT id AS campaign_id, COUNT(subscriber_lists.subscriber_id) AS to_send,
        COALESCE(MAX(subscriber_lists.subscriber_id), 0) AS max_subscriber_id FROM camps
    LEFT JOIN campaign_lists ON (campaign_lists.campaign_id = camps.id)
    LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = campaign_lists.list_id)
    WHERE campaign_lists.campaign_id = ANY(SELECT id FROM camps)
    GROUP BY camps.id
),
u AS (
    -- For each campaign above, update the to_send count.
    UPDATE campaigns AS ca
    SET to_send = co.to_send,
    max_subscriber_id = co.max_subscriber_id,
    started_at=(CASE WHEN ca.started_at IS NULL THEN NOW() ELSE ca.started_at END)
    FROM (SELECT * FROM counts) co
    WHERE ca.id = co.campaign_id
)
SELECT * FROM camps;

-- name: next-campaign-subscribers
-- Returns a batch of subscribers in a given campaign starting from the last checkpoint
-- (last_subscriber_id). Every fetch updates the checkpoint and the sent count, which means
-- every fetch returns a new batch of subscribers until all rows are exhausted.
WITH camp AS (
    SELECT last_subscriber_id, max_subscriber_id
    FROM campaigns
    WHERE id=$1 AND status='running'
),
subs AS (
    SELECT * FROM subscribers
    LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id AND subscriber_lists.status != 'unsubscribed')
    WHERE subscriber_lists.list_id=ANY(
        SELECT list_id FROM campaign_lists where campaign_id=$1 AND list_id IS NOT NULL
    )
    AND id > (SELECT last_subscriber_id FROM camp)
    AND id <= (SELECT max_subscriber_id FROM camp)
    ORDER BY id LIMIT $2
),
u AS (
    UPDATE campaigns
    SET last_subscriber_id=(SELECT MAX(id) FROM subs),
        sent=sent + (SELECT COUNT(id) FROM subs),
        updated_at=NOW()
    WHERE (SELECT COUNT(id) FROM subs) > 0 AND id=$1
)
SELECT * FROM subs;

-- name: get-one-campaign-subscriber
SELECT * FROM subscribers
LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id AND subscriber_lists.status != 'unsubscribed')
WHERE subscriber_lists.list_id=ANY(
    SELECT list_id FROM campaign_lists where campaign_id=$1 AND list_id IS NOT NULL
)
LIMIT 1;

-- name: update-campaign
WITH camp AS (
    UPDATE campaigns SET
        name=(CASE WHEN $2 != '' THEN $2 ELSE name END),
        subject=(CASE WHEN $3 != '' THEN $3 ELSE subject END),
        from_email=(CASE WHEN $4 != '' THEN $4 ELSE from_email END),
        body=(CASE WHEN $5 != '' THEN $5 ELSE body END),
        content_type=(CASE WHEN $6 != '' THEN $6::content_type ELSE content_type END),
        send_at=(CASE WHEN $7 != '' THEN $7::TIMESTAMP WITH TIME ZONE ELSE send_at END),
        tags=(CASE WHEN ARRAY_LENGTH($8::VARCHAR(100)[], 1) > 0 THEN $8 ELSE tags END),
        template_id=(CASE WHEN $9 != 0 THEN $9 ELSE template_id END),
        updated_at=NOW()
    WHERE id = $1 RETURNING id
),
    -- Reset the relationships
d AS (
    DELETE FROM campaign_lists WHERE campaign_id = $1
)
INSERT INTO campaign_lists (campaign_id, list_id, list_name)
    (SELECT $1 as campaign_id, id, name FROM lists WHERE id=ANY($10::INT[]))
    ON CONFLICT (campaign_id, list_id) DO UPDATE SET list_name = EXCLUDED.list_name;

-- name: update-campaign-counts
UPDATE campaigns SET
    to_send=(CASE WHEN $2 != 0 THEN $2 ELSE to_send END),
    sent=(CASE WHEN $3 != 0 THEN $3 ELSE sent END),
    last_subscriber_id=(CASE WHEN $4 != 0 THEN $4 ELSE last_subscriber_id END),
    updated_at=NOW()
WHERE id=$1;

-- name: update-campaign-status
UPDATE campaigns SET status=$2, updated_at=NOW() WHERE id = $1;

-- name: delete-campaign
DELETE FROM campaigns WHERE id=$1 AND (status = 'draft' OR status = 'scheduled');

-- users
-- name: get-users
SELECT * FROM users WHERE $1 = 0 OR id = $1 OFFSET $2 LIMIT $3;

-- name: create-user
INSERT INTO users (email, name, password, type, status) VALUES($1, $2, $3, $4, $5) RETURNING id;

-- name: update-user
UPDATE users SET
    email=(CASE WHEN $2 != '' THEN $2 ELSE email END),
    name=(CASE WHEN $3 != '' THEN $3 ELSE name END),
    password=(CASE WHEN $4 != '' THEN $4 ELSE password END),
    type=(CASE WHEN $5 != '' THEN $5::user_type ELSE type END),
    status=(CASE WHEN $6 != '' THEN $6::user_status ELSE status END),
    updated_at=NOW()
WHERE id = $1;

-- name: delete-user
-- Delete a user, except for the primordial super admin.
DELETE FROM users WHERE $1 != 1 AND id=$1;


-- templates
-- name: get-templates
-- Only if the second param ($2) is true, body is returned.
SELECT id, name, (CASE WHEN $2 = false THEN body ELSE '' END) as body,
    is_default, created_at, updated_at
    FROM templates WHERE $1 = 0 OR id = $1
    ORDER BY created_at;

-- name: create-template
INSERT INTO templates (name, body) VALUES($1, $2) RETURNING id;

-- name: update-template
UPDATE templates SET
    name=(CASE WHEN $2 != '' THEN $2 ELSE name END),
    body=(CASE WHEN $3 != '' THEN $3 ELSE body END),
    updated_at=NOW()
WHERE id = $1;

-- name: set-default-template
WITH u AS (
    UPDATE templates SET is_default=true WHERE id=$1 RETURNING id
)
UPDATE templates SET is_default=false WHERE id != $1;

-- name: delete-template
-- Delete a template as long as there's more than one.
DELETE FROM templates WHERE id=$1 AND (SELECT COUNT(id) FROM templates) > 1 AND is_default = false;

-- media
-- name: insert-media
INSERT INTO media (uuid, filename, thumb, width, height, created_at) VALUES($1, $2, $3, $4, $5, NOW());

-- name: get-media
SELECT * FROM media ORDER BY created_at DESC;

-- name: delete-media
DELETE FROM media WHERE id=$1 RETURNING filename;

-- links
-- name: create-link
INSERT INTO links (uuid, url) VALUES($1, $2) ON CONFLICT (url) DO UPDATE SET url=EXCLUDED.url RETURNING uuid;

-- name: register-link-click
WITH link AS (
    SELECT url, links.id AS link_id, campaigns.id as campaign_id, subscribers.id AS subscriber_id FROM links
    LEFT JOIN campaigns ON (campaigns.uuid = $2)
    LEFT JOIN subscribers ON (subscribers.uuid = $3)
    WHERE links.uuid = $1
)
INSERT INTO link_clicks (campaign_id, subscriber_id, link_id)
    VALUES((SELECT campaign_id FROM link), (SELECT subscriber_id FROM link), (SELECT link_id FROM link))
    RETURNING (SELECT url FROM link);


-- -- name: get-stats
-- WITH lists AS (
--     SELECT type, COUNT(id) AS num FROM lists GROUP BY type
-- ),
-- subs AS (
--     SELECT status, COUNT(id) AS num FROM subscribers GROUP by status
-- ),
-- orphans AS (
--     SELECT COUNT(id) FROM subscribers LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id)
--     WHERE subscriber_lists.subscriber_id IS NULL
-- ),
-- camps AS (
--     SELECT status, COUNT(id) AS num FROM campaigns GROUP by status
-- )
-- SELECT JSON_BUILD_OBJECT('lists', lists);
-- row_to_json(t)
-- from (
--   select type, num from lists
-- ) t,