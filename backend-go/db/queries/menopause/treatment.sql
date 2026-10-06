-- Menopause treatment & care (CB-MENO-03, tables from goose 00022): HRT / supplement / lifestyle items, their intakes
-- (one row per item and day), the side effects of a day, and the care reminders the items are. Health data: every
-- statement is scoped by user_id in the statement itself (an item id of another user matches nothing).

-- name: GetTreatmentItem :one
SELECT * FROM `treatment_items`
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
LIMIT 1;

-- name: CountTreatmentItems :one
SELECT COUNT(*) FROM `treatment_items` WHERE user_id = ?;

-- name: NextTreatmentSortOrder :one
-- The next sort_order of a kind (appended at the end of its list).
SELECT CAST(COALESCE(MAX(sort_order), -1) + 1 AS SIGNED) AS next_order
FROM `treatment_items`
WHERE user_id = sqlc.arg(user_id) AND kind = sqlc.arg(kind);

-- name: InsertTreatmentItem :execlastid
INSERT INTO `treatment_items`
  (user_id, kind, name, dose, schedule, started_on, review_on, weekly_goal, goal_unit, stopped_on, reminder_id,
   sort_order, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(kind), sqlc.arg(name), sqlc.narg(dose), sqlc.narg(schedule), sqlc.narg(started_on),
   sqlc.narg(review_on), sqlc.narg(weekly_goal), sqlc.narg(goal_unit), sqlc.narg(stopped_on), sqlc.narg(reminder_id),
   sqlc.arg(sort_order), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateTreatmentItem :exec
-- A full replace of the user-editable columns (kind and sort order stay).
UPDATE `treatment_items`
SET name = sqlc.arg(name), dose = sqlc.narg(dose), schedule = sqlc.narg(schedule), started_on = sqlc.narg(started_on),
    review_on = sqlc.narg(review_on), weekly_goal = sqlc.narg(weekly_goal), goal_unit = sqlc.narg(goal_unit),
    stopped_on = sqlc.narg(stopped_on), reminder_id = sqlc.narg(reminder_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteTreatmentItem :execrows
-- Intakes go with it (FK cascade); side effects keep the day and lose the link (SET NULL).
DELETE FROM `treatment_items` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: UpsertTreatmentIntake :exec
-- One intake per (item, day): logging the day again replaces the amount and keeps the first taken_at.
INSERT INTO `treatment_intakes` (user_id, treatment_item_id, intake_date, amount, taken_at, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(treatment_item_id), sqlc.arg(intake_date), sqlc.narg(amount), sqlc.arg(taken_at),
        sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at);

-- name: DeleteTreatmentIntake :execrows
DELETE FROM `treatment_intakes`
WHERE user_id = sqlc.arg(user_id) AND treatment_item_id = sqlc.arg(treatment_item_id)
  AND intake_date = sqlc.arg(intake_date);

-- name: ListReminderIntakeDays :many
-- The days a care reminder was ticked in /care (any slot) in [from, to]: a menopause item that is a care medication
-- counts those days as taken too, so both screens agree.
SELECT DISTINCT reminder_id, intake_date FROM `reminder_intakes`
WHERE user_id = sqlc.arg(user_id) AND intake_date >= sqlc.arg(from_date) AND intake_date <= sqlc.arg(to_date)
ORDER BY intake_date, reminder_id;

-- name: DeleteSideEffectsOn :exec
-- The day's side effects are replaced as a set.
DELETE FROM `side_effect_logs` WHERE user_id = sqlc.arg(user_id) AND log_date = sqlc.arg(log_date);

-- name: InsertSideEffect :exec
INSERT INTO `side_effect_logs` (user_id, treatment_item_id, log_date, code, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.narg(treatment_item_id), sqlc.arg(log_date), sqlc.arg(code), sqlc.arg(now),
        sqlc.arg(now));

-- name: GetStoredLifeMode :one
-- The stored life mode (bloom user_life_profiles.life_mode): the doctor report's menopause section is built for a
-- user in menopause mode only.
SELECT life_mode FROM `user_life_profiles` WHERE user_id = ? LIMIT 1;
