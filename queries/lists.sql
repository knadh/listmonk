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

