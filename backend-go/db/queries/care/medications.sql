-- Care reminders: medications (T-M3-01). Rows live in `reminders` with type = 'medication';
-- the structured fields are in `meta` (internal/care). Every query is scoped by user_id.

-- name: ListMedications :many
SELECT * FROM `reminders`
WHERE user_id = ? AND `type` = 'medication'
ORDER BY created_at DESC, id DESC;

-- name: ListActiveMedications :many
SELECT * FROM `reminders`
WHERE user_id = ? AND `type` = 'medication' AND is_active = 1
ORDER BY created_at DESC, id DESC;

-- name: GetMedication :one
SELECT * FROM `reminders`
WHERE id = ? AND user_id = ? AND `type` = 'medication'
LIMIT 1;

-- name: InsertMedication :execlastid
INSERT INTO `reminders` (
    user_id, `type`, title, subtitle, notes, scheduled_at, recurrence, recurrence_time,
    starts_on, ends_on, is_active, meta, created_at, updated_at
) VALUES (?, 'medication', ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateMedication :exec
UPDATE `reminders`
SET title = ?, subtitle = ?, notes = ?, recurrence = ?, recurrence_time = ?,
    starts_on = ?, ends_on = ?, is_active = ?, meta = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND `type` = 'medication';

-- name: UpdateMedicationSwitches :exec
-- A PUT that only flips is_active / notify (review #6, T-M2-34): no full re-validation, so legacy
-- rows (POST /reminders: no starts_on / times) can be toggled too.
UPDATE `reminders`
SET is_active = ?, meta = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND `type` = 'medication';

-- name: DeleteMedication :execrows
DELETE FROM `reminders`
WHERE id = ? AND user_id = ? AND `type` = 'medication';

-- name: ActivePregnancyDueDate :one
-- The due date of the user's active pregnancy (pregnancy mode on); drives duration = pregnancy_end.
SELECT estimated_due_date FROM `pregnancy_profiles`
WHERE user_id = ? AND pregnancy_mode = 1
LIMIT 1;
