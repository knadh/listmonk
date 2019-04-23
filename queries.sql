-- subscribers
-- name: get-subscriber
-- Get a single subscriber by id or UUID.
SELECT * FROM subscribers WHERE CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END;

-- subscribers
-- name: get-subscribers-by-emails
-- Get subscribers by emails.
SELECT * FROM subscribers WHERE email=ANY($1);

-- name: get-subscriber-lists
-- Get lists associations of subscribers given a list of subscriber IDs.
-- This query is used to lazy load given a list of subscriber IDs.
-- The query returns results in the same order as the given subscriber IDs, and for non-existent subscriber IDs,
-- the query still returns a row with 0 values. Thus, for lazy loading, the application simply iterate on the results in
-- the same order as the list of campaigns it would've queried and attach the results.
WITH subs AS (
    SELECT subscriber_id, JSON_AGG(
        ROW_TO_JSON(
            (SELECT l FROM (SELECT subscriber_lists.status AS subscription_status, lists.*) l)
        )
    ) AS lists FROM lists
    LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = lists.id)
    WHERE subscriber_lists.subscriber_id = ANY($1)
    GROUP BY subscriber_id
)
SELECT id as subscriber_id,
    COALESCE(s.lists, '[]') AS lists
    FROM (SELECT id FROM UNNEST($1) AS id) x
    LEFT JOIN subs AS s ON (s.subscriber_id = id);

-- name: insert-subscriber
WITH sub AS (
    INSERT INTO subscribers (uuid, email, name, status, attribs)
    VALUES($1, $2, $3, $4, $5)
    returning id
),
subs AS (
    INSERT INTO subscriber_lists (subscriber_id, list_id, status)
    VALUES(
        (SELECT id FROM sub),
        UNNEST($6::INT[]),
        (CASE WHEN $4='blacklisted' THEN 'unsubscribed'::subscription_status ELSE 'unconfirmed' END)
    )
    ON CONFLICT (subscriber_id, list_id) DO UPDATE
    SET updated_at=NOW()
)
SELECT id from sub;

-- name: upsert-subscriber
-- Upserts a subscriber where existing subscribers get their names and attributes overwritten.
-- The status field is only updated when $6 = 'override_status'.
WITH sub AS (
    INSERT INTO subscribers (uuid, email, name, attribs)
    VALUES($1, $2, $3, $4)
    ON CONFLICT (email) DO UPDATE
        SET name=$3,
        attribs=$4,
        updated_at=NOW()
    RETURNING uuid, id
),
subs AS (
    INSERT INTO subscriber_lists (subscriber_id, list_id)
    VALUES((SELECT id FROM sub), UNNEST($5::INT[]))
    ON CONFLICT (subscriber_id, list_id) DO UPDATE
    SET updated_at=NOW()
)
SELECT uuid, id from sub;

-- name: upsert-blacklist-subscriber
-- Upserts a subscriber where the update will only set the status to blacklisted
-- unlike upsert-subscribers where name and attributes are updated. In addition, all
-- existing subscriptions are marked as 'unsubscribed'.
-- This is used in the bulk importer.
WITH sub AS (
    INSERT INTO subscribers (uuid, email, name, attribs, status)
    VALUES($1, $2, $3, $4, 'blacklisted')
    ON CONFLICT (email) DO UPDATE SET status='blacklisted', updated_at=NOW()
    RETURNING id
)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE subscriber_id = (SELECT id FROM sub);

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
INSERT INTO subscriber_lists (subscriber_id, list_id, status)
    VALUES(
        (SELECT id FROM s),
        UNNEST($6),
        (CASE WHEN $4='blacklisted' THEN 'unsubscribed'::subscription_status ELSE 'unconfirmed' END)
    )
    ON CONFLICT (subscriber_id, list_id) DO UPDATE
    SET status = (CASE WHEN $4='blacklisted' THEN 'unsubscribed'::subscription_status ELSE 'unconfirmed' END);

-- name: delete-subscribers
-- Delete one or more subscribers.
DELETE FROM subscribers WHERE id = ANY($1);

-- name: blacklist-subscribers
WITH b AS (
    UPDATE subscribers SET status='blacklisted', updated_at=NOW()
    WHERE id = ANY($1::INT[])
)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE subscriber_id = ANY($1::INT[]);

-- name: add-subscribers-to-lists
INSERT INTO subscriber_lists (subscriber_id, list_id)
    (SELECT a, b FROM UNNEST($1::INT[]) a, UNNEST($2::INT[]) b)
    ON CONFLICT (subscriber_id, list_id) DO NOTHING;

-- name: delete-subscriptions
DELETE FROM subscriber_lists
    WHERE (subscriber_id, list_id) = ANY(SELECT a, b FROM UNNEST($1::INT[]) a, UNNEST($2::INT[]) b);

-- name: unsubscribe-subscribers-from-lists
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE (subscriber_id, list_id) = ANY(SELECT a, b FROM UNNEST($1::INT[]) a, UNNEST($2::INT[]) b);

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

-- Partial and RAW queries used to construct arbitrary subscriber
-- queries for segmentation follow.

-- name: query-subscribers
-- raw: true
-- Unprepared statement for issuring arbitrary WHERE conditions for
-- searching subscribers. While the results are sliced using offset+limit,
-- there's a COUNT() OVER() that still returns the total result count
-- for pagination in the frontend, albeit being a field that'll repeat
-- with every resultant row.
SELECT COUNT(*) OVER () AS total, subscribers.* FROM subscribers
    LEFT JOIN subscriber_lists
    ON (
        -- Optional list filtering.
        (CASE WHEN CARDINALITY($1::INT[]) > 0 THEN true ELSE false END)
        AND subscriber_lists.subscriber_id = subscribers.id
    )
    WHERE subscriber_lists.list_id = ALL($1::INT[])
    %s
    ORDER BY $2 DESC OFFSET $3 LIMIT $4;

-- name: query-subscribers-template
-- raw: true
-- This raw query is reused in multiple queries (blacklist, add to list, delete)
-- etc., so it's kept has a raw template to be injected into other raw queries,
-- and for the same reason, it is not terminated with a semicolon.
--
-- All queries that embed this query should expect
-- $1=true/false (dry-run or not) and $2=[]INT (option list IDs).
-- That is, their positional arguments should start from $3.
SELECT subscribers.id FROM subscribers
LEFT JOIN subscriber_lists
ON (
    -- Optional list filtering.
    (CASE WHEN CARDINALITY($2::INT[]) > 0 THEN true ELSE false END)
    AND subscriber_lists.subscriber_id = subscribers.id
)
WHERE subscriber_lists.list_id = ALL($2::INT[]) %s
LIMIT (CASE WHEN $1 THEN 1 END)

-- name: delete-subscribers-by-query
-- raw: true
WITH subs AS (%s)
DELETE FROM subscribers WHERE id=ANY(SELECT id FROM subs);

-- name: blacklist-subscribers-by-query
-- raw: true
WITH subs AS (%s),
b AS (
    UPDATE subscribers SET status='blacklisted', updated_at=NOW()
    WHERE id = ANY(SELECT id FROM subs)
)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE subscriber_id = ANY(SELECT id FROM subs);

-- name: add-subscribers-to-lists-by-query
-- raw: true
WITH subs AS (%s)
INSERT INTO subscriber_lists (subscriber_id, list_id)
    (SELECT a, b FROM UNNEST(ARRAY(SELECT id FROM subs)) a, UNNEST($3::INT[]) b)
    ON CONFLICT (subscriber_id, list_id) DO NOTHING;

-- name: delete-subscriptions-by-query
-- raw: true
WITH subs AS (%s)
DELETE FROM subscriber_lists
    WHERE (subscriber_id, list_id) = ANY(SELECT a, b FROM UNNEST(ARRAY(SELECT id FROM subs)) a, UNNEST($3::INT[]) b);

-- name: unsubscribe-subscribers-from-lists-by-query
-- raw: true
WITH subs AS (%s)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE (subscriber_id, list_id) = ANY(SELECT a, b FROM UNNEST(ARRAY(SELECT id FROM subs)) a, UNNEST($3::INT[]) b);


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

-- name: query-campaigns
-- Here, 'lists' is returned as an aggregated JSON array from campaign_lists because
-- the list reference may have been deleted.
-- While the results are sliced using offset+limit,
-- there's a COUNT() OVER() that still returns the total result count
-- for pagination in the frontend, albeit being a field that'll repeat
-- with every resultant row.
SELECT COUNT(*) OVER () AS total, campaigns.*, (
        SELECT COALESCE(ARRAY_TO_JSON(ARRAY_AGG(l)), '[]') FROM (
            SELECT COALESCE(campaign_lists.list_id, 0) AS id,
            campaign_lists.list_name AS name
            FROM campaign_lists WHERE campaign_lists.campaign_id = campaigns.id
        ) l
    ) AS lists
FROM campaigns
WHERE ($1 = 0 OR id = $1)
    AND status=ANY(CASE WHEN ARRAY_LENGTH($2::campaign_status[], 1) != 0 THEN $2::campaign_status[] ELSE ARRAY[status] END)
    AND ($3 = '' OR (to_tsvector(name || subject) @@ to_tsquery($3)))
ORDER BY created_at DESC OFFSET $4 LIMIT $5;

-- name: get-campaign
SELECT * FROM campaigns WHERE id = $1;

-- name: get-campaign-stats
-- This query is used to lazy load campaign stats (views, counts, list of lists) given a list of campaign IDs.
-- The query returns results in the same order as the given campaign IDs, and for non-existent campaign IDs,
-- the query still returns a row with 0 values. Thus, for lazy loading, the application simply iterate on the results in
-- the same order as the list of campaigns it would've queried and attach the results.
WITH lists AS (
    SELECT campaign_id, JSON_AGG(JSON_BUILD_OBJECT('id', list_id, 'name', list_name)) AS lists FROM campaign_lists
    WHERE campaign_id = ANY($1) GROUP BY campaign_id
), views AS (
    SELECT campaign_id, COUNT(campaign_id) as num FROM campaign_views
    WHERE campaign_id = ANY($1)
    GROUP BY campaign_id
),
clicks AS (
    SELECT campaign_id, COUNT(campaign_id) as num FROM link_clicks
    WHERE campaign_id = ANY($1)
    GROUP BY campaign_id
)
SELECT id as campaign_id,
    COALESCE(v.num, 0) AS views,
    COALESCE(c.num, 0) AS clicks,
    COALESCE(l.lists, '[]') AS lists
FROM (SELECT id FROM UNNEST($1) AS id) x
LEFT JOIN lists AS l ON (l.campaign_id = id)
LEFT JOIN views AS v ON (v.campaign_id = id)
LEFT JOIN clicks AS c ON (c.campaign_id = id);

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

-- name: get-campaign-status
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
    LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = campaign_lists.list_id AND subscriber_lists.status != 'unsubscribed')
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
    SELECT DISTINCT ON(id) id AS uniq_id, * FROM subscribers
    LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id AND subscriber_lists.status != 'unsubscribed')
    WHERE subscriber_lists.list_id=ANY(
        SELECT list_id FROM campaign_lists where campaign_id=$1 AND list_id IS NOT NULL
    )
    AND subscribers.status != 'blacklisted'
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
ORDER BY RANDOM() LIMIT 1;

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
d AS (
    -- Reset list relationships
    DELETE FROM campaign_lists WHERE campaign_id = $1 AND NOT(list_id = ANY($10))
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

-- name: register-campaign-view
WITH view AS (
    SELECT campaigns.id as campaign_id, subscribers.id AS subscriber_id FROM campaigns
    LEFT JOIN subscribers ON (subscribers.uuid = $2)
    WHERE campaigns.uuid = $1
)
INSERT INTO campaign_views (campaign_id, subscriber_id)
    VALUES((SELECT campaign_id FROM view), (SELECT subscriber_id FROM view));

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
-- Delete a template as long as there's more than one. One deletion, set all campaigns
-- with that template to the default template instead.
WITH tpl AS (
    DELETE FROM templates WHERE id = $1 AND (SELECT COUNT(id) FROM templates) > 1 AND is_default = false RETURNING id
),
def AS (
    SELECT id FROM templates WHERE is_default = true LIMIT 1
)
UPDATE campaigns SET template_id = (SELECT id FROM def) WHERE (SELECT id FROM tpl) > 0 AND template_id = $1
    RETURNING (SELECT id FROM tpl);


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


-- name: get-dashboard-stats
WITH lists AS (
    SELECT JSON_OBJECT_AGG(type, num) FROM (SELECT type, COUNT(id) AS num FROM lists GROUP BY type) row
),
subs AS (
    SELECT JSON_OBJECT_AGG(status, num) FROM (SELECT status, COUNT(id) AS num FROM subscribers GROUP by status) row
),
orphans AS (
    SELECT COUNT(id) FROM subscribers LEFT JOIN subscriber_lists ON (subscribers.id = subscriber_lists.subscriber_id)
    WHERE subscriber_lists.subscriber_id IS NULL
),
camps AS (
    SELECT JSON_OBJECT_AGG(status, num) FROM (SELECT status, COUNT(id) AS num FROM campaigns GROUP by status) row
),
clicks AS (
    -- Clicks by day for the last 3 months
    SELECT JSON_AGG(ROW_TO_JSON(row))
    FROM (SELECT COUNT(*) AS count, created_at::DATE as date
          FROM link_clicks GROUP by date ORDER BY date DESC LIMIT 100
    ) row
),
views AS (
    -- Views by day for the last 3 months
    SELECT JSON_AGG(ROW_TO_JSON(row))
    FROM (SELECT COUNT(*) AS count, created_at::DATE as date
          FROM campaign_views GROUP by date ORDER BY date DESC LIMIT 100
    ) row
)
SELECT JSON_BUILD_OBJECT('lists', COALESCE((SELECT * FROM lists), '[]'),
                        'subscribers', COALESCE((SELECT * FROM subs), '[]'),
                        'orphan_subscribers', (SELECT * FROM orphans),
                        'campaigns', COALESCE((SELECT * FROM camps), '[]'),
                        'link_clicks', COALESCE((SELECT * FROM clicks), '[]'),
                        'campaign_views', COALESCE((SELECT * FROM views), '[]')) AS stats;
