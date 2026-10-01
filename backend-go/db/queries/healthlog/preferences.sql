-- Log preferences and custom items (B-N3-02, health_log_preferences / health_log_custom_items, migration
-- 00021). Every query is scoped by user_id.

-- name: GetLogPreferences :one
SELECT * FROM `health_log_preferences` WHERE user_id = ? AND mode = ? LIMIT 1;

-- name: UpsertLogPreferences :exec
INSERT INTO `health_log_preferences` (user_id, mode, category_order, hidden, pinned, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(mode), sqlc.narg(category_order), sqlc.narg(hidden), sqlc.narg(pinned),
        sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE category_order = VALUES(category_order), hidden = VALUES(hidden), pinned = VALUES(pinned),
                        updated_at = VALUES(updated_at);

-- name: DeleteLogPreferences :exec
DELETE FROM `health_log_preferences` WHERE user_id = ? AND mode = ?;

-- name: ListLogCustomItems :many
-- Active and deleted (history labels), oldest first.
SELECT * FROM `health_log_custom_items` WHERE user_id = ? ORDER BY id;

-- name: ListActiveLogCustomItems :many
SELECT * FROM `health_log_custom_items` WHERE user_id = ? AND deleted_at IS NULL ORDER BY id;

-- name: CountActiveLogCustomItems :one
SELECT COUNT(*) FROM `health_log_custom_items` WHERE user_id = ? AND deleted_at IS NULL;

-- name: GetLogCustomItem :one
SELECT * FROM `health_log_custom_items` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: InsertLogCustomItem :execlastid
INSERT INTO `health_log_custom_items` (user_id, category, param, label, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: RenameLogCustomItem :exec
UPDATE `health_log_custom_items` SET label = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL;

-- name: SoftDeleteLogCustomItem :exec
UPDATE `health_log_custom_items` SET deleted_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;
