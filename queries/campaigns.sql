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
            WHEN $14::INT IS NOT NULL THEN id = $14::INT
            ELSE $8 != 'visual' AND is_default = TRUE
        END
    LIMIT 1
),
camp AS (
    INSERT INTO campaigns (uuid, type, name, subject, from_email, body, altbody,
        content_type, send_at, headers, attribs, tags, messenger, template_id, to_send,
        max_subscriber_id, archive, archive_slug, archive_template_id, archive_meta, body_source)
        SELECT $1, $2, $3, $4, $5,
            -- body
            COALESCE(NULLIF($6, ''), (SELECT body FROM tpl), ''),
            $7,
            $8::content_type,
            $9, $10, $11, $12, $13,
            (SELECT id FROM tpl),
            0,
            0,
            $16, $17,
            -- archive_template_id
            $18,
            $19,
            -- body_source
            COALESCE($21, (SELECT body_source FROM tpl))
        RETURNING id
),
med AS (
    INSERT INTO campaign_media (campaign_id, media_id, filename)
        (SELECT (SELECT id FROM camp), id, filename FROM media WHERE id=ANY($20::INT[]))
),
insLists AS (
    INSERT INTO campaign_lists (campaign_id, list_id, list_name)
        SELECT (SELECT id FROM camp), id, name FROM lists WHERE id=ANY($15::INT[])
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
-- Mailchimp-style: count unique subscribers (one open per person). Anonymous views (subscriber_id NULL) count as 1 each.
views AS (
    SELECT campaign_id,
        COUNT(DISTINCT subscriber_id) + COUNT(*) FILTER (WHERE subscriber_id IS NULL) AS num
    FROM campaign_views
    WHERE campaign_id = ANY($1)
    GROUP BY campaign_id
),
-- Unique recipients who clicked at least one link; anonymous clicks count as 1 each.
clicks AS (
    SELECT campaign_id,
        COUNT(DISTINCT subscriber_id) + COUNT(*) FILTER (WHERE subscriber_id IS NULL) AS num
    FROM link_clicks
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
    WHERE campaigns.id = $1 AND campaigns.status='running';

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
        attribs=$10,
        tags=$11::VARCHAR(100)[],
        messenger=$12,
        -- template_id shouldn't be saved for visual campaigns.
        template_id=(CASE WHEN $7::content_type = 'visual' THEN NULL ELSE $13::INT END),
        archive=$15,
        archive_slug=$16,
        archive_template_id=(CASE WHEN $7::content_type = 'visual' THEN NULL ELSE $17::INT END),
        archive_meta=$18,
        body_source=$20,
        updated_at=NOW()
    WHERE id = $1 RETURNING id
),
clists AS (
    -- Reset list relationships
    DELETE FROM campaign_lists WHERE campaign_id = $1 AND NOT(list_id = ANY($14))
),
med AS (
    DELETE FROM campaign_media WHERE campaign_id = $1
    AND ( media_id IS NULL or NOT(media_id = ANY($19))) RETURNING media_id
),
medi AS (
    INSERT INTO campaign_media (campaign_id, media_id, filename)
        (SELECT $1 AS campaign_id, id, filename FROM media WHERE id=ANY($19::INT[]))
        ON CONFLICT (campaign_id, media_id) DO NOTHING
)
INSERT INTO campaign_lists (campaign_id, list_id, list_name)
    (SELECT $1 as campaign_id, id, name FROM lists WHERE id=ANY($14::INT[]))
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

-- name: delete-campaigns
DELETE FROM campaigns c
WHERE (
    CASE
        WHEN CARDINALITY($1::INT[]) > 0 THEN id = ANY($1)
        ELSE $2 = '' OR TO_TSVECTOR(CONCAT(name, ' ', subject)) @@ TO_TSQUERY($2) OR CONCAT(c.name, ' ', c.subject) ILIKE $2
    END
)
-- Get all campaigns or filter by permitted list IDs.
AND (
    $3 OR EXISTS (
        SELECT 1 FROM campaign_lists WHERE campaign_id = c.id AND list_id = ANY($4::INT[])
    )
);

-- name: register-campaign-view
WITH view AS (
    SELECT campaigns.id as campaign_id, subscribers.id AS subscriber_id FROM campaigns
    LEFT JOIN subscribers ON (CASE WHEN $2::TEXT != '' THEN subscribers.uuid = $2::UUID ELSE FALSE END)
    WHERE campaigns.uuid = $1
)
INSERT INTO campaign_views (campaign_id, subscriber_id)
    VALUES((SELECT campaign_id FROM view), (SELECT subscriber_id FROM view));

