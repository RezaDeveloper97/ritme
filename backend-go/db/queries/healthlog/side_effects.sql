-- Side effects of POST /health-logs on user_profiles and cycle_histories
-- (CycleHistoryService, UserProfile::markRecalculated).

-- name: GetUserProfile :one
-- $user->profile (hasOne without ordering: the lowest id on MariaDB).
SELECT * FROM `user_profiles` WHERE user_id = ? ORDER BY id LIMIT 1;

-- name: UpdateProfileLastPeriodStart :exec
-- CycleHistoryService::updateProfileLMP ($profile->update(['last_period_start' => …]) when dirty).
UPDATE `user_profiles` SET last_period_start = ?, updated_at = ? WHERE id = ?;

-- name: MarkProfileRecalculated :exec
-- UserProfile::markRecalculated(): increment('calculation_version', 1, [status, started, completed])
-- (Eloquent's increment also touches updated_at).
UPDATE `user_profiles`
SET calculation_version = calculation_version + 1,
    calculation_status = 'completed',
    calculation_started_at = sqlc.arg(now),
    calculation_completed_at = sqlc.arg(now),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: GetCycleHistoryByStart :one
SELECT * FROM `cycle_histories` WHERE user_id = ? AND period_start_date = ? LIMIT 1;

-- name: GetLatestCycleHistory :one
-- CycleHistory::where('user_id', $u)->orderBy('period_start_date', 'desc')->first().
SELECT * FROM `cycle_histories` WHERE user_id = ? ORDER BY period_start_date DESC LIMIT 1;

-- name: InsertCycleHistory :execlastid
-- The bleeding-log bridge (engine v6, T-M9-28, DECISIONS.md I-02, D-124 / D-138): a period built from `flow` logs is
-- actual data — source bleeding_log, confirmed (Laravel's CycleHistory::create wrote an unconfirmed user_logged row,
-- D-55). period_end_date / bleeding_length are NULL for an open period. is_estimated / data_quality_flags keep their
-- column defaults; cycle_length is written by periods.RecomputeCycleLengths in the same transaction.
INSERT INTO `cycle_histories`
  (user_id, period_start_date, period_end_date, cycle_length, bleeding_length, is_confirmed, source, created_at, updated_at)
VALUES
  (sqlc.arg(user_id), sqlc.arg(period_start_date), sqlc.narg(period_end_date), sqlc.narg(cycle_length),
   sqlc.narg(bleeding_length), 1, 'bleeding_log', sqlc.arg(now), sqlc.arg(now));

-- name: UpdateCycleHistoryEnd :exec
UPDATE `cycle_histories` SET period_end_date = ?, bleeding_length = ?, updated_at = ? WHERE id = ?;

-- name: GetCycleHistoryOnOrBefore :one
-- B-N3-14b (D-55): the period that starts on or before a back-dated bleeding day (it may contain the day).
-- Estimates are no periods for the bridge (T-M9-28).
SELECT * FROM `cycle_histories` WHERE user_id = sqlc.arg(user_id) AND period_start_date <= sqlc.arg(on_date)
  AND is_estimated = 0
ORDER BY period_start_date DESC LIMIT 1;

-- name: GetCycleHistoryAfter :one
-- B-N3-14b (D-55): the first period that starts after a bleeding day (it may start a day or two later).
-- Estimates are no periods for the bridge (T-M9-28).
SELECT * FROM `cycle_histories` WHERE user_id = sqlc.arg(user_id) AND period_start_date > sqlc.arg(after_date)
  AND is_estimated = 0
ORDER BY period_start_date ASC LIMIT 1;

-- name: UpdateCycleHistoryStart :exec
-- B-N3-14b (D-55): a bleeding day logged right before a period moves its start back.
UPDATE `cycle_histories` SET period_start_date = ?, bleeding_length = ?, updated_at = ? WHERE id = ?;

-- name: UpdateBleedingLogPeriod :exec
-- T-M9-28 (I-02): the bridge rewrites only its own rows (source bleeding_log); a button row is never touched.
UPDATE `cycle_histories`
SET period_start_date = sqlc.arg(period_start_date), period_end_date = sqlc.narg(period_end_date),
    bleeding_length = sqlc.narg(bleeding_length), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND source = 'bleeding_log';

-- name: DeleteBleedingLogPeriod :exec
-- T-M9-28 (I-02): a bleeding_log period with no bleeding day left is deleted (never a button row).
DELETE FROM `cycle_histories` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND source = 'bleeding_log';

-- name: ListBleedingDaysBetween :many
-- T-M9-28 (I-02): the user's bleeding days (a `flow` value) in [from_date, to_date], oldest first — what a
-- bleeding_log period is rebuilt from.
SELECT log_date FROM `daily_health_logs`
WHERE user_id = sqlc.arg(user_id)
  AND log_date >= sqlc.arg(from_date)
  AND log_date <= sqlc.arg(to_date)
  AND bleeding_intensity IS NOT NULL AND bleeding_intensity <> ''
ORDER BY log_date;

-- name: GetLogPostpartumBirth :one
-- T-M9-28 (I-05): the birth date of a postpartum user — flow before birth + postpartum.flow_from_day is lochia.
SELECT birth_date FROM `postpartum_profiles` WHERE user_id = ? LIMIT 1;
