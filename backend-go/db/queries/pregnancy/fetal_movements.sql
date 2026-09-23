-- PregnancyFetalMovement (App\Models\PregnancyFetalMovement). Unique (user_id, log_date).

-- name: ListFetalMovements :many
SELECT * FROM `pregnancy_fetal_movements`
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.arg(has_from) = 0 OR log_date >= CAST(sqlc.arg(from_value) AS CHAR))
  AND (sqlc.arg(has_to) = 0 OR log_date <= CAST(sqlc.arg(to_value) AS CHAR))
ORDER BY log_date DESC;

-- name: GetFetalMovement :one
SELECT * FROM `pregnancy_fetal_movements` WHERE user_id = ? AND log_date = ? LIMIT 1;

-- name: InsertFetalMovement :execlastid
INSERT INTO `pregnancy_fetal_movements` (
  user_id, log_date, pregnancy_week, movement_status, movement_count, first_movement_time, last_movement_time,
  notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateFetalMovement :exec
UPDATE `pregnancy_fetal_movements` SET
  pregnancy_week = ?, movement_status = ?, movement_count = ?, first_movement_time = ?, last_movement_time = ?,
  notes = ?, updated_at = ?
WHERE id = ?;
