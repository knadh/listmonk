
-- subscribers
-- name: get-subscriber
-- Get a single subscriber by id or UUID or email.
SELECT * FROM subscribers WHERE
    CASE
        WHEN $1 > 0 THEN id = $1
        WHEN $2 != '' THEN uuid = $2::UUID
        WHEN $3 != '' THEN email = $3
    END;

-- name: get-subscribers-by-emails
-- Get subscribers by emails.
SELECT * FROM subscribers WHERE email=ANY($1);

-- name: get-subscriber-lists
WITH sub AS (
    SELECT id FROM subscribers WHERE CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END
)
SELECT * FROM lists
    LEFT JOIN subscriber_lists ON (lists.id = subscriber_lists.list_id)
    WHERE subscriber_id = (SELECT id FROM sub)
    -- Optional list IDs or UUIDs to filter.
    AND (CASE WHEN CARDINALITY($3::INT[]) > 0 THEN id = ANY($3::INT[])
          WHEN CARDINALITY($4::UUID[]) > 0 THEN uuid = ANY($4::UUID[])
          ELSE TRUE
    END)
    AND (CASE WHEN $5 != '' THEN subscriber_lists.status = $5::subscription_status ELSE TRUE END)
    AND (CASE WHEN $6 != '' THEN lists.optin = $6::list_optin ELSE TRUE END)
    ORDER BY id;

-- name: get-subscriber-lists-lazy
-- Get lists associations of subscribers given a list of subscriber IDs.
-- This query is used to lazy load given a list of subscriber IDs.
-- The query returns results in the same order as the given subscriber IDs, and for non-existent subscriber IDs,
-- the query still returns a row with 0 values. Thus, for lazy loading, the application simply iterate on the results in
-- the same order as the list of campaigns it would've queried and attach the results.
WITH subs AS (
    SELECT subscriber_id, JSON_AGG(
        ROW_TO_JSON(
            (SELECT l FROM (
                SELECT
                    subscriber_lists.status AS subscription_status,
                    subscriber_lists.created_at AS subscription_created_at,
                    subscriber_lists.updated_at AS subscription_updated_at,
                    subscriber_lists.meta AS subscription_meta,
                    lists.*
            ) l)
        )
    ) AS lists FROM lists
    LEFT JOIN subscriber_lists ON (subscriber_lists.list_id = lists.id)
    WHERE subscriber_lists.subscriber_id = ANY($1)
    GROUP BY subscriber_id
)
SELECT id as subscriber_id,
    COALESCE(s.lists, '[]') AS lists
    FROM (SELECT id FROM UNNEST($1) AS id) x
    LEFT JOIN subs AS s ON (s.subscriber_id = id)
    ORDER BY ARRAY_POSITION($1, id);

-- name: get-subscriptions
-- Retrieves all lists a subscriber is attached to.
-- if $3 is set to true, all lists are fetched including the subscriber's subscriptions.
-- subscription_status, and subscription_created_at are null in that case.
WITH sub AS (
    SELECT id FROM subscribers WHERE CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END
)
SELECT lists.*,
    subscriber_lists.status as subscription_status,
    subscriber_lists.created_at as subscription_created_at,
    subscriber_lists.meta as subscription_meta
    FROM lists LEFT JOIN subscriber_lists
    ON (subscriber_lists.list_id = lists.id AND subscriber_lists.subscriber_id = (SELECT id FROM sub))
    WHERE CASE WHEN $3 = TRUE THEN TRUE ELSE subscriber_lists.status IS NOT NULL END
    ORDER BY subscriber_lists.status;

-- name: insert-subscriber
WITH sub AS (
    INSERT INTO subscribers (uuid, email, name, status, attribs)
    VALUES($1, $2, $3, $4, $5)
    RETURNING id, status
),
listIDs AS (
    SELECT id FROM lists WHERE
        (CASE WHEN CARDINALITY($6::INT[]) > 0 THEN id=ANY($6)
              ELSE uuid=ANY($7::UUID[]) END)
),
subs AS (
    INSERT INTO subscriber_lists (subscriber_id, list_id, status)
    VALUES(
        (SELECT id FROM sub),
        UNNEST(ARRAY(SELECT id FROM listIDs)),
        (CASE WHEN $4='blocklisted' THEN 'unsubscribed'::subscription_status ELSE $8::subscription_status END)
    )
    ON CONFLICT (subscriber_id, list_id) DO UPDATE
        SET updated_at=NOW(),
            status=(
                CASE WHEN $4='blocklisted' OR (SELECT status FROM sub)='blocklisted'
                THEN 'unsubscribed'::subscription_status
                ELSE $8::subscription_status END
            )
)
SELECT id from sub;

-- name: upsert-subscriber
-- Upserts a subscriber where existing subscribers get their names and attributes overwritten.
-- If $7 = true, update values, otherwise, skip.
WITH sub AS (
    INSERT INTO subscribers as s (uuid, email, name, attribs, status)
    VALUES($1, $2, $3, $4, 'enabled')
    ON CONFLICT (email)
    DO UPDATE SET
        name=(CASE WHEN $7 THEN $3 ELSE s.name END),
        attribs=(CASE WHEN $7 THEN $4 ELSE s.attribs END),
        updated_at=NOW()
    RETURNING uuid, id
),
subs AS (
    INSERT INTO subscriber_lists (subscriber_id, list_id, status)
    VALUES((SELECT id FROM sub), UNNEST($5::INT[]), $6)
    ON CONFLICT (subscriber_id, list_id) DO UPDATE
    SET updated_at=NOW(), status=(CASE WHEN $7 THEN $6 ELSE subscriber_lists.status END)
)
SELECT uuid, id from sub;

-- name: upsert-blocklist-subscriber
-- Upserts a subscriber where the update will only set the status to blocklisted
-- unlike upsert-subscribers where name and attributes are updated. In addition, all
-- existing subscriptions are marked as 'unsubscribed'.
-- This is used in the bulk importer.
WITH sub AS (
    INSERT INTO subscribers (uuid, email, name, attribs, status)
    VALUES($1, $2, $3, $4, 'blocklisted')
    ON CONFLICT (email) DO UPDATE SET status='blocklisted', updated_at=NOW()
    RETURNING id
)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE subscriber_id = (SELECT id FROM sub);

-- name: update-subscriber
UPDATE subscribers SET
    email=(CASE WHEN $2 != '' THEN $2 ELSE email END),
    name=(CASE WHEN $3 != '' THEN $3 ELSE name END),
    status=(CASE WHEN $4 != '' THEN $4::subscriber_status ELSE status END),
    attribs=(CASE WHEN $5 != '' THEN $5::JSONB ELSE attribs END),
    updated_at=NOW()
WHERE id = $1;

-- name: update-subscriber-with-lists
-- Updates a subscriber's data, and given a list of list_ids, inserts subscriptions
-- for them while deleting existing subscriptions not in the list.
WITH s AS (
    UPDATE subscribers SET
        email=(CASE WHEN $2 != '' THEN $2 ELSE email END),
        name=(CASE WHEN $3 != '' THEN $3 ELSE name END),
        status=(CASE WHEN $4 != '' THEN $4::subscriber_status ELSE status END),
        attribs=(CASE WHEN $5 != '' THEN $5::JSONB ELSE attribs END),
        updated_at=NOW()
    WHERE id = $1 RETURNING id
),
listIDs AS (
    SELECT id FROM lists WHERE
        (CASE WHEN CARDINALITY($6::INT[]) > 0 THEN id=ANY($6)
              ELSE uuid=ANY($7::UUID[]) END)
),
d AS (
    DELETE FROM subscriber_lists WHERE $9 = TRUE AND subscriber_id = $1 AND list_id != ALL(SELECT id FROM listIDs)
)
INSERT INTO subscriber_lists (subscriber_id, list_id, status)
    VALUES(
        (SELECT id FROM s),
        UNNEST(ARRAY(SELECT id FROM listIDs)),
        (CASE WHEN $4='blocklisted' THEN 'unsubscribed'::subscription_status ELSE $8::subscription_status END)
    )
    ON CONFLICT (subscriber_id, list_id) DO UPDATE
    SET status = (
        CASE
            WHEN $4='blocklisted' THEN 'unsubscribed'::subscription_status
            -- When subscriber is edited from the admin form, retain the status. Otherwise, a blocklisted
            -- subscriber when being re-enabled, their subscription statuses change.
            WHEN $9 = TRUE THEN subscriber_lists.status
            ELSE $8::subscription_status
        END
    );

-- name: delete-subscribers
-- Delete one or more subscribers by ID or UUID.
DELETE FROM subscribers WHERE CASE WHEN ARRAY_LENGTH($1::INT[], 1) > 0 THEN id = ANY($1) ELSE uuid = ANY($2::UUID[]) END;

-- name: delete-blocklisted-subscribers
DELETE FROM subscribers WHERE status = 'blocklisted';

-- name: delete-orphan-subscribers
DELETE FROM subscribers a WHERE NOT EXISTS
    (SELECT 1 FROM subscriber_lists b WHERE b.subscriber_id = a.id);

-- name: blocklist-subscribers
WITH b AS (
    UPDATE subscribers SET status='blocklisted', updated_at=NOW()
    WHERE id = ANY($1::INT[])
)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE subscriber_id = ANY($1::INT[]);

-- name: add-subscribers-to-lists
INSERT INTO subscriber_lists (subscriber_id, list_id, status)
    (SELECT a, b, (CASE WHEN $3 != '' THEN $3::subscription_status ELSE 'unconfirmed' END) FROM UNNEST($1::INT[]) a, UNNEST($2::INT[]) b)
    ON CONFLICT (subscriber_id, list_id) DO UPDATE SET status=(CASE WHEN $3 != '' THEN $3::subscription_status ELSE subscriber_lists.status END);

-- name: delete-subscriptions
DELETE FROM subscriber_lists
    WHERE (subscriber_id, list_id) = ANY(SELECT a, b FROM UNNEST($1::INT[]) a, UNNEST($2::INT[]) b);

-- name: confirm-subscription-optin
WITH subID AS (
    SELECT id FROM subscribers WHERE uuid = $1::UUID
),
listIDs AS (
    SELECT id FROM lists WHERE uuid = ANY($2::UUID[])
)
UPDATE subscriber_lists SET status='confirmed', meta=meta || $3, updated_at=NOW()
    WHERE subscriber_id = (SELECT id FROM subID) AND list_id = ANY(SELECT id FROM listIDs);

-- name: unsubscribe-subscribers-from-lists
WITH listIDs AS (
    SELECT ARRAY(
        SELECT id FROM lists WHERE
        (CASE WHEN CARDINALITY($2::INT[]) > 0 THEN id=ANY($2) ELSE uuid=ANY($3::UUID[]) END)
    ) id
)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE (subscriber_id, list_id) = ANY(SELECT a, b FROM UNNEST($1::INT[]) a, UNNEST((SELECT id FROM listIDs)) b);

-- name: unsubscribe-by-campaign
-- Unsubscribes a subscriber given a campaign UUID (from all the lists in the campaign) and the subscriber UUID.
-- If $3 is TRUE, then all subscriptions of the subscriber is blocklisted
-- and all existing subscriptions, irrespective of lists, unsubscribed.
WITH lists AS (
    SELECT list_id FROM campaign_lists
    LEFT JOIN campaigns ON (campaign_lists.campaign_id = campaigns.id)
    WHERE campaigns.uuid = $1
),
sub AS (
    UPDATE subscribers SET status = (CASE WHEN $3 IS TRUE THEN 'blocklisted' ELSE status END)
    WHERE uuid = $2 RETURNING id
)
UPDATE subscriber_lists SET status = 'unsubscribed', updated_at=NOW() WHERE
    subscriber_id = (SELECT id FROM sub) AND status != 'unsubscribed' AND
    -- If $3 is false, unsubscribe from the campaign's lists, otherwise all lists.
    CASE WHEN $3 IS FALSE THEN list_id = ANY(SELECT list_id FROM lists) ELSE list_id != 0 END;

-- name: delete-unconfirmed-subscriptions
WITH optins AS (
    SELECT id FROM lists WHERE optin = 'double'
)
DELETE FROM subscriber_lists
    WHERE status = 'unconfirmed' AND list_id IN (SELECT id FROM optins) AND created_at < $1;

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

-- Partial and RAW queries used to construct arbitrary subscriber
-- queries for segmentation follow.

-- name: query-subscribers
-- raw: true
-- Unprepared statement for issuring arbitrary WHERE conditions for
-- searching subscribers. While the results are sliced using offset+limit,
-- there's a COUNT() OVER() that still returns the total result count
-- for pagination in the frontend, albeit being a field that'll repeat
-- with every resultant row.
-- %s = arbitrary expression, %s = order by field, %s = order direction
SELECT subscribers.* FROM subscribers
    LEFT JOIN subscriber_lists
    ON (
        -- Optional list filtering.
        (CASE WHEN CARDINALITY($1::INT[]) > 0 THEN true ELSE false END)
        AND subscriber_lists.subscriber_id = subscribers.id
    )
    WHERE (CARDINALITY($1) = 0 OR subscriber_lists.list_id = ANY($1::INT[]))
    %query%
    ORDER BY %order% OFFSET $2 LIMIT (CASE WHEN $3 < 1 THEN NULL ELSE $3 END);

-- name: query-subscribers-count
-- Replica of query-subscribers for obtaining the results count.
SELECT COUNT(*) AS total FROM subscribers
    LEFT JOIN subscriber_lists
    ON (
        -- Optional list filtering.
        (CASE WHEN CARDINALITY($1::INT[]) > 0 THEN true ELSE false END)
        AND subscriber_lists.subscriber_id = subscribers.id
    )
    WHERE (CARDINALITY($1) = 0 OR subscriber_lists.list_id = ANY($1::INT[])) %s;

-- name: query-subscribers-for-export
-- raw: true
-- Unprepared statement for issuring arbitrary WHERE conditions for
-- searching subscribers to do bulk CSV export.
-- %s = arbitrary expression
SELECT subscribers.id,
       subscribers.uuid,
       subscribers.email,
       subscribers.name,
       subscribers.status,
       subscribers.attribs,
       subscribers.created_at,
       subscribers.updated_at
       FROM subscribers
    LEFT JOIN subscriber_lists
    ON (
        -- Optional list filtering.
        (CASE WHEN CARDINALITY($1::INT[]) > 0 THEN true ELSE false END)
        AND subscriber_lists.subscriber_id = subscribers.id
    )
    WHERE subscriber_lists.list_id = ALL($1::INT[]) AND id > $2
    AND (CASE WHEN CARDINALITY($3::INT[]) > 0 THEN id=ANY($3) ELSE true END)
    %query%
    ORDER BY subscribers.id ASC LIMIT (CASE WHEN $4 < 1 THEN NULL ELSE $4 END);

-- name: query-subscribers-template
-- raw: true
-- This raw query is reused in multiple queries (blocklist, add to list, delete)
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

-- name: blocklist-subscribers-by-query
-- raw: true
WITH subs AS (%s),
b AS (
    UPDATE subscribers SET status='blocklisted', updated_at=NOW()
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
SELECT * FROM lists WHERE (CASE WHEN $1 = '' THEN 1=1 ELSE type=$1::list_type END)
    ORDER BY CASE WHEN $2 = 'id' THEN id END, CASE WHEN $2 = 'name' THEN name END;

-- name: query-lists
WITH ls AS (
	SELECT COUNT(*) OVER () AS total, lists.* FROM lists
    WHERE
        CASE
            WHEN $1 > 0 THEN id = $1
            WHEN $2 != '' THEN uuid = $2::UUID
            WHEN $3 != '' THEN to_tsvector(name) @@ to_tsquery ($3)
            ELSE true
        END
    ORDER BY %order%
    OFFSET $4 LIMIT (CASE WHEN $5 < 1 THEN NULL ELSE $5 END)
),
counts AS (
    SELECT list_id, JSON_OBJECT_AGG(status, subscriber_count) AS subscriber_statuses FROM (
        SELECT COUNT(*) as subscriber_count, list_id, status FROM subscriber_lists
        WHERE ($1 = 0 OR list_id = $1)
        GROUP BY list_id, status
    ) row GROUP BY list_id
)
SELECT ls.*, subscriber_statuses FROM ls
    LEFT JOIN counts ON (counts.list_id = ls.id) ORDER BY %order%;


-- name: get-lists-by-optin
-- Can have a list of IDs or a list of UUIDs.
SELECT * FROM lists WHERE (CASE WHEN $1 != '' THEN optin=$1::list_optin ELSE TRUE END) AND
    (CASE WHEN $2::INT[] IS NOT NULL THEN id = ANY($2::INT[])
          WHEN $3::UUID[] IS NOT NULL THEN uuid = ANY($3::UUID[])
    END) ORDER BY name;

-- name: create-list
INSERT INTO lists (uuid, name, type, optin, tags, description) VALUES($1, $2, $3, $4, $5, $6) RETURNING id;

-- name: update-list
UPDATE lists SET
    name=(CASE WHEN $2 != '' THEN $2 ELSE name END),
    type=(CASE WHEN $3 != '' THEN $3::list_type ELSE type END),
    optin=(CASE WHEN $4 != '' THEN $4::list_optin ELSE optin END),
    tags=$5::VARCHAR(100)[],
    description=(CASE WHEN $6 != '' THEN $6 ELSE description END),
    updated_at=NOW()
WHERE id = $1;

-- name: update-lists-date
UPDATE lists SET updated_at=NOW() WHERE id = ANY($1);

-- name: delete-lists
DELETE FROM lists WHERE id = ALL($1);


-- campaigns
-- name: create-campaign
-- This creates the campaign and inserts campaign_lists relationships.
WITH campLists AS (
    -- Get the list_ids and their optin statuses for the campaigns found in the previous step.
    SELECT lists.id AS list_id, campaign_id, optin FROM lists
    INNER JOIN campaign_lists ON (campaign_lists.list_id = lists.id)
    WHERE lists.id = ANY($14::INT[])
),
tpl AS (
    -- If there's no template_id given, use the default template.
    SELECT (CASE WHEN $13 = 0 THEN id ELSE $13 END) AS id FROM templates WHERE is_default IS TRUE
),
counts AS (
    SELECT COALESCE(COUNT(id), 0) as to_send, COALESCE(MAX(id), 0) as max_sub_id
    FROM subscribers
    LEFT JOIN campLists ON (campLists.campaign_id = ANY($14::INT[]))
    LEFT JOIN subscriber_lists ON (
        subscriber_lists.status != 'unsubscribed' AND
        subscribers.id = subscriber_lists.subscriber_id AND
        subscriber_lists.list_id = campLists.list_id AND

        -- For double opt-in lists, consider only 'confirmed' subscriptions. For single opt-ins,
        -- any status except for 'unsubscribed' (already excluded above) works.
        (CASE WHEN campLists.optin = 'double' THEN subscriber_lists.status = 'confirmed' ELSE true END)
    )
    WHERE subscriber_lists.list_id=ANY($14::INT[])
    AND subscribers.status='enabled'
),
camp AS (
    INSERT INTO campaigns (uuid, type, name, subject, from_email, body, altbody, content_type, send_at, headers, tags, messenger, template_id, to_send, max_subscriber_id, archive, archive_template_id, archive_meta)
        SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
            (SELECT id FROM tpl), (SELECT to_send FROM counts),
            (SELECT max_sub_id FROM counts), $15,
            (CASE WHEN $16 = 0 THEN (SELECT id FROM tpl) ELSE $16 END), $17
        RETURNING id
),
med AS (
    INSERT INTO campaign_media (campaign_id, media_id, filename)
        (SELECT (SELECT id FROM camp), id, filename FROM media WHERE id=ANY($18::INT[]))
)
INSERT INTO campaign_lists (campaign_id, list_id, list_name)
    (SELECT (SELECT id FROM camp), id, name FROM lists WHERE id=ANY($14::INT[]))
    RETURNING (SELECT id FROM camp);

-- name: query-campaigns
-- Here, 'lists' is returned as an aggregated JSON array from campaign_lists because
-- the list reference may have been deleted.
-- While the results are sliced using offset+limit,
-- there's a COUNT() OVER() that still returns the total result count
-- for pagination in the frontend, albeit being a field that'll repeat
-- with every resultant row.
SELECT  c.id, c.uuid, c.name, c.subject, c.from_email,
        c.messenger, c.started_at, c.to_send, c.sent, c.type,
        c.body, c.altbody, c.send_at, c.headers, c.status, c.content_type, c.tags,
        c.template_id, c.archive, c.archive_template_id, c.archive_meta,
        c.created_at, c.updated_at,
        COUNT(*) OVER () AS total,
        (
            SELECT COALESCE(ARRAY_TO_JSON(ARRAY_AGG(l)), '[]') FROM (
                SELECT COALESCE(campaign_lists.list_id, 0) AS id,
                campaign_lists.list_name AS name
                FROM campaign_lists WHERE campaign_lists.campaign_id = c.id
        ) l
    ) AS lists
FROM campaigns c
WHERE ($1 = 0 OR id = $1)
    AND status=ANY(CASE WHEN CARDINALITY($2::campaign_status[]) != 0 THEN $2::campaign_status[] ELSE ARRAY[status] END)
    AND ($3 = '' OR TO_TSVECTOR(CONCAT(name, ' ', subject)) @@ TO_TSQUERY($3))
ORDER BY %order% OFFSET $4 LIMIT (CASE WHEN $5 < 1 THEN NULL ELSE $5 END);

-- name: get-campaign
SELECT campaigns.*,
    COALESCE(templates.body, (SELECT body FROM templates WHERE is_default = true LIMIT 1)) AS template_body
    FROM campaigns
    LEFT JOIN templates ON (
        CASE WHEN $3 = 'default' THEN templates.id = campaigns.template_id
        ELSE templates.id = campaigns.archive_template_id END
    )
    WHERE CASE WHEN $1 > 0 THEN campaigns.id = $1 ELSE uuid = $2 END;

-- name: get-archived-campaigns
SELECT COUNT(*) OVER () AS total, campaigns.*,
    COALESCE(templates.body, (SELECT body FROM templates WHERE is_default = true LIMIT 1)) AS template_body
    FROM campaigns
    LEFT JOIN templates ON (
        CASE WHEN $3 = 'default' THEN templates.id = campaigns.template_id
        ELSE templates.id = campaigns.archive_template_id END
    )
    WHERE campaigns.archive=true AND campaigns.type='regular' AND campaigns.status=ANY('{running, paused, finished}')
    ORDER by campaigns.created_at DESC OFFSET $1 LIMIT $2;

-- name: get-campaign-stats
-- This query is used to lazy load campaign stats (views, counts, list of lists) given a list of campaign IDs.
-- The query returns results in the same order as the given campaign IDs, and for non-existent campaign IDs,
-- the query still returns a row with 0 values. Thus, for lazy loading, the application simply iterate on the results in
-- the same order as the list of campaigns it would've queried and attach the results.
WITH lists AS (
    SELECT campaign_id, JSON_AGG(JSON_BUILD_OBJECT('id', list_id, 'name', list_name)) AS lists FROM campaign_lists
    WHERE campaign_id = ANY($1) GROUP BY campaign_id
),
media AS (
    SELECT campaign_id, JSON_AGG(JSON_BUILD_OBJECT('id', media_id, 'filename', filename)) AS media FROM campaign_media
    WHERE campaign_id = ANY($1) GROUP BY campaign_id
),
views AS (
    SELECT campaign_id, COUNT(campaign_id) as num FROM campaign_views
    WHERE campaign_id = ANY($1)
    GROUP BY campaign_id
),
clicks AS (
    SELECT campaign_id, COUNT(campaign_id) as num FROM link_clicks
    WHERE campaign_id = ANY($1)
    GROUP BY campaign_id
),
bounces AS (
    SELECT campaign_id, COUNT(campaign_id) as num FROM bounces
    WHERE campaign_id = ANY($1)
    GROUP BY campaign_id
)
SELECT id as campaign_id,
    COALESCE(v.num, 0) AS views,
    COALESCE(c.num, 0) AS clicks,
    COALESCE(b.num, 0) AS bounces,
    COALESCE(l.lists, '[]') AS lists,
    COALESCE(m.media, '[]') AS media
FROM (SELECT id FROM UNNEST($1) AS id) x
LEFT JOIN lists AS l ON (l.campaign_id = id)
LEFT JOIN media AS m ON (m.campaign_id = id)
LEFT JOIN views AS v ON (v.campaign_id = id)
LEFT JOIN clicks AS c ON (c.campaign_id = id)
LEFT JOIN bounces AS b ON (b.campaign_id = id)
ORDER BY ARRAY_POSITION($1, id);

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
LEFT JOIN templates ON (templates.id = (CASE WHEN $2=0 THEN campaigns.template_id ELSE $2 END))
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
    WHERE (status='running' OR (status='scheduled' AND NOW() >= campaigns.send_at))
    AND NOT(campaigns.id = ANY($1::INT[]))
),
campLists AS (
    -- Get the list_ids and their optin statuses for the campaigns found in the previous step.
    SELECT lists.id AS list_id, campaign_id, optin FROM lists
    INNER JOIN campaign_lists ON (campaign_lists.list_id = lists.id)
    WHERE campaign_lists.campaign_id = ANY(SELECT id FROM camps)
),
campMedia AS (
    -- Get the list_ids and their optin statuses for the campaigns found in the previous step.
    SELECT campaign_id, ARRAY_AGG(campaign_media.media_id)::INT[] AS media_id FROM campaign_media
    WHERE campaign_id = ANY(SELECT id FROM camps) AND media_id IS NOT NULL
    GROUP BY campaign_id
),
counts AS (
    -- For each campaign above, get the total number of subscribers and the max_subscriber_id
    -- across all its lists.
    SELECT id AS campaign_id,
                 COUNT(DISTINCT(subscriber_lists.subscriber_id)) AS to_send,
                 COALESCE(MAX(subscriber_lists.subscriber_id), 0) AS max_subscriber_id
    FROM camps
    LEFT JOIN campLists ON (campLists.campaign_id = camps.id)
    LEFT JOIN subscriber_lists ON (
        subscriber_lists.list_id = campLists.list_id AND
        (CASE
            -- For optin campaigns, only e-mail 'unconfirmed' subscribers belonging to 'double' optin lists.
            WHEN camps.type = 'optin' THEN subscriber_lists.status = 'unconfirmed' AND campLists.optin = 'double'

            -- For regular campaigns with double optin lists, only e-mail 'confirmed' subscribers.
            WHEN campLists.optin = 'double' THEN subscriber_lists.status = 'confirmed'

            -- For regular campaigns with non-double optin lists, e-mail everyone
            -- except unsubscribed subscribers.
            ELSE subscriber_lists.status != 'unsubscribed'
        END)
    )
    GROUP BY camps.id
),
u AS (
    -- For each campaign, update the to_send count and set the max_subscriber_id.
    UPDATE campaigns AS ca
    SET to_send = co.to_send,
        status = (CASE WHEN status != 'running' THEN 'running' ELSE status END),
        max_subscriber_id = co.max_subscriber_id,
        started_at=(CASE WHEN ca.started_at IS NULL THEN NOW() ELSE ca.started_at END)
    FROM (SELECT * FROM counts) co
    WHERE ca.id = co.campaign_id
)
SELECT camps.*, campMedia.media_id FROM camps LEFT JOIN campMedia ON (campMedia.campaign_id = camps.id);

-- name: get-campaign-analytics-unique-counts
WITH intval AS (
    -- For intervals < a week, aggregate counts hourly, otherwise daily.
    SELECT CASE WHEN (EXTRACT (EPOCH FROM ($3::TIMESTAMP - $2::TIMESTAMP)) / 86400) >= 7 THEN 'day' ELSE 'hour' END
),
uniqIDs AS (
    SELECT DISTINCT ON(subscriber_id) subscriber_id, campaign_id, DATE_TRUNC((SELECT * FROM intval), created_at) AS "timestamp"
    FROM %s
    WHERE campaign_id=ANY($1) AND created_at >= $2 AND created_at <= $3
    ORDER BY subscriber_id, "timestamp"
)
SELECT COUNT(*) AS "count", campaign_id, "timestamp"
    FROM uniqIDs GROUP BY campaign_id, "timestamp";

-- name: get-campaign-analytics-counts
-- raw: true
WITH intval AS (
    -- For intervals < a week, aggregate counts hourly, otherwise daily.
    SELECT CASE WHEN (EXTRACT (EPOCH FROM ($3::TIMESTAMP - $2::TIMESTAMP)) / 86400) >= 7 THEN 'day' ELSE 'hour' END
)
SELECT campaign_id, COUNT(*) AS "count", DATE_TRUNC((SELECT * FROM intval), created_at) AS "timestamp"
    FROM %s
    WHERE campaign_id=ANY($1) AND created_at >= $2 AND created_at <= $3
    GROUP BY campaign_id, "timestamp" ORDER BY "timestamp" ASC;

-- name: get-campaign-bounce-counts
WITH intval AS (
    -- For intervals < a week, aggregate counts hourly, otherwise daily.
    SELECT CASE WHEN (EXTRACT (EPOCH FROM ($3::TIMESTAMP - $2::TIMESTAMP)) / 86400) >= 7 THEN 'day' ELSE 'hour' END
)
SELECT campaign_id, COUNT(*) AS "count", DATE_TRUNC((SELECT * FROM intval), created_at) AS "timestamp"
    FROM bounces
    WHERE campaign_id=ANY($1) AND created_at >= $2 AND created_at <= $3
    GROUP BY campaign_id, "timestamp" ORDER BY "timestamp" ASC;

-- name: get-campaign-link-counts
-- raw: true
-- %s = * or DISTINCT subscriber_id (prepared based on based on individual tracking=on/off). Prepared on boot.
SELECT COUNT(%s) AS "count", url
    FROM link_clicks
    LEFT JOIN links ON (link_clicks.link_id = links.id)
    WHERE campaign_id=ANY($1) AND link_clicks.created_at >= $2 AND link_clicks.created_at <= $3
    GROUP BY links.url ORDER BY "count" DESC LIMIT 50;

-- name: next-campaign-subscribers
-- Returns a batch of subscribers in a given campaign starting from the last checkpoint
-- (last_subscriber_id). Every fetch updates the checkpoint and the sent count, which means
-- every fetch returns a new batch of subscribers until all rows are exhausted.
WITH camps AS (
    SELECT last_subscriber_id, max_subscriber_id, type FROM campaigns WHERE id = $1 AND status='running'
),
campLists AS (
    SELECT lists.id AS list_id, optin FROM lists
    LEFT JOIN campaign_lists ON (campaign_lists.list_id = lists.id)
    WHERE campaign_lists.campaign_id = $1
),
subIDs AS (
    SELECT DISTINCT ON (subscriber_lists.subscriber_id) subscriber_id, list_id, status FROM subscriber_lists
    WHERE
        -- ARRAY_AGG is 20x faster instead of a simple SELECT because the query planner
        -- understands the CTE's cardinality after the scalar array conversion. Huh.
        list_id = ANY((SELECT ARRAY_AGG(list_id) FROM campLists)::INT[]) AND
        status != 'unsubscribed' AND
        subscriber_id > (SELECT last_subscriber_id FROM camps) AND
        subscriber_id <= (SELECT max_subscriber_id FROM camps)
    ORDER BY subscriber_id LIMIT $2
),
subs AS (
    SELECT subscribers.* FROM subIDs
    LEFT JOIN campLists ON (campLists.list_id = subIDs.list_id)
    INNER JOIN subscribers ON (
        subscribers.status != 'blocklisted' AND
        subscribers.id = subIDs.subscriber_id AND

        (CASE
            -- For optin campaigns, only e-mail 'unconfirmed' subscribers.
            WHEN (SELECT type FROM camps) = 'optin' THEN subIDs.status = 'unconfirmed' AND campLists.optin = 'double'

            -- For regular campaigns with double optin lists, only e-mail 'confirmed' subscribers.
            WHEN campLists.optin = 'double' THEN subIDs.status = 'confirmed'

            -- For regular campaigns with non-double optin lists, e-mail everyone
            -- except unsubscribed subscribers.
            ELSE subIDs.status != 'unsubscribed'
        END)
    )
),
u AS (
    UPDATE campaigns
    SET last_subscriber_id = (SELECT MAX(id) FROM subs),
        sent = sent + (SELECT COUNT(id) FROM subs),
        updated_at = NOW()
    WHERE (SELECT COUNT(id) FROM subs) > 0 AND id=$1
)
SELECT * FROM subs;

-- name: delete-campaign-views
DELETE FROM campaign_views WHERE created_at < $1;

-- name: delete-campaign-link-clicks
DELETE FROM link_clicks WHERE created_at < $1;

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
        name=$2,
        subject=$3,
        from_email=$4,
        body=$5,
        altbody=(CASE WHEN $6 = '' THEN NULL ELSE $6 END),
        content_type=$7::content_type,
        send_at=$8::TIMESTAMP WITH TIME ZONE,
        status=(CASE WHEN NOT $9 THEN 'draft' ELSE status END),
        headers=$10,
        tags=$11::VARCHAR(100)[],
        messenger=$12,
        template_id=$13,
        archive=$15,
        archive_template_id=$16,
        archive_meta=$17,
        updated_at=NOW()
    WHERE id = $1 RETURNING id
),
clists AS (
    -- Reset list relationships
    DELETE FROM campaign_lists WHERE campaign_id = $1 AND NOT(list_id = ANY($14))
),
med AS (
    DELETE FROM campaign_media WHERE campaign_id = $1
    AND media_id IS NULL or NOT(media_id = ANY($18)) RETURNING media_id
),
medi AS (
    INSERT INTO campaign_media (campaign_id, media_id, filename)
        (SELECT $1 AS campaign_id, id, filename FROM media WHERE id=ANY($18::INT[]))
        ON CONFLICT (campaign_id, media_id) DO NOTHING
)
INSERT INTO campaign_lists (campaign_id, list_id, list_name)
    (SELECT $1 as campaign_id, id, name FROM lists WHERE id=ANY($14::INT[]))
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

-- name: update-campaign-archive
UPDATE campaigns SET
    archive=$2,
    archive_template_id=(CASE WHEN $3 > 0 THEN $3 ELSE archive_template_id END),
    archive_meta=(CASE WHEN $4::TEXT != '' THEN $4::JSONB ELSE archive_meta END),
    updated_at=NOW()
    WHERE id=$1;

-- name: delete-campaign
DELETE FROM campaigns WHERE id=$1;

-- name: register-campaign-view
WITH view AS (
    SELECT campaigns.id as campaign_id, subscribers.id AS subscriber_id FROM campaigns
    LEFT JOIN subscribers ON (CASE WHEN $2::TEXT != '' THEN subscribers.uuid = $2::UUID ELSE FALSE END)
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
SELECT id, name, type, subject, (CASE WHEN $2 = false THEN body ELSE '' END) as body,
    is_default, created_at, updated_at
    FROM templates WHERE ($1 = 0 OR id = $1) AND ($3 = '' OR type = $3::template_type)
    ORDER BY created_at;

-- name: create-template
INSERT INTO templates (name, type, subject, body) VALUES($1, $2, $3, $4) RETURNING id;

-- name: update-template
UPDATE templates SET
    name=(CASE WHEN $2 != '' THEN $2 ELSE name END),
    subject=(CASE WHEN $3 != '' THEN $3 ELSE name END),
    body=(CASE WHEN $4 != '' THEN $4 ELSE body END),
    updated_at=NOW()
WHERE id = $1;

-- name: set-default-template
WITH u AS (
    UPDATE templates SET is_default=true WHERE id=$1 AND type='campaign' RETURNING id
)
UPDATE templates SET is_default=false WHERE id != $1;

-- name: delete-template
-- Delete a template as long as there's more than one. On deletion, set all campaigns
-- with that template to the default template instead.
WITH tpl AS (
    DELETE FROM templates WHERE id = $1 AND (SELECT COUNT(id) FROM templates) > 1 AND is_default = false RETURNING id
),
def AS (
    SELECT id FROM templates WHERE is_default = true AND type='campaign' LIMIT 1
),
up AS (
    UPDATE campaigns SET template_id = (SELECT id FROM def) WHERE (SELECT id FROM tpl) > 0 AND template_id = $1
)
SELECT id FROM tpl;


-- media
-- name: insert-media
INSERT INTO media (uuid, filename, thumb, content_type, provider, meta, created_at) VALUES($1, $2, $3, $4, $5, $6, NOW()) RETURNING id;

-- name: query-media
SELECT COUNT(*) OVER () AS total, * FROM media
    WHERE ($1 = '' OR filename ILIKE $1) AND provider=$2 ORDER BY created_at DESC OFFSET $3 LIMIT $4;

-- name: get-media
SELECT * FROM media WHERE CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END;

-- name: delete-media
DELETE FROM media WHERE id=$1 RETURNING filename;

-- links
-- name: create-link
INSERT INTO links (uuid, url) VALUES($1, $2) ON CONFLICT (url) DO UPDATE SET url=EXCLUDED.url RETURNING uuid;

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

-- name: get-dashboard-charts
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
SELECT JSON_BUILD_OBJECT('link_clicks', COALESCE((SELECT * FROM clicks), '[]'),
                        'campaign_views', COALESCE((SELECT * FROM views), '[]'));

-- name: get-dashboard-counts
WITH subs AS (
    SELECT COUNT(*) AS num, status FROM subscribers GROUP BY status
)
SELECT JSON_BUILD_OBJECT('subscribers', JSON_BUILD_OBJECT(
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
                        'messages', (SELECT SUM(sent) AS messages FROM campaigns));

-- name: get-settings
SELECT JSON_OBJECT_AGG(key, value) AS settings
    FROM (
        SELECT * FROM settings ORDER BY key
    ) t;

-- name: update-settings
UPDATE settings AS s SET value = c.value
    -- For each key in the incoming JSON map, update the row with the key and its value.
    FROM(SELECT * FROM JSONB_EACH($1)) AS c(key, value) WHERE s.key = c.key;

-- name: record-bounce
-- Insert a bounce and count the bounces for the subscriber and either unsubscribe them,
WITH sub AS (
    SELECT id, status FROM subscribers WHERE CASE WHEN $1 != '' THEN uuid = $1::UUID ELSE email = $2 END
),
camp AS (
    SELECT id FROM campaigns WHERE $3 != '' AND uuid = $3::UUID
),
num AS (
    -- Add a +1 to include the current insertion that is happening.
    SELECT COUNT(*) + 1 AS num FROM bounces WHERE subscriber_id = (SELECT id FROM sub) AND type = $4
),
-- block1 and block2 will run when $8 = 'blocklist' and the number of bounces exceed $8.
block1 AS (
    UPDATE subscribers SET status='blocklisted'
    WHERE $9 = 'blocklist' AND (SELECT num FROM num) >= $8 AND id = (SELECT id FROM sub) AND (SELECT status FROM sub) != 'blocklisted'
),
block2 AS (
    UPDATE subscriber_lists SET status='unsubscribed'
    WHERE $9 = 'unsubscribe' AND (SELECT num FROM num) >= $8 AND subscriber_id = (SELECT id FROM sub) AND (SELECT status FROM sub) != 'blocklisted'
),
bounce AS (
    -- Record the bounce if the subscriber is not already blocklisted;
    INSERT INTO bounces (subscriber_id, campaign_id, type, source, meta, created_at)
    SELECT (SELECT id FROM sub), (SELECT id FROM camp), $4, $5, $6, $7
    WHERE NOT EXISTS (SELECT 1 WHERE (SELECT status FROM sub) = 'blocklisted' OR (SELECT num FROM num) > $8)
)
-- This delete  will only run when $9 = 'delete' and the number of bounces exceed $8.
DELETE FROM subscribers
    WHERE $9 = 'delete' AND (SELECT num FROM num) >= $8 AND id = (SELECT id FROM sub);

-- name: query-bounces
SELECT COUNT(*) OVER () AS total,
    bounces.id,
    bounces.type,
    bounces.source,
    bounces.meta,
    bounces.created_at,
    bounces.subscriber_id,
    subscribers.uuid AS subscriber_uuid,
    subscribers.email AS email,
    subscribers.email AS email,
    (
        CASE WHEN bounces.campaign_id IS NOT NULL
        THEN JSON_BUILD_OBJECT('id', bounces.campaign_id, 'name', campaigns.name)
        ELSE NULL END
    ) AS campaign
FROM bounces
LEFT JOIN subscribers ON (subscribers.id = bounces.subscriber_id)
LEFT JOIN campaigns ON (campaigns.id = bounces.campaign_id)
WHERE ($1 = 0 OR bounces.id = $1)
    AND ($2 = 0 OR bounces.campaign_id = $2)
    AND ($3 = 0 OR bounces.subscriber_id = $3)
    AND ($4 = '' OR bounces.source = $4)
ORDER BY %order% OFFSET $5 LIMIT $6;

-- name: delete-bounces
DELETE FROM bounces WHERE CARDINALITY($1::INT[]) = 0 OR id = ANY($1);

-- name: delete-bounces-by-subscriber
WITH sub AS (
    SELECT id FROM subscribers WHERE CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END
)
DELETE FROM bounces WHERE subscriber_id = (SELECT id FROM sub);


-- name: get-db-info
SELECT JSON_BUILD_OBJECT('version', (SELECT VERSION()),
                        'size_mb', (SELECT ROUND(pg_database_size((SELECT CURRENT_DATABASE()))/(1024^2)))) AS info;
