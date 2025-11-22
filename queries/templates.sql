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

