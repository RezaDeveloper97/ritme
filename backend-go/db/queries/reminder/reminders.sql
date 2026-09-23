-- Reminders (App\Models\Reminder, ReminderController).

-- name: ListReminders :many
-- $user->reminders()->orderByDesc('created_at')->get(). Ties (same created_at) come back in
-- primary-key order on MariaDB (index scan by user_id → id), made explicit here.
SELECT * FROM `reminders`
WHERE user_id = ?
ORDER BY created_at DESC, id ASC;

-- name: ListRemindersByType :many
-- ...->ofType($type).
SELECT * FROM `reminders`
WHERE user_id = ? AND `type` = ?
ORDER BY created_at DESC, id ASC;

-- name: GetUserReminder :one
-- $user->reminders()->find($id).
SELECT * FROM `reminders` WHERE id = ? AND user_id = ? LIMIT 1;

-- name: InsertReminder :execlastid
INSERT INTO `reminders` (
    user_id, `type`, title, subtitle, notes, scheduled_at, recurrence, recurrence_time,
    starts_on, ends_on, is_active, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateReminder :exec
-- $reminder->update($validated) when at least one attribute is dirty (the full row is
-- written back; untouched columns keep their loaded value).
UPDATE `reminders`
SET `type` = ?, title = ?, subtitle = ?, notes = ?, scheduled_at = ?, recurrence = ?,
    recurrence_time = ?, starts_on = ?, ends_on = ?, is_active = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteReminder :exec
DELETE FROM `reminders` WHERE id = ?;
