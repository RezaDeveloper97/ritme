-- Checkups (T-M4-01): the inputs of the status engine (internal/checkups/engine) for one user.
-- Every query is scoped by user_id; the catalog query returns the shared rows plus the user's own
-- custom checkups only.

-- name: ListActiveCheckupTypesForUser :many
SELECT * FROM `checkup_types`
WHERE is_active = 1 AND (user_id IS NULL OR user_id = CAST(sqlc.arg(user_id) AS UNSIGNED))
ORDER BY sort_order, id;

-- name: ListCheckupRecordsForUser :many
-- Newest first per type (the engine reads the latest done_on and its next_due_on override).
SELECT * FROM `checkup_records`
WHERE user_id = ?
ORDER BY checkup_type_id, done_on DESC, id DESC;

-- name: ListCheckupSettingsForUser :many
SELECT * FROM `user_checkup_settings`
WHERE user_id = ?
ORDER BY checkup_type_id;
