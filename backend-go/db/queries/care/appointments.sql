-- Care reminders: doctor appointments (T-M3-02). Rows live in `reminders` with type = 'appointment';
-- the structured fields are in `meta` (internal/care). Every query is scoped by user_id.

-- name: ListAppointments :many
-- Soonest first; the scope (upcoming/past/all) is applied by internal/care.
SELECT * FROM `reminders`
WHERE user_id = ? AND `type` = 'appointment'
ORDER BY scheduled_at ASC, id ASC;

-- name: GetAppointment :one
SELECT * FROM `reminders`
WHERE id = ? AND user_id = ? AND `type` = 'appointment'
LIMIT 1;

-- name: InsertAppointment :execlastid
INSERT INTO `reminders` (
    user_id, `type`, title, subtitle, notes, scheduled_at, recurrence, recurrence_time,
    starts_on, ends_on, is_active, meta, created_at, updated_at
) VALUES (?, 'appointment', ?, ?, ?, ?, 'none', NULL, NULL, NULL, ?, ?, ?, ?);

-- name: UpdateAppointment :exec
UPDATE `reminders`
SET title = ?, subtitle = ?, notes = ?, scheduled_at = ?, is_active = ?, meta = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND `type` = 'appointment';

-- name: DeleteAppointment :execrows
DELETE FROM `reminders`
WHERE id = ? AND user_id = ? AND `type` = 'appointment';
