-- Placeholder so the `reminder` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/reminder/.

-- name: SampleReminderReminder :one
SELECT * FROM `reminders` WHERE id = ? LIMIT 1;
