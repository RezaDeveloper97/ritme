-- Reminders by SMS for Ritme Plus (goose 00122, internal/remindersms). The sender passes the Plus users in; nothing
-- here reads a plus_ table.

-- name: ListActiveMedicationsOfUsers :many
SELECT * FROM `reminders`
WHERE `type` = 'medication' AND is_active = 1 AND user_id IN (sqlc.slice('user_ids'))
ORDER BY user_id, id;

-- name: ListActiveAppointmentsOfUsers :many
-- Appointments whose visit is inside the window (the caller derives remind-at and skips cancelled ones).
SELECT * FROM `reminders`
WHERE `type` = 'appointment' AND is_active = 1 AND user_id IN (sqlc.slice('user_ids'))
  AND scheduled_at >= sqlc.arg(from_at) AND scheduled_at < sqlc.arg(to_at)
ORDER BY user_id, scheduled_at, id;

-- name: ClaimReminderSMS :execrows
-- 1 = this process handles the occurrence; 0 = another one already did.
INSERT IGNORE INTO `reminder_sms` (user_id, reminder_id, occurrence, sent, sent_at)
VALUES (?, ?, ?, 0, ?);

-- name: MarkReminderSMSSent :exec
UPDATE `reminder_sms` SET sent = 1 WHERE reminder_id = ? AND occurrence = ?;

-- name: CountReminderSMSSince :one
-- The SMS a user was sent since a moment (the per-day cap).
SELECT COUNT(*) FROM `reminder_sms` WHERE user_id = ? AND sent = 1 AND sent_at >= ?;

-- name: PruneReminderSMS :exec
DELETE FROM `reminder_sms` WHERE sent_at < ?;

-- name: GetUserMobile :one
SELECT mobile FROM `users` WHERE id = ? LIMIT 1;
