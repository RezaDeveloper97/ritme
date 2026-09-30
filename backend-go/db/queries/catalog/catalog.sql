-- Content catalog (CB-CORE-03, docs/canvas-build/catalog.md): /api/v1/catalog/{group} and the admin CRUD under
-- /api/admin/v1/catalog. Shared, admin-owned content — no user data lives in this table.

-- name: ListActiveCatalogItems :many
-- One group's active items in display order (the public read, cached per group).
SELECT id, code, sort_order, audiences, title, body, meta, needs_review
FROM `catalog_items`
WHERE `group` = ? AND is_active = 1
ORDER BY sort_order, id;

-- name: ListCatalogGroups :many
-- Every group with its item counts (admin group picker).
SELECT `group`,
       COUNT(*) AS items_count,
       CAST(SUM(is_active) AS SIGNED) AS active_count
FROM `catalog_items`
GROUP BY `group`
ORDER BY `group`;

-- name: CountAdminCatalogItems :one
SELECT COUNT(*) FROM `catalog_items`
WHERE `group` = sqlc.arg(catalog_group)
  AND (code LIKE CAST(sqlc.arg(pattern) AS CHAR)
       OR title LIKE CAST(sqlc.arg(pattern) AS CHAR) OR title LIKE CAST(sqlc.arg(json_pattern) AS CHAR))
  AND is_active >= sqlc.arg(active_min) AND is_active <= sqlc.arg(active_max);

-- name: ListAdminCatalogItems :many
SELECT * FROM `catalog_items`
WHERE `group` = sqlc.arg(catalog_group)
  AND (code LIKE CAST(sqlc.arg(pattern) AS CHAR)
       OR title LIKE CAST(sqlc.arg(pattern) AS CHAR) OR title LIKE CAST(sqlc.arg(json_pattern) AS CHAR))
  AND is_active >= sqlc.arg(active_min) AND is_active <= sqlc.arg(active_max)
ORDER BY sort_order, id
LIMIT ? OFFSET ?;

-- name: GetCatalogItem :one
-- An item by id, only inside the group named by the URL (an id of another group is a 404).
SELECT * FROM `catalog_items` WHERE id = sqlc.arg(id) AND `group` = sqlc.arg(catalog_group) LIMIT 1;

-- name: CatalogCodeExists :one
SELECT EXISTS(SELECT 1 FROM `catalog_items` WHERE `group` = sqlc.arg(catalog_group) AND code = sqlc.arg(code)) AS found;

-- name: NextCatalogSortOrder :one
SELECT CAST(COALESCE(MAX(sort_order), 0) + 1 AS SIGNED) AS next_sort_order
FROM `catalog_items` WHERE `group` = ?;

-- name: CreateCatalogItem :execresult
INSERT INTO `catalog_items` (`group`, code, sort_order, is_active, audiences, title, body, meta, needs_review,
                             created_at, updated_at)
VALUES (sqlc.arg(catalog_group), sqlc.arg(code), sqlc.arg(sort_order), sqlc.arg(is_active), sqlc.narg(audiences),
        sqlc.arg(title), sqlc.narg(body), sqlc.narg(meta), sqlc.arg(needs_review), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateCatalogItem :exec
UPDATE `catalog_items`
SET sort_order = sqlc.arg(sort_order), is_active = sqlc.arg(is_active), audiences = sqlc.narg(audiences),
    title = sqlc.arg(title), body = sqlc.narg(body), meta = sqlc.narg(meta), needs_review = sqlc.arg(needs_review),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND `group` = sqlc.arg(catalog_group);

-- name: DeleteCatalogItem :execresult
DELETE FROM `catalog_items` WHERE id = sqlc.arg(id) AND `group` = sqlc.arg(catalog_group);
