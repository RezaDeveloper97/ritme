-- Admin-editable Plus settings (key/value; code defaults when a row is missing).

-- name: GetSetting :one
SELECT `value` FROM plus_settings WHERE `key` = ?;

-- name: UpsertSetting :exec
INSERT INTO plus_settings (`key`, `value`, created_at, updated_at)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE `value` = VALUES(`value`), updated_at = VALUES(updated_at);

-- name: DeleteSetting :exec
-- Clears an override so the code / env default applies again.
DELETE FROM plus_settings WHERE `key` = ?;
