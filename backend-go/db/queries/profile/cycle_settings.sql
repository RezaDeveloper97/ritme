-- Cycle settings (B-N1-09, goose 00013): GET/PUT /profile/cycle-settings, Go only. The reminder switches and
-- times live on the B-N1-11 `notification_preferences` row (categories + schedule); the «خودکار از داده‌ها»
-- flag in `cycle_preferences`. Every statement is scoped by user_id.

-- name: GetCyclePreferences :one
SELECT * FROM `cycle_preferences` WHERE user_id = ? LIMIT 1;

-- name: UpsertCyclePreferences :exec
INSERT INTO `cycle_preferences` (user_id, lengths_auto, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(lengths_auto), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  lengths_auto = VALUES(lengths_auto),
  updated_at = VALUES(updated_at);

-- name: UpsertReminderPreferences :exec
-- Writes the category switches and the schedule; on first save the other columns take the loaded values
-- (the defaults), and an existing row keeps its quiet hours / neutral copy.
INSERT INTO `notification_preferences`
  (user_id, categories, schedule, quiet_hours_enabled, quiet_start, quiet_end, neutral_copy, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(categories), sqlc.arg(schedule), sqlc.arg(quiet_hours_enabled), sqlc.arg(quiet_start),
   sqlc.arg(quiet_end), sqlc.arg(neutral_copy), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  categories = VALUES(categories),
  schedule = VALUES(schedule),
  updated_at = VALUES(updated_at);
