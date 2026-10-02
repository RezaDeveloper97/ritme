-- IVF treatment (CB-IVF-01, goose 00027). Health data: every statement is scoped by user_id in the statement itself.
-- Medicines are care medication reminders and doses taken are care intakes (internal/care store, reused as is);
-- the statements here only read/write the ivf_* side tables, the life-profile switch and the few `reminders` /
-- `reminder_intakes` reads the IVF screens need.

-- name: GetIVFSwitch :one
-- bloom's «IVF/IUI» switch (B-N2-03); false without a life-profile row.
SELECT EXISTS(SELECT 1 FROM `user_life_profiles` WHERE user_id = ? AND ivf_iui = 1) AS ivf_iui;

-- name: SetIVFSwitch :exec
-- Flips only the switch (a missing row is created with the column defaults, like PUT /profile/life-stage).
INSERT INTO `user_life_profiles` (user_id, ivf_iui, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(ivf_iui), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  ivf_iui = VALUES(ivf_iui),
  updated_at = VALUES(updated_at);

-- name: GetActiveCycle :one
SELECT * FROM `ivf_cycles` WHERE user_id = ? AND active_user_id IS NOT NULL LIMIT 1;

-- name: LockActiveCycle :one
-- The open cycle, row-locked for a write transaction.
SELECT * FROM `ivf_cycles` WHERE user_id = ? AND active_user_id IS NOT NULL LIMIT 1 FOR UPDATE;

-- name: CycleStats :one
-- How many cycles the user has recorded and the highest number (the next cycle's default number is max + 1).
SELECT COUNT(*) AS cycles, CAST(COALESCE(MAX(number), 0) AS UNSIGNED) AS max_number
FROM `ivf_cycles` WHERE user_id = ?;

-- name: InsertCycle :execlastid
INSERT INTO `ivf_cycles`
  (user_id, active_user_id, number, protocol, stage, started_on, stim_started_on, notify_companion, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(active_user_id), sqlc.arg(number), sqlc.narg(protocol), sqlc.arg(stage), sqlc.arg(started_on),
   sqlc.narg(stim_started_on), sqlc.arg(notify_companion), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateCycle :exec
-- Every editable field of an open cycle (the service merges a partial PUT over the stored row).
UPDATE `ivf_cycles`
SET number = sqlc.arg(number), protocol = sqlc.narg(protocol), stage = sqlc.arg(stage), started_on = sqlc.arg(started_on),
    stim_started_on = sqlc.narg(stim_started_on), retrieval_at = sqlc.narg(retrieval_at),
    transfer_at = sqlc.narg(transfer_at), beta_on = sqlc.narg(beta_on), next_scan_at = sqlc.narg(next_scan_at),
    notify_companion = sqlc.arg(notify_companion), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: CloseCycle :exec
-- Records the result and closes the cycle (frees the one-open-cycle slot).
UPDATE `ivf_cycles`
SET active_user_id = NULL, outcome = sqlc.arg(outcome), outcome_on = sqlc.arg(outcome_on), closed_at = sqlc.arg(now),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: ListCycleMeds :many
-- The cycle's medicines with the care reminder each one is, in the order they were added.
SELECT sqlc.embed(m), sqlc.embed(r)
FROM `ivf_meds` m
JOIN `reminders` r ON r.id = m.reminder_id AND r.user_id = m.user_id
WHERE m.user_id = ? AND m.cycle_id = ?
ORDER BY m.id;

-- name: GetMed :one
SELECT sqlc.embed(m), sqlc.embed(r)
FROM `ivf_meds` m
JOIN `reminders` r ON r.id = m.reminder_id AND r.user_id = m.user_id
WHERE m.id = ? AND m.user_id = ?
LIMIT 1;

-- name: InsertMed :execlastid
INSERT INTO `ivf_meds`
  (user_id, cycle_id, reminder_id, role, route, trigger_at, stock_units, stock_unit, doses_per_unit, stock_counted_at,
   created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(cycle_id), sqlc.arg(reminder_id), sqlc.arg(role), sqlc.arg(route), sqlc.narg(trigger_at),
   sqlc.narg(stock_units), sqlc.narg(stock_unit), sqlc.arg(doses_per_unit), sqlc.narg(stock_counted_at),
   sqlc.arg(now), sqlc.arg(now));

-- name: UpdateMed :exec
UPDATE `ivf_meds`
SET role = sqlc.arg(role), route = sqlc.arg(route), trigger_at = sqlc.narg(trigger_at),
    stock_units = sqlc.narg(stock_units), stock_unit = sqlc.narg(stock_unit), doses_per_unit = sqlc.arg(doses_per_unit),
    stock_counted_at = sqlc.narg(stock_counted_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: CountIntakesSince :one
-- Doses of a medicine ticked (care intakes) since the stock was counted — they used the stock up.
SELECT COUNT(*) FROM `reminder_intakes`
WHERE user_id = sqlc.arg(user_id) AND reminder_id = sqlc.arg(reminder_id) AND taken_at >= sqlc.arg(since);

-- name: SetReminderActive :exec
-- Switches a cycle's medicine reminder on/off (a negative or cancelled cycle stops its injection reminders).
UPDATE `reminders` SET is_active = sqlc.arg(is_active), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: UpsertDoseLog :exec
INSERT INTO `ivf_dose_logs` (user_id, ivf_med_id, dose_date, slot, site, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(ivf_med_id), sqlc.arg(dose_date), sqlc.arg(slot), sqlc.narg(site), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  site = VALUES(site),
  updated_at = VALUES(updated_at);

-- name: DeleteDoseLog :exec
DELETE FROM `ivf_dose_logs`
WHERE user_id = sqlc.arg(user_id) AND ivf_med_id = sqlc.arg(ivf_med_id) AND dose_date = sqlc.arg(dose_date)
  AND slot = sqlc.arg(slot);

-- name: ListDoseLogsBetween :many
-- The injection sites logged from `from_date` to `to_date` (inclusive).
SELECT ivf_med_id, dose_date, slot, site FROM `ivf_dose_logs`
WHERE user_id = sqlc.arg(user_id) AND dose_date >= sqlc.arg(from_date) AND dose_date <= sqlc.arg(to_date);

-- name: RecentSites :many
-- The most recent injection sites, newest first (the rotation reads the last use of each site).
SELECT site, dose_date, slot FROM `ivf_dose_logs`
WHERE user_id = ? AND site IS NOT NULL
ORDER BY dose_date DESC, slot DESC, id DESC
LIMIT 200;

-- name: ListIntakesBetween :many
-- Doses ticked (care intakes) from `from_date` to `to_date` (inclusive).
SELECT reminder_id, intake_date, slot FROM `reminder_intakes`
WHERE user_id = sqlc.arg(user_id) AND intake_date >= sqlc.arg(from_date) AND intake_date <= sqlc.arg(to_date);

-- name: ListScans :many
SELECT * FROM `ivf_scans` WHERE user_id = ? AND cycle_id = ? ORDER BY scan_date;

-- name: UpsertScan :exec
-- One scan per cycle day; saving the day again replaces it.
INSERT INTO `ivf_scans`
  (user_id, cycle_id, scan_date, right_lt_10, right_10_14, right_15_17, right_18_plus,
   left_lt_10, left_10_14, left_15_17, left_18_plus, endometrium_mm, e2, e2_unit, notes, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(cycle_id), sqlc.arg(scan_date), sqlc.arg(right_lt_10), sqlc.arg(right_10_14),
   sqlc.arg(right_15_17), sqlc.arg(right_18_plus), sqlc.arg(left_lt_10), sqlc.arg(left_10_14), sqlc.arg(left_15_17),
   sqlc.arg(left_18_plus), sqlc.narg(endometrium_mm), sqlc.narg(e2), sqlc.narg(e2_unit), sqlc.narg(notes),
   sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  right_lt_10 = VALUES(right_lt_10), right_10_14 = VALUES(right_10_14), right_15_17 = VALUES(right_15_17),
  right_18_plus = VALUES(right_18_plus), left_lt_10 = VALUES(left_lt_10), left_10_14 = VALUES(left_10_14),
  left_15_17 = VALUES(left_15_17), left_18_plus = VALUES(left_18_plus), endometrium_mm = VALUES(endometrium_mm),
  e2 = VALUES(e2), e2_unit = VALUES(e2_unit), notes = VALUES(notes), updated_at = VALUES(updated_at);

-- name: DeleteScan :execrows
DELETE FROM `ivf_scans` WHERE user_id = ? AND cycle_id = ? AND scan_date = ?;

-- name: ListTWWLogs :many
SELECT log_date, mood FROM `ivf_tww_logs` WHERE user_id = ? AND cycle_id = ? ORDER BY log_date;

-- name: UpsertTWWLog :exec
INSERT INTO `ivf_tww_logs` (user_id, cycle_id, log_date, mood, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(cycle_id), sqlc.arg(log_date), sqlc.arg(mood), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  mood = VALUES(mood),
  updated_at = VALUES(updated_at);

-- name: DeleteTWWLog :exec
DELETE FROM `ivf_tww_logs` WHERE user_id = ? AND cycle_id = ? AND log_date = ?;

-- name: ListReminderLinks :many
-- The care appointments the cycle dates created, with their current row (a reminder the user deleted has no link).
SELECT l.kind, sqlc.embed(r)
FROM `ivf_reminders` l
JOIN `reminders` r ON r.id = l.reminder_id AND r.user_id = l.user_id
WHERE l.user_id = ? AND l.cycle_id = ?
ORDER BY l.id;

-- name: InsertReminderLink :exec
INSERT INTO `ivf_reminders` (user_id, cycle_id, kind, reminder_id, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(cycle_id), sqlc.arg(kind), sqlc.arg(reminder_id), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  reminder_id = VALUES(reminder_id),
  updated_at = VALUES(updated_at);

-- name: MoveAppointment :exec
-- A changed cycle date moves its appointment; the user's own edits (title, notes, details, switch) stay.
UPDATE `reminders` SET scheduled_at = sqlc.arg(scheduled_at), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND `type` = 'appointment';
