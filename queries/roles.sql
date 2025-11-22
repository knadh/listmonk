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
