-- Contraception (CB-CONTRA-01, goose 00017). Health data: every statement is scoped by user_id in the statement
-- itself. The «track contraception» flag is user_life_profiles.track_contraception (B-N2-01) and the pill reminder
-- is notification_preferences' `pill` category (B-N1-09) — written here, never copied.

-- name: GetMethod :one
SELECT * FROM `contraception_methods` WHERE user_id = ? LIMIT 1;

-- name: UpsertMethod :exec
-- One method per user; saving again replaces every field.
INSERT INTO `contraception_methods`
  (user_id, method, pack_type, pack_started_on, packs_left, packs_counted_on, inserted_on, iud_lifetime_years,
   followup_done, injected_on, replace_on, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(method), sqlc.narg(pack_type), sqlc.narg(pack_started_on), sqlc.narg(packs_left),
   sqlc.narg(packs_counted_on), sqlc.narg(inserted_on), sqlc.narg(iud_lifetime_years), sqlc.arg(followup_done),
   sqlc.narg(injected_on), sqlc.narg(replace_on), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  method = VALUES(method),
  pack_type = VALUES(pack_type),
  pack_started_on = VALUES(pack_started_on),
  packs_left = VALUES(packs_left),
  packs_counted_on = VALUES(packs_counted_on),
  inserted_on = VALUES(inserted_on),
  iud_lifetime_years = VALUES(iud_lifetime_years),
  followup_done = VALUES(followup_done),
  injected_on = VALUES(injected_on),
  replace_on = VALUES(replace_on),
  updated_at = VALUES(updated_at);

-- name: DeleteMethod :exec
DELETE FROM `contraception_methods` WHERE user_id = ?;

-- name: GetTrackContraception :one
-- The B-N2-01 switch; false without a life-profile row.
SELECT EXISTS(SELECT 1 FROM `user_life_profiles` WHERE user_id = ? AND track_contraception = 1) AS tracking;

-- name: SetTrackContraception :exec
-- Flips only the switch (a missing row is created with the column defaults, like PUT /profile/life-stage).
INSERT INTO `user_life_profiles` (user_id, track_contraception, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(track_contraception), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  track_contraception = VALUES(track_contraception),
  updated_at = VALUES(updated_at);

-- name: UpsertPillLog :exec
INSERT INTO `contraception_pill_logs` (user_id, log_date, status, logged_at, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(log_date), sqlc.arg(status), sqlc.arg(logged_at), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  status = VALUES(status),
  logged_at = VALUES(logged_at),
  updated_at = VALUES(updated_at);

-- name: DeletePillLog :execrows
DELETE FROM `contraception_pill_logs` WHERE user_id = ? AND log_date = ?;

-- name: ListPillLogs :many
-- The user's pill days from `from_date` to `to_date` (inclusive), newest first (pack grid and streak).
SELECT log_date, status FROM `contraception_pill_logs`
WHERE user_id = sqlc.arg(user_id) AND log_date >= sqlc.arg(from_date) AND log_date <= sqlc.arg(to_date)
ORDER BY log_date DESC;

-- name: ListReminderLinks :many
-- The care reminders the method created, with their current row (a reminder the user deleted has no link left).
SELECT l.kind, sqlc.embed(r)
FROM `contraception_reminders` l
JOIN `reminders` r ON r.id = l.reminder_id AND r.user_id = l.user_id
WHERE l.user_id = ?
ORDER BY l.id;

-- name: InsertReminderLink :exec
INSERT INTO `contraception_reminders` (user_id, kind, reminder_id, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(kind), sqlc.arg(reminder_id), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  reminder_id = VALUES(reminder_id),
  updated_at = VALUES(updated_at);

-- name: InsertReminder :execlastid
-- A care reminder (type appointment | custom) created for the method; it lives in `reminders` like any other, so
-- /care/appointments and GET /reminders show it and the user can edit, switch off or delete it.
INSERT INTO `reminders` (
    user_id, `type`, title, subtitle, notes, scheduled_at, recurrence, recurrence_time,
    starts_on, ends_on, is_active, meta, created_at, updated_at
) VALUES (
    sqlc.arg(user_id), sqlc.arg(type), sqlc.arg(title), NULL, NULL, sqlc.narg(scheduled_at), sqlc.arg(recurrence),
    sqlc.narg(recurrence_time), sqlc.narg(starts_on), NULL, 1, sqlc.narg(meta), sqlc.arg(now), sqlc.arg(now)
);

-- name: UpdateReminderSchedule :exec
-- A method change moves the reminder's date; the user's own edits (switch, notes, appointment details) stay.
UPDATE `reminders`
SET title = sqlc.arg(title), scheduled_at = sqlc.narg(scheduled_at), recurrence = sqlc.arg(recurrence),
    recurrence_time = sqlc.narg(recurrence_time), starts_on = sqlc.narg(starts_on), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeleteReminder :exec
-- Removes a method reminder (its link goes by FK cascade).
DELETE FROM `reminders` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);
