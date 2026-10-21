-- Skipped cycles «پریود نداشتم» (T-M9-24, DECISIONS.md E-15; table cycle_skips, goose 00098). One row per skipped
-- predicted start; passed to the engine exactly as stored. Health data: every query is scoped by user_id (IDOR).

-- name: ListCycleSkips :many
SELECT id, user_id, expected_start, created_at, updated_at FROM `cycle_skips`
WHERE user_id = ?
ORDER BY expected_start, id;

-- name: InsertCycleSkip :execrows
-- Idempotent: skipping the same predicted start again changes nothing (0 rows).
INSERT INTO `cycle_skips` (user_id, expected_start, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(expected_start), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE id = id;

-- name: DeleteCycleSkip :execrows
-- Undoing a skip restores the previous engine output (E-15). 0 rows = no such skip of this user.
DELETE FROM `cycle_skips` WHERE user_id = sqlc.arg(user_id) AND expected_start = sqlc.arg(expected_start);
