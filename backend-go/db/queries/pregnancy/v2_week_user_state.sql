-- Pregnancy v2 per-week user state (table pregnancy_week_user_state, T-M7-01): bookmark and the done
-- task keys of that week's checklist. Every query is scoped by user_id. Unique (user_id, week).

-- name: GetWeekUserState :one
SELECT * FROM `pregnancy_week_user_state` WHERE user_id = ? AND week = ? LIMIT 1;

-- name: ListWeekUserStates :many
SELECT * FROM `pregnancy_week_user_state` WHERE user_id = ? ORDER BY week;

-- name: ListWeekUserStatesRange :many
SELECT * FROM `pregnancy_week_user_state`
WHERE user_id = sqlc.arg(user_id) AND week >= sqlc.arg(week_from) AND week <= sqlc.arg(week_to)
ORDER BY week;

-- name: UpsertWeekUserState :exec
-- Callers merge the partial PUT (bookmarked?, done_task_keys?) with the current row first.
INSERT INTO `pregnancy_week_user_state` (user_id, week, bookmarked, done_task_keys, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(week), sqlc.arg(bookmarked), sqlc.arg(done_task_keys), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  bookmarked = VALUES(bookmarked),
  done_task_keys = VALUES(done_task_keys),
  updated_at = VALUES(updated_at);
