-- Admin kill-switches for whole app modules (app_modules; a missing row = enabled).

-- name: ListModules :many
SELECT `key`, enabled FROM app_modules ORDER BY id;

-- name: UpsertModule :exec
INSERT INTO app_modules (`key`, enabled, created_at, updated_at)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE enabled = VALUES(enabled), updated_at = VALUES(updated_at);
