-- Pelvic floor program (CB-PELV-01, docs/canvas-build/README.md → PELV). Health data: every query is scoped by
-- user_id in the statement itself.

-- name: GetProgram :one
SELECT * FROM `pelvic_programs` WHERE user_id = ? LIMIT 1;

-- name: UpsertProgram :exec
-- One program per user; starting again moves started_on (sessions and diary are kept).
INSERT INTO `pelvic_programs` (user_id, started_on, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(started_on), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  started_on = VALUES(started_on),
  updated_at = VALUES(updated_at);

-- name: DeleteProgram :exec
DELETE FROM `pelvic_programs` WHERE user_id = ?;

-- name: AddSession :exec
-- One row per (user, day): each saved session adds to the day's totals (capped at the column sizes).
INSERT INTO `pelvic_sessions` (user_id, session_date, sessions_count, sets_completed, duration_sec, level_code, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(session_date), 1, sqlc.arg(sets_completed), sqlc.arg(duration_sec), sqlc.arg(level_code), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  sessions_count = LEAST(sessions_count + 1, 65535),
  sets_completed = LEAST(sets_completed + VALUES(sets_completed), 65535),
  duration_sec = LEAST(duration_sec + VALUES(duration_sec), 86400),
  level_code = VALUES(level_code),
  updated_at = VALUES(updated_at);

-- name: ListSessionDays :many
-- The user's trained days from `from` to `to` (inclusive), newest first (streak and week dots).
SELECT session_date
FROM `pelvic_sessions`
WHERE user_id = sqlc.arg(user_id)
  AND session_date >= sqlc.arg(from_date)
  AND session_date <= sqlc.arg(to_date)
  AND sessions_count > 0
ORDER BY session_date DESC;

-- name: GetBladderLog :one
SELECT * FROM `pelvic_bladder_logs` WHERE user_id = ? AND log_date = ? LIMIT 1;

-- name: UpsertBladderLog :exec
-- One row per (user, day); created_at is kept on update.
INSERT INTO `pelvic_bladder_logs` (user_id, log_date, leak, night_voids, uti_symptoms, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(log_date), sqlc.arg(leak), sqlc.arg(night_voids), sqlc.arg(uti_symptoms), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  leak = VALUES(leak),
  night_voids = VALUES(night_voids),
  uti_symptoms = VALUES(uti_symptoms),
  updated_at = VALUES(updated_at);

-- name: DeleteBladderLog :exec
DELETE FROM `pelvic_bladder_logs` WHERE user_id = ? AND log_date = ?;
