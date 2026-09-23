-- Pregnancy v2 per-day extras (table pregnancy_daily_extras, T-M7-01): the day-log fields
-- pregnancy_symptom_logs lacks (mood, water, heartburn, constipation, visit note). Every query is
-- scoped by user_id. Unique (user_id, log_date).

-- name: GetDailyExtras :one
SELECT * FROM `pregnancy_daily_extras` WHERE user_id = ? AND log_date = ? LIMIT 1;

-- name: ListDailyExtrasRange :many
-- from..to inclusive (doctor report, alert windows).
SELECT * FROM `pregnancy_daily_extras`
WHERE user_id = sqlc.arg(user_id) AND log_date >= sqlc.arg(date_from) AND log_date <= sqlc.arg(date_to)
ORDER BY log_date;

-- name: UpsertDailyExtras :exec
-- One row per (user_id, log_date); created_at is kept on update. Idempotent (offline outbox resends).
INSERT INTO `pregnancy_daily_extras` (
  user_id, log_date, mood, water_glasses, heartburn_severity, constipation_severity, visit_note, created_at, updated_at
) VALUES (
  sqlc.arg(user_id), sqlc.arg(log_date), sqlc.arg(mood), sqlc.arg(water_glasses), sqlc.arg(heartburn_severity),
  sqlc.arg(constipation_severity), sqlc.arg(visit_note), sqlc.arg(now), sqlc.arg(now)
)
ON DUPLICATE KEY UPDATE
  mood = VALUES(mood),
  water_glasses = VALUES(water_glasses),
  heartburn_severity = VALUES(heartburn_severity),
  constipation_severity = VALUES(constipation_severity),
  visit_note = VALUES(visit_note),
  updated_at = VALUES(updated_at);

-- name: SetDailyVisitNote :exec
-- Alert action add_to_visit_note (T-M7-04): creates the day's row when missing, keeps the other fields.
INSERT INTO `pregnancy_daily_extras` (user_id, log_date, visit_note, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(log_date), sqlc.arg(visit_note), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE visit_note = VALUES(visit_note), updated_at = VALUES(updated_at);

-- name: DeleteDailyExtras :exec
DELETE FROM `pregnancy_daily_extras` WHERE user_id = ? AND log_date = ?;
