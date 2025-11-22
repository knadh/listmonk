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

