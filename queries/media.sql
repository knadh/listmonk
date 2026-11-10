-- media
-- name: insert-media
INSERT INTO media (uuid, filename, thumb, content_type, provider, meta, tags, created_at) VALUES($1, $2, $3, $4, $5, $6, COALESCE($7::TEXT[], '{}'), NOW()) RETURNING id;

-- name: query-media
SELECT COUNT(*) OVER () AS total, * FROM media
    WHERE ($1 = '' OR filename ILIKE $1 OR EXISTS (SELECT 1 FROM UNNEST(tags) AS tag WHERE tag ILIKE $1))
        AND provider=$2 AND (CARDINALITY($5::TEXT[]) = 0 OR tags @> $5::TEXT[])
    ORDER BY created_at DESC OFFSET $3 LIMIT $4;

-- name: get-media-tags
SELECT tag, COUNT(DISTINCT id) AS count
    FROM media CROSS JOIN UNNEST(tags) AS tag
    WHERE provider=$1 GROUP BY tag ORDER BY tag;

-- name: get-media
SELECT * FROM media WHERE
    CASE
        WHEN $1 > 0 THEN id = $1
        WHEN $2 != '' THEN uuid = $2::UUID
        WHEN $3 != '' THEN filename = $3    
        ELSE false
    END;

-- name: update-media
UPDATE media SET tags=$2::TEXT[] WHERE id=$1;

-- name: delete-media
DELETE FROM media WHERE id=$1 RETURNING filename;

