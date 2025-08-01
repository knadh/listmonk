-- subscribers
-- name: get-subscriber
-- Get a single subscriber by id or UUID or email.
SELECT * FROM subscribers WHERE
    CASE
        WHEN $1 > 0 THEN id = $1
        WHEN $2 != '' THEN uuid = $2::UUID
        WHEN $3 != '' THEN email = $3
    END;

-- name: has-subscriber-list
-- Used for checking access permission by list.
SELECT s.id AS subscriber_id,
    CASE
        WHEN EXISTS (SELECT 1 FROM subscriber_lists sl WHERE sl.subscriber_id = s.id AND sl.list_id = ANY($2))
        THEN TRUE
        ELSE FALSE
    END AS has
FROM subscribers s WHERE s.id = ANY($1);

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
    RETURNING uuid, id, status
),
subs AS (
    INSERT INTO subscriber_lists (subscriber_id, list_id, status)
    SELECT sub.id, listID, CASE WHEN sub.status = 'blocklisted' THEN 'unsubscribed' ELSE $6::subscription_status END
    FROM sub, UNNEST($5::INT[]) AS listID
    ON CONFLICT (subscriber_id, list_id) DO UPDATE
    SET updated_at = NOW(),
        status = CASE WHEN $7 THEN EXCLUDED.status ELSE subscriber_lists.status END
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
            WHEN subscriber_lists.status = 'confirmed' THEN 'confirmed'
            WHEN subscriber_lists.status = 'unsubscribed' THEN 'unsubscribed'::subscription_status
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
SELECT subscribers.* FROM subscribers
    LEFT JOIN subscriber_lists
    ON (
        -- Optional list filtering.
        (CASE WHEN CARDINALITY($1::INT[]) > 0 THEN true ELSE false END)
        AND subscriber_lists.subscriber_id = subscribers.id
        AND ($2 = '' OR subscriber_lists.status = $2::subscription_status)
    )
    WHERE (CARDINALITY($1) = 0 OR subscriber_lists.list_id = ANY($1::INT[]))
    AND (CASE WHEN $3 != '' THEN name ~* $3 OR email ~* $3 ELSE TRUE END)
    AND %query%
    ORDER BY %order% OFFSET $4 LIMIT (CASE WHEN $5 < 1 THEN NULL ELSE $5 END);

-- name: query-subscribers-count
-- Replica of query-subscribers for obtaining the results count.
SELECT COUNT(*) AS total FROM subscribers
    LEFT JOIN subscriber_lists
    ON (
        -- Optional list filtering.
        (CASE WHEN CARDINALITY($1::INT[]) > 0 THEN true ELSE false END)
        AND subscriber_lists.subscriber_id = subscribers.id
        AND ($2 = '' OR subscriber_lists.status = $2::subscription_status)
    )
    WHERE (CARDINALITY($1) = 0 OR subscriber_lists.list_id = ANY($1::INT[]))
    AND (CASE WHEN $3 != '' THEN name ~* $3 OR email ~* $3 ELSE TRUE END)
    AND %query%;

-- name: query-subscribers-count-all
-- Cached query for getting the "all" subscriber count without arbitrary conditions.
SELECT COALESCE(SUM(subscriber_count), 0) AS total FROM mat_list_subscriber_stats
    WHERE list_id = ANY(CASE WHEN CARDINALITY($1::INT[]) > 0 THEN $1 ELSE '{0}' END)
    AND ($2 = '' OR status = $2::subscription_status);

-- name: query-subscribers-for-export
-- raw: true
-- Unprepared statement for issuring arbitrary WHERE conditions for
-- searching subscribers to do bulk CSV export.
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
        AND ($4 = '' OR subscriber_lists.status = $4::subscription_status)
    )
    WHERE subscriber_lists.list_id = ALL($1::INT[]) AND id > $2
    AND (CASE WHEN CARDINALITY($3::INT[]) > 0 THEN id=ANY($3) ELSE true END)
    AND (CASE WHEN $5 != '' THEN name ~* $5 OR email ~* $5 ELSE TRUE END)
    AND %query%
    ORDER BY subscribers.id ASC LIMIT (CASE WHEN $6 < 1 THEN NULL ELSE $6 END);

-- name: query-subscribers-template
-- raw: true
-- This raw query is reused in multiple queries (blocklist, add to list, delete)
-- etc., so it's kept has a raw template to be injected into other raw queries,
-- and for the same reason, it is not terminated with a semicolon.
--
-- All queries that embed this query should expect
-- $1=true/false (dry-run or not) and $2=[]INT (option list IDs).
-- That is, their positional arguments should start from $4.
SELECT subscribers.id FROM subscribers
LEFT JOIN subscriber_lists
ON (
    -- Optional list filtering.
    (CASE WHEN CARDINALITY($2::INT[]) > 0 THEN true ELSE false END)
    AND subscriber_lists.subscriber_id = subscribers.id
    AND ($3 = '' OR subscriber_lists.status = $3::subscription_status)
)
WHERE subscriber_lists.list_id = ALL($2::INT[])
    AND (CASE WHEN $4 != '' THEN name ~* $4 OR email ~* $4 ELSE TRUE END)
    AND %query%
LIMIT (CASE WHEN $1 THEN 1 END)

-- name: delete-subscribers-by-query
-- raw: true
WITH subs AS (%query%)
DELETE FROM subscribers WHERE id=ANY(SELECT id FROM subs);

-- name: blocklist-subscribers-by-query
-- raw: true
WITH subs AS (%query%),
b AS (
    UPDATE subscribers SET status='blocklisted', updated_at=NOW()
    WHERE id = ANY(SELECT id FROM subs)
)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE subscriber_id = ANY(SELECT id FROM subs);

-- name: add-subscribers-to-lists-by-query
-- raw: true
WITH subs AS (%query%)
INSERT INTO subscriber_lists (subscriber_id, list_id, status)
    (SELECT a, b, (CASE WHEN $6 != '' THEN $6::subscription_status ELSE 'unconfirmed' END) FROM UNNEST(ARRAY(SELECT id FROM subs)) a, UNNEST($5::INT[]) b)
    ON CONFLICT (subscriber_id, list_id) DO NOTHING;

-- name: delete-subscriptions-by-query
-- raw: true
WITH subs AS (%query%)
DELETE FROM subscriber_lists
    WHERE (subscriber_id, list_id) = ANY(SELECT a, b FROM UNNEST(ARRAY(SELECT id FROM subs)) a, UNNEST($5::INT[]) b);

-- name: unsubscribe-subscribers-from-lists-by-query
-- raw: true
WITH subs AS (%query%)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE (subscriber_id, list_id) = ANY(SELECT a, b FROM UNNEST(ARRAY(SELECT id FROM subs)) a, UNNEST($5::INT[]) b);


-- lists
-- name: get-lists
SELECT * FROM lists WHERE (CASE WHEN $1 = '' THEN 1=1 ELSE type=$1::list_type END)
    AND CASE
        -- Optional list IDs based on user permission.
        WHEN $3 = TRUE THEN TRUE ELSE id = ANY($4::INT[])
    END
    ORDER BY CASE WHEN $2 = 'id' THEN id END, CASE WHEN $2 = 'name' THEN name END;

-- name: query-lists
WITH ls AS (
    SELECT COUNT(*) OVER () AS total, lists.* FROM lists WHERE
    CASE
        WHEN $1 > 0 THEN id = $1
        WHEN $2 != '' THEN uuid = $2::UUID
        WHEN $3 != '' THEN to_tsvector(name) @@ to_tsquery ($3)
        ELSE TRUE
    END
    AND ($4 = '' OR type = $4::list_type)
    AND ($5 = '' OR optin = $5::list_optin)
    AND (CARDINALITY($6::VARCHAR(100)[]) = 0 OR $6 <@ tags)
    AND CASE
        -- Optional list IDs based on user permission.
        WHEN $7 = TRUE THEN TRUE ELSE id = ANY($8::INT[])
    END
    OFFSET $9 LIMIT (CASE WHEN $10 < 1 THEN NULL ELSE $10 END)
),
statuses AS (
    SELECT
        list_id,
        COALESCE(JSONB_OBJECT_AGG(status, subscriber_count) FILTER (WHERE status IS NOT NULL), '{}') AS subscriber_statuses,
        SUM(subscriber_count) AS subscriber_count
    FROM mat_list_subscriber_stats
    GROUP BY list_id
)
SELECT ls.*, COALESCE(ss.subscriber_statuses, '{}') AS subscriber_statuses, COALESCE(ss.subscriber_count, 0) AS subscriber_count
    FROM ls LEFT JOIN statuses ss ON (ls.id = ss.list_id) ORDER BY %order%;

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
WITH tpl AS (
    -- Select the template for the given template ID or use the default template.
    SELECT
        -- If the template is a visual template, then use it's HTML body as the campaign
        -- body and its block source as the campaign's block source,
        -- and don't set a template_id in the campaigns table, as it's essentially an
        -- HTML template body "import" during creation.
        (CASE WHEN type = 'campaign_visual' THEN NULL ELSE id END) AS id,
        (CASE WHEN type = 'campaign_visual' THEN body ELSE '' END) AS body,
        (CASE WHEN type = 'campaign_visual' THEN body_source ELSE NULL END) AS body_source,
        (CASE WHEN type = 'campaign_visual' THEN 'visual' ELSE 'richtext' END) AS content_type
    FROM templates
    WHERE
        CASE
            -- If a template ID is present, use it. If not, use the default template only if
            -- it's not a visual template.
            WHEN $13::INT IS NOT NULL THEN id = $13::INT
            ELSE $8 != 'visual' AND is_default = TRUE
        END
    LIMIT 1
),
counts AS (
    -- This is going to be slow on large databases.
    SELECT
        COALESCE(COUNT(DISTINCT sl.subscriber_id), 0) AS to_send, COALESCE(MAX(s.id), 0) AS max_sub_id
    FROM subscriber_lists sl
        JOIN lists l ON sl.list_id = l.id
        JOIN subscribers s ON sl.subscriber_id = s.id
    WHERE sl.list_id = ANY($14::INT[])
      AND s.status != 'blocklisted'
      AND (
        (l.optin = 'double' AND sl.status = 'confirmed') OR
        (l.optin != 'double' AND sl.status != 'unsubscribed')
      )
),
camp AS (
    INSERT INTO campaigns (uuid, type, name, subject, from_email, body, altbody,
        content_type, send_at, headers, tags, messenger, template_id, to_send,
        max_subscriber_id, archive, archive_slug, archive_template_id, archive_meta, body_source)
        SELECT $1, $2, $3, $4, $5,
            -- body
            COALESCE(NULLIF($6, ''), (SELECT body FROM tpl), ''),
            $7,
            $8::content_type,
            $9, $10, $11, $12,
            (SELECT id FROM tpl),
            (SELECT to_send FROM counts),
            (SELECT max_sub_id FROM counts),
            $15, $16,
            -- archive_template_id
            $17,
            $18,
            -- body_source
            COALESCE($20, (SELECT body_source FROM tpl))
        RETURNING id
),
med AS (
    INSERT INTO campaign_media (campaign_id, media_id, filename)
        (SELECT (SELECT id FROM camp), id, filename FROM media WHERE id=ANY($19::INT[]))
),
insLists AS (
    INSERT INTO campaign_lists (campaign_id, list_id, list_name)
        SELECT (SELECT id FROM camp), id, name FROM lists WHERE id=ANY($14::INT[])
)
SELECT id FROM camp;

-- name: query-campaigns
-- Here, 'lists' is returned as an aggregated JSON array from campaign_lists because
-- the list reference may have been deleted.
-- While the results are sliced using offset+limit,
-- there's a COUNT() OVER() that still returns the total result count
-- for pagination in the frontend, albeit being a field that'll repeat
-- with every resultant row.
SELECT  c.*,
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
    AND (CARDINALITY($2::campaign_status[]) = 0 OR status = ANY($2))
    AND (CARDINALITY($3::VARCHAR(100)[]) = 0 OR $3 <@ tags)
    AND ($4 = '' OR TO_TSVECTOR(CONCAT(name, ' ', subject)) @@ TO_TSQUERY($4) OR CONCAT(c.name, ' ', c.subject) ILIKE $4)
    -- Get all campaigns or filter by list IDs.
    AND (
        $5 OR EXISTS (
            SELECT 1 FROM campaign_lists WHERE campaign_id = c.id AND list_id = ANY($6::INT[])
        )
    )
ORDER BY %order% OFFSET $7 LIMIT (CASE WHEN $8 < 1 THEN NULL ELSE $8 END);

-- name: get-campaign
SELECT campaigns.*,
    COALESCE(templates.body, (SELECT body FROM templates WHERE is_default = true LIMIT 1), '') AS template_body
    FROM campaigns
    LEFT JOIN templates ON (
        CASE WHEN $4 = 'default' THEN templates.id = campaigns.template_id
        ELSE templates.id = campaigns.archive_template_id END
    )
    WHERE CASE
            WHEN $1 > 0 THEN campaigns.id = $1
            WHEN $3 != '' THEN campaigns.archive_slug = $3
            ELSE uuid = $2
          END;

-- name: get-archived-campaigns
SELECT COUNT(*) OVER () AS total, campaigns.*,
    COALESCE(templates.body, (SELECT body FROM templates WHERE is_default = true LIMIT 1), '') AS template_body
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
SELECT campaigns.*, COALESCE(templates.body, '') AS template_body,
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
SELECT id, status, to_send, sent, started_at, updated_at FROM campaigns WHERE status=$1;

-- name: campaign-has-lists
-- Returns TRUE if the campaign $1 has any of the lists given in $2.
SELECT EXISTS (
    SELECT TRUE FROM campaign_lists WHERE campaign_id = $1 AND list_id = ANY($2::INT[])
);

-- name: next-campaigns
-- Retreives campaigns that are running (or scheduled and the time's up) and need
-- to be processed. It updates the to_send count and max_subscriber_id of the campaign,
-- that is, the total number of subscribers to be processed across all lists of a campaign.
-- Thus, it has a sideaffect.
-- In addition, it finds the max_subscriber_id, the upper limit across all lists of
-- a campaign. This is used to fetch and slice subscribers for the campaign in next-campaign-subscribers.
WITH camps AS (
    -- Get all running campaigns and their template bodies (if the template's deleted, the default template body instead)
    SELECT campaigns.*, COALESCE(templates.body, (SELECT body FROM templates WHERE is_default = true LIMIT 1), '') AS template_body
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
    SELECT camps.id AS campaign_id, COUNT(DISTINCT sl.subscriber_id) AS to_send, COALESCE(MAX(sl.subscriber_id), 0) AS max_subscriber_id
    FROM camps
    JOIN campLists cl ON cl.campaign_id = camps.id
    JOIN subscriber_lists sl ON sl.list_id = cl.list_id
        AND (
            CASE
                WHEN camps.type = 'optin' THEN sl.status = 'unconfirmed' AND cl.optin = 'double'
                WHEN cl.optin = 'double' THEN sl.status = 'confirmed'
                ELSE sl.status != 'unsubscribed'
            END
        )
    JOIN subscribers s ON (s.id = sl.subscriber_id AND s.status != 'blocklisted')
    GROUP BY camps.id
),
updateCounts AS (
    WITH uc (campaign_id, sent_count) AS (SELECT * FROM unnest($1::INT[], $2::INT[]))
    UPDATE campaigns
    SET sent = sent + uc.sent_count
    FROM uc WHERE campaigns.id = uc.campaign_id
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
    FROM uniqIDs GROUP BY campaign_id, "timestamp" ORDER BY "timestamp" ASC;

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

-- name: get-running-campaign
-- Returns the metadata for a running campaign that is required by next-campaign-subscribers to retrieve
-- a batch of campaign subscribers for processing.
SELECT campaigns.id AS campaign_id, campaigns.type as campaign_type, last_subscriber_id, max_subscriber_id, lists.id AS list_id
    FROM campaigns
    LEFT JOIN campaign_lists ON (campaign_lists.campaign_id = campaigns.id)
    LEFT JOIN lists ON (lists.id = campaign_lists.list_id)
    WHERE campaigns.id = $1 AND status='running';

-- name: next-campaign-subscribers
-- Returns a batch of subscribers in a given campaign starting from the last checkpoint
-- (last_subscriber_id). Every fetch updates the checkpoint and the sent count, which means
-- every fetch returns a new batch of subscribers until all rows are exhausted.
--
-- In previous versions, get-running-campaign + this was a single query spread across multiple
-- CTEs, but despite numerous permutations and combinations, Postgres query planner simply would not use
-- the right indexes on subscriber_lists when the JOIN or ids were referenced dynamically from campLists
-- (be it a CTE or various kinds of joins). However, statically providing the list IDs to JOIN on ($5::INT[])
-- the query planner works as expected. The difference is staggering. ~15 seconds on a subscribers table with 15m
-- rows and a subscriber_lists table with 70 million rows when fetching subscribers for a campaign with a single list,
-- vs. a few million seconds using this current approach.
WITH campLists AS (
    SELECT lists.id AS list_id, optin FROM lists
    LEFT JOIN campaign_lists ON campaign_lists.list_id = lists.id
    WHERE campaign_lists.campaign_id = $1
),
subs AS (
    SELECT s.*
    FROM (
        SELECT DISTINCT s.id
        FROM subscriber_lists sl
        JOIN campLists ON sl.list_id = campLists.list_id
        JOIN subscribers s ON s.id = sl.subscriber_id
        WHERE
            sl.list_id = ANY($5::INT[])
            -- last_subscriber_id
            AND s.id > $3
             -- max_subscriber_id
            AND s.id <= $4
             -- Subscriber should not be blacklisted.
            AND s.status != 'blocklisted'
            AND (
                -- If it's an optin campaign and the list is double-optin, only pick unconfirmed subscribers.
                ($2 = 'optin' AND sl.status = 'unconfirmed' AND campLists.optin = 'double')
                OR (
                    -- It is a regular campaign.
                    $2 != 'optin' AND (
                        -- It is a double optin list. Only pick confirmed subscribers.
                        (campLists.optin = 'double' AND sl.status = 'confirmed') OR

                        -- It is a single optin list. Pick all non-unsubscribed subscribers.
                        (campLists.optin != 'double' AND sl.status != 'unsubscribed')
                    )
                )
            )
        ORDER BY s.id LIMIT $6
    ) subIDs JOIN subscribers s ON (s.id = subIDs.id) ORDER BY s.id
),
u AS (
    UPDATE campaigns
    SET last_subscriber_id = (SELECT MAX(id) FROM subs), updated_at = NOW()
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
        status=(
            CASE
                WHEN status = 'scheduled' AND $8 IS NULL THEN 'draft'
                ELSE status
            END
        ),
        headers=$9,
        tags=$10::VARCHAR(100)[],
        messenger=$11,
        -- template_id shouldn't be saved for visual campaigns.
        template_id=(CASE WHEN $7::content_type = 'visual' THEN NULL ELSE $12::INT END),
        archive=$14,
        archive_slug=$15,
        archive_template_id=(CASE WHEN $7::content_type = 'visual' THEN NULL ELSE $16::INT END),
        archive_meta=$17,
        body_source=$19,
        updated_at=NOW()
    WHERE id = $1 RETURNING id
),
clists AS (
    -- Reset list relationships
    DELETE FROM campaign_lists WHERE campaign_id = $1 AND NOT(list_id = ANY($13))
),
med AS (
    DELETE FROM campaign_media WHERE campaign_id = $1
    AND ( media_id IS NULL or NOT(media_id = ANY($18))) RETURNING media_id
),
medi AS (
    INSERT INTO campaign_media (campaign_id, media_id, filename)
        (SELECT $1 AS campaign_id, id, filename FROM media WHERE id=ANY($18::INT[]))
        ON CONFLICT (campaign_id, media_id) DO NOTHING
)
INSERT INTO campaign_lists (campaign_id, list_id, list_name)
    (SELECT $1 as campaign_id, id, name FROM lists WHERE id=ANY($13::INT[]))
    ON CONFLICT (campaign_id, list_id) DO UPDATE SET list_name = EXCLUDED.list_name;

-- name: update-campaign-counts
UPDATE campaigns SET
    to_send=(CASE WHEN $2 != 0 THEN $2 ELSE to_send END),
    sent=sent+$3,
    last_subscriber_id=(CASE WHEN $4 > 0 THEN $4 ELSE to_send END),
    updated_at=NOW()
WHERE id=$1;

-- name: update-campaign-status
UPDATE campaigns SET
    status=(
        CASE
            WHEN send_at IS NOT NULL AND $2 = 'running' THEN 'scheduled'
            ELSE $2::campaign_status
        END
    ),
    updated_at=NOW()
WHERE id = $1;

-- name: update-campaign-archive
UPDATE campaigns SET
    archive=$2,
    archive_slug=(CASE WHEN $3::TEXT = '' THEN NULL ELSE $3 END),
    archive_template_id=(CASE WHEN $4 > 0 THEN $4 ELSE archive_template_id END),
    archive_meta=(CASE WHEN $5::TEXT != '' THEN $5::JSONB ELSE archive_meta END),
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

-- templates
-- name: get-templates
-- Only if the second param ($2 - noBody) is true, body and body_source is returned.
SELECT id, name, type, subject,
    (CASE WHEN $2 = false THEN body ELSE '' END) as body,
    (CASE WHEN $2 = false THEN body_source ELSE NULL END) as body_source,
    is_default, created_at, updated_at
    FROM templates WHERE ($1 = 0 OR id = $1) AND ($3 = '' OR type = $3::template_type)
    ORDER BY created_at;

-- name: create-template
INSERT INTO templates (name, type, subject, body, body_source) VALUES($1, $2, $3, $4, $5) RETURNING id;

-- name: update-template
UPDATE templates SET
    name=(CASE WHEN $2 != '' THEN $2 ELSE name END),
    subject=(CASE WHEN $3 != '' THEN $3 ELSE name END),
    body=(CASE WHEN $4 != '' THEN $4 ELSE body END),
    body_source=(CASE WHEN $5 != '' THEN $5 ELSE body_source END),
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
    SELECT id FROM templates WHERE is_default = true AND (type='campaign' OR type='campaign_visual') LIMIT 1
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
SELECT * FROM media WHERE
    CASE
        WHEN $1 > 0 THEN id = $1
        WHEN $2 != '' THEN uuid = $2::UUID
        WHEN $3 != '' THEN filename = $3    
        ELSE false
    END;

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
SELECT data FROM mat_dashboard_charts;

-- name: get-dashboard-counts
SELECT data FROM mat_dashboard_counts;

-- name: get-settings
SELECT JSON_OBJECT_AGG(key, value) AS settings FROM (SELECT * FROM settings ORDER BY key) t;

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
    subscribers.status as subscriber_status,
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
DELETE FROM bounces WHERE $2 = TRUE OR id = ANY($1);

-- name: delete-bounces-by-subscriber
WITH sub AS (
    SELECT id FROM subscribers WHERE CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END
)
DELETE FROM bounces WHERE subscriber_id = (SELECT id FROM sub);

-- name: blocklist-bounced-subscribers
WITH subs AS (
    SELECT subscriber_id FROM bounces
),
b AS (
    UPDATE subscribers SET status='blocklisted', updated_at=NOW()
    WHERE id = ANY(SELECT subscriber_id FROM subs)
)
UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()
    WHERE subscriber_id = ANY(SELECT subscriber_id FROM subs);

-- name: get-db-info
SELECT JSON_BUILD_OBJECT('version', (SELECT VERSION()),
                        'size_mb', (SELECT ROUND(pg_database_size((SELECT CURRENT_DATABASE()))/(1024^2)))) AS info;

-- name: create-user
INSERT INTO users (username, password_login, password, email, name, type, user_role_id, list_role_id, status)
    VALUES($1, $2, (
        CASE
            -- For user types with password_login enabled, bcrypt and store the hash of the password.
            WHEN $6::user_type != 'api' AND $2 AND $3 != ''
                THEN CRYPT($3, GEN_SALT('bf'))
            WHEN $6 = 'api'
            -- For APIs, store the password (token) as-is.
                THEN $3
            ELSE NULL
        END
    ), $4, $5, $6, (SELECT id FROM roles WHERE id = $7 AND type = 'user'), (SELECT id FROM roles WHERE id = $8 AND type = 'list'), $9) RETURNING id;

-- name: update-user
WITH u AS (
    -- Edit is only allowed if there are more than 1 active super users or
    -- if the only superadmin user's status/role isn't being changed.
    SELECT
        CASE
            WHEN (SELECT COUNT(*) FROM users WHERE id != $1 AND status = 'enabled' AND type = 'user' AND user_role_id = 1) = 0  AND ($8 != 1 OR $10 != 'enabled')
            THEN FALSE
            ELSE TRUE
        END AS canEdit
)
UPDATE users SET
    username=(CASE WHEN $2 != '' THEN $2 ELSE username END),
    password_login=$3,
    password=(CASE WHEN $3 = TRUE THEN (CASE WHEN $4 != '' THEN CRYPT($4, GEN_SALT('bf')) ELSE password END) ELSE NULL END),
    email=(CASE WHEN $5 != '' THEN $5 ELSE email END),
    name=(CASE WHEN $6 != '' THEN $6 ELSE name END),
    type=(CASE WHEN $7 != '' THEN $7::user_type ELSE type END),
    user_role_id=(CASE WHEN $8 != 0 THEN (SELECT id FROM roles WHERE id = $8 AND type = 'user') ELSE user_role_id END),
    list_role_id=(
        CASE
            WHEN $9 < 0 THEN NULL
            WHEN $9 > 0 THEN (SELECT id FROM roles WHERE id = $9 AND type = 'list')
            ELSE list_role_id END
    ),
    status=(CASE WHEN $10 != '' THEN $10::user_status ELSE status END),
    updated_at=NOW()
    WHERE id=$1 AND (SELECT canEdit FROM u) = TRUE;

-- name: delete-users
WITH u AS (
    SELECT COUNT(*) AS num FROM users WHERE NOT(id = ANY($1)) AND user_role_id=1 AND type='user' AND status='enabled'
)
DELETE FROM users WHERE id = ALL($1) AND (SELECT num FROM u) > 0;

-- name: get-users
WITH ur AS (
    SELECT id, name, permissions FROM roles WHERE type = 'user' AND parent_id IS NULL
),
lr AS (
    SELECT r.id, r.name, r.permissions, r.list_id, l.name AS list_name
    FROM roles r
    LEFT JOIN lists l ON r.list_id = l.id
    WHERE r.type = 'list' AND r.parent_id IS NULL
),
lp AS (
    SELECT lr.id AS list_role_id,
        JSONB_AGG(
            JSONB_BUILD_OBJECT(
                'id', COALESCE(cr.list_id, lr.list_id),
                'name', COALESCE(cl.name, lr.list_name),
                'permissions', COALESCE(cr.permissions, lr.permissions)
            )
        ) AS list_role_perms
    FROM lr
    LEFT JOIN roles cr ON cr.parent_id = lr.id AND cr.type = 'list'
    LEFT JOIN lists cl ON cr.list_id = cl.id
    GROUP BY lr.id
)
SELECT
    users.*,
    ur.id AS user_role_id,
    ur.name AS user_role_name,
    ur.permissions AS user_role_permissions,
    lp.list_role_id,
    lr.name AS list_role_name,
    lp.list_role_perms
FROM users
    LEFT JOIN ur ON users.user_role_id = ur.id
    LEFT JOIN lp ON users.list_role_id = lp.list_role_id
    LEFT JOIN lr ON lp.list_role_id = lr.id
    ORDER BY users.created_at;

-- name: get-user
WITH sel AS (
    SELECT * FROM users
    WHERE
    (
        CASE
            WHEN $1::INT != 0 THEN users.id = $1
            WHEN $2::TEXT != '' THEN username = $2
            WHEN $3::TEXT != '' THEN email = $3
        END
    )
)
SELECT
    sel.*,
    ur.id AS user_role_id,
    ur.name AS user_role_name,
    ur.permissions AS user_role_permissions,
    lr.id AS list_role_id,
    lr.name AS list_role_name,
    lp.list_role_perms
FROM sel
    LEFT JOIN roles ur ON sel.user_role_id = ur.id AND ur.type = 'user' AND ur.parent_id IS NULL
    LEFT JOIN (
        SELECT r.id, r.name, r.permissions, r.list_id, l.name AS list_name
        FROM roles r
        LEFT JOIN lists l ON r.list_id = l.id
        WHERE r.type = 'list' AND r.parent_id IS NULL
    ) lr ON sel.list_role_id = lr.id
    LEFT JOIN LATERAL (
        SELECT JSONB_AGG(
                JSONB_BUILD_OBJECT(
                    'id', COALESCE(cr.list_id, lr.list_id),
                    'name', COALESCE(cl.name, lr.list_name),
                    'permissions', COALESCE(cr.permissions, lr.permissions)
                )
            ) AS list_role_perms
        FROM roles cr
        LEFT JOIN lists cl ON cr.list_id = cl.id
        WHERE cr.parent_id = lr.id AND cr.type = 'list'
        GROUP BY lr.id
    ) lp ON TRUE;


-- name: get-api-tokens
SELECT username, password FROM users WHERE status='enabled' AND type='api';

-- name: login-user
WITH u AS (
    SELECT users.*, r.name as role_name, r.permissions FROM users
    LEFT JOIN roles r ON (r.id = users.user_role_id)
    WHERE username = $1 AND status != 'disabled' AND password_login = TRUE
    AND CRYPT($2, password) = password
)
UPDATE users SET loggedin_at = NOW() WHERE id = (SELECT id FROM u) RETURNING *;

-- name: update-user-profile
UPDATE users SET name=$2, email=(CASE WHEN password_login THEN $3 ELSE email END),
    password=(CASE WHEN $4 = TRUE THEN (CASE WHEN $5 != '' THEN CRYPT($5, GEN_SALT('bf')) ELSE password END) ELSE NULL END)
    WHERE id=$1;

-- name: update-user-login
UPDATE users SET loggedin_at=NOW(), avatar=(CASE WHEN $2 != '' THEN $2 ELSE avatar END) WHERE id=$1;

-- name: get-user-roles
WITH mainroles AS (
    SELECT ur.* FROM roles ur WHERE type = 'user' AND ur.parent_id IS NULL AND
    CASE WHEN $1::INT != 0 THEN ur.id = $1 ELSE TRUE END
),
listPerms AS (
    SELECT ur.parent_id, JSONB_AGG(JSONB_BUILD_OBJECT('id', ur.list_id, 'name', lists.name, 'permissions', ur.permissions)) AS listPerms
    FROM roles ur
    LEFT JOIN lists ON(lists.id = ur.list_id)
    WHERE ur.parent_id IS NOT NULL GROUP BY ur.parent_id
)
SELECT p.*, COALESCE(l.listPerms, '[]'::JSONB) AS "list_permissions" FROM mainroles p
    LEFT JOIN listPerms l ON p.id = l.parent_id ORDER BY p.created_at;

-- name: get-list-roles
WITH mainroles AS (
    SELECT ur.* FROM roles ur WHERE type = 'list' AND ur.parent_id IS NULL
),
listPerms AS (
    SELECT ur.parent_id, JSONB_AGG(JSONB_BUILD_OBJECT('id', ur.list_id, 'name', lists.name, 'permissions', ur.permissions)) AS listPerms
    FROM roles ur
    LEFT JOIN lists ON(lists.id = ur.list_id)
    WHERE ur.parent_id IS NOT NULL GROUP BY ur.parent_id
)
SELECT p.*, COALESCE(l.listPerms, '[]'::JSONB) AS "list_permissions" FROM mainroles p
    LEFT JOIN listPerms l ON p.id = l.parent_id ORDER BY p.created_at;


-- name: create-role
INSERT INTO roles (name, type, permissions, created_at, updated_at) VALUES($1, $2, $3, NOW(), NOW()) RETURNING *;

-- name: upsert-list-permissions
WITH d AS (
    -- Delete lists that aren't included.
    DELETE FROM roles WHERE parent_id = $1 AND list_id != ALL($2::INT[])
),
p AS (
    -- Get (list_id, perms[]), (list_id, perms[])
    SELECT UNNEST($2) AS list_id, JSONB_ARRAY_ELEMENTS(TO_JSONB($3::TEXT[][])) AS perms
)
INSERT INTO roles (parent_id, list_id, permissions, type)
    SELECT $1, list_id, ARRAY_REMOVE(ARRAY(SELECT JSONB_ARRAY_ELEMENTS_TEXT(perms)), ''), 'list' FROM p
    ON CONFLICT (parent_id, list_id) DO UPDATE SET permissions = EXCLUDED.permissions;

-- name: delete-list-permission
DELETE FROM roles WHERE parent_id=$1 AND list_id=$2;

-- name: update-role
UPDATE roles SET name=$2, permissions=$3 WHERE id=$1 and parent_id IS NULL RETURNING *;

-- name: delete-role
DELETE FROM roles WHERE id=$1;
