-- Care reminders: doses taken (reminder_intakes, T-M3-01). One row per (reminder, day, slot).

-- name: InsertIntake :execrows
-- Idempotent tick: a second tick of the same dose keeps the first taken_at.
INSERT IGNORE INTO `reminder_intakes` (user_id, reminder_id, intake_date, slot, taken_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetIntake :one
SELECT * FROM `reminder_intakes`
WHERE reminder_id = ? AND user_id = ? AND intake_date = ? AND slot = ?
LIMIT 1;

-- name: DeleteIntake :execrows
DELETE FROM `reminder_intakes`
WHERE reminder_id = ? AND user_id = ? AND intake_date = ? AND slot = ?;

-- name: ListIntakesOnDate :many
-- Every dose the user took on one day (GET /care/today); uses INDEX (user_id, intake_date).
SELECT reminder_id, slot FROM `reminder_intakes`
WHERE user_id = ? AND intake_date = ?;
