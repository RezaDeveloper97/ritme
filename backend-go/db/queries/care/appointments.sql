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

-- name: ListAppointmentsFrom :many
-- Appointments at or after a moment, soonest first, cancelled ones included (the family projection lists both).
-- Bounded by care.MaxAppointments (1000), the hard ceiling on a user's appointment rows, so it never truncates.
SELECT * FROM `reminders`
WHERE user_id = ? AND `type` = 'appointment' AND scheduled_at >= ?
ORDER BY scheduled_at ASC, id ASC
LIMIT 1000;

-- name: NextAppointmentFrom :many
-- GET /care/today's next_appointment: the earliest not-cancelled appointment at or after a moment (zero or one row).
-- meta.status is scheduled | cancelled; anything else reads as scheduled (care.ParseAppointment), so only an exact
-- 'cancelled' is skipped here.
SELECT * FROM `reminders`
WHERE user_id = ? AND `type` = 'appointment' AND scheduled_at >= ?
  AND COALESCE(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.status')), '') <> 'cancelled'
ORDER BY scheduled_at ASC, id ASC
LIMIT 1;
