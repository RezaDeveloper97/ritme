-- Notification settings (B-N1-11, goose 00011): one row per user, Go only. Read by
-- GET /profile/notification-settings and by every push sender through internal/notifications.

-- name: GetNotificationPreferences :one
SELECT * FROM `notification_preferences` WHERE user_id = ? LIMIT 1;

-- name: UpsertNotificationPreferences :exec
INSERT INTO `notification_preferences`
  (user_id, categories, quiet_hours_enabled, quiet_start, quiet_end, neutral_copy, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(categories), sqlc.arg(quiet_hours_enabled), sqlc.arg(quiet_start),
   sqlc.arg(quiet_end), sqlc.arg(neutral_copy), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  categories = VALUES(categories),
  quiet_hours_enabled = VALUES(quiet_hours_enabled),
  quiet_start = VALUES(quiet_start),
  quiet_end = VALUES(quiet_end),
  neutral_copy = VALUES(neutral_copy),
  updated_at = VALUES(updated_at);
