-- Pregnancy tools (bloom B-N5-03; internal/pregnancy/tools): kick-count sessions and contraction sessions. Every
-- query is scoped by user_id. active_lock = 1 marks the running session / contraction; NULL once it ended.

-- name: InsertKickSession :execlastid
INSERT INTO `pregnancy_kick_sessions` (user_id, started_at, kicks, pregnancy_week, active_lock, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(started_at), 0, sqlc.narg(pregnancy_week), 1, sqlc.arg(now), sqlc.arg(now));

-- name: GetKickSession :one
SELECT * FROM `pregnancy_kick_sessions` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: GetActiveKickSession :one
SELECT * FROM `pregnancy_kick_sessions` WHERE user_id = sqlc.arg(user_id) AND active_lock = 1 LIMIT 1;

-- name: AddKick :execrows
-- One kick on a running session, atomically. tenth_kick_at is assigned before kicks (MariaDB evaluates SET left to
-- right on the updated row), so it sees the count before this kick.
UPDATE `pregnancy_kick_sessions`
SET tenth_kick_at = IF(tenth_kick_at IS NULL AND kicks + 1 >= sqlc.arg(target), sqlc.arg(at), tenth_kick_at),
    kicks = kicks + 1, last_kick_at = sqlc.arg(at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND active_lock = 1 AND kicks < sqlc.arg(max_kicks);

-- name: UndoKick :execrows
-- Takes the last kick back (the time to the target is cleared when the count drops under it).
UPDATE `pregnancy_kick_sessions`
SET tenth_kick_at = IF(kicks - 1 < sqlc.arg(target), NULL, tenth_kick_at), kicks = kicks - 1, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND active_lock = 1 AND kicks > 0;

-- name: StopKickSession :execrows
UPDATE `pregnancy_kick_sessions`
SET ended_at = sqlc.arg(ended_at), active_lock = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND active_lock = 1;

-- name: DeleteKickSession :execrows
DELETE FROM `pregnancy_kick_sessions` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ListKickSessions :many
-- The user's ended kick sessions, newest first.
SELECT * FROM `pregnancy_kick_sessions`
WHERE user_id = sqlc.arg(user_id) AND ended_at IS NOT NULL
ORDER BY started_at DESC, id DESC LIMIT ?;

-- name: ListKickSessionsBetween :many
-- Ended sessions started in [date_from, date_to) (the day's kick total written to pregnancy_fetal_movements).
SELECT * FROM `pregnancy_kick_sessions`
WHERE user_id = sqlc.arg(user_id) AND ended_at IS NOT NULL
  AND started_at >= sqlc.arg(date_from) AND started_at < sqlc.arg(date_to)
ORDER BY started_at, id;

-- name: MarkFetalMovementFelt :exec
-- The first counted kick marks the profile's first felt movement (like POST /pregnancy/fetal-movement with felt).
UPDATE `pregnancy_profiles`
SET fetal_movement_felt = 1, first_fetal_movement_date = COALESCE(first_fetal_movement_date, sqlc.arg(day)),
    updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id) AND pregnancy_mode = 1 AND fetal_movement_felt = 0;

-- name: InsertContractionSession :execlastid
INSERT INTO `pregnancy_contraction_sessions` (user_id, started_at, active_lock, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(started_at), 1, sqlc.arg(now), sqlc.arg(now));

-- name: GetContractionSession :one
SELECT * FROM `pregnancy_contraction_sessions` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: GetActiveContractionSession :one
SELECT * FROM `pregnancy_contraction_sessions` WHERE user_id = sqlc.arg(user_id) AND active_lock = 1 LIMIT 1;

-- name: SetContractionAlertAt :exec
UPDATE `pregnancy_contraction_sessions`
SET alert_at = sqlc.arg(alert_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND alert_at IS NULL;

-- name: FinishContractionSession :execrows
UPDATE `pregnancy_contraction_sessions`
SET ended_at = sqlc.arg(ended_at), active_lock = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND active_lock = 1;

-- name: DeleteContractionSession :execrows
DELETE FROM `pregnancy_contraction_sessions` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ListContractionSessions :many
-- The user's ended contraction sessions, newest first.
SELECT * FROM `pregnancy_contraction_sessions`
WHERE user_id = sqlc.arg(user_id) AND ended_at IS NOT NULL
ORDER BY started_at DESC, id DESC LIMIT ?;

-- name: InsertContraction :execlastid
INSERT INTO `pregnancy_contractions` (session_id, user_id, started_at, active_lock, created_at, updated_at)
VALUES (sqlc.arg(session_id), sqlc.arg(user_id), sqlc.arg(started_at), 1, sqlc.arg(now), sqlc.arg(now));

-- name: StopContraction :execrows
UPDATE `pregnancy_contractions`
SET ended_at = sqlc.arg(ended_at), active_lock = NULL, updated_at = sqlc.arg(now)
WHERE session_id = sqlc.arg(session_id) AND user_id = sqlc.arg(user_id) AND active_lock = 1;

-- name: ListSessionContractions :many
SELECT * FROM `pregnancy_contractions`
WHERE session_id = sqlc.arg(session_id) AND user_id = sqlc.arg(user_id)
ORDER BY started_at, id;

-- name: ListContractionsSince :many
-- The user's contractions started on or after since (grouped by session for the history list).
SELECT * FROM `pregnancy_contractions`
WHERE user_id = sqlc.arg(user_id) AND started_at >= sqlc.arg(since)
ORDER BY started_at, id;

-- name: ListActiveSessionContractions :many
-- The contractions of the user's running session (the alert engine's 5-1-1 facts).
SELECT c.* FROM `pregnancy_contractions` c
JOIN `pregnancy_contraction_sessions` s ON s.id = c.session_id
WHERE c.user_id = sqlc.arg(user_id) AND s.user_id = sqlc.arg(user_id) AND s.active_lock = 1
ORDER BY c.started_at, c.id;
