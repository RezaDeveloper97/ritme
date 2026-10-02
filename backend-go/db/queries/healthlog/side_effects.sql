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
-- CycleHistory::create([user_id, period_start_date, cycle_length, is_confirmed => false]);
-- is_estimated / source / data_quality_flags keep their column defaults.
INSERT INTO `cycle_histories` (user_id, period_start_date, cycle_length, is_confirmed, created_at, updated_at)
VALUES (?, ?, ?, 0, ?, ?);

-- name: UpdateCycleHistoryEnd :exec
UPDATE `cycle_histories` SET period_end_date = ?, bleeding_length = ?, updated_at = ? WHERE id = ?;

-- name: GetCycleHistoryOnOrBefore :one
-- B-N3-14b (D-55): the period that starts on or before a back-dated bleeding day (it may contain the day).
SELECT * FROM `cycle_histories` WHERE user_id = sqlc.arg(user_id) AND period_start_date <= sqlc.arg(on_date)
ORDER BY period_start_date DESC LIMIT 1;

-- name: GetCycleHistoryAfter :one
-- B-N3-14b (D-55): the first period that starts after a bleeding day (it may start a day or two later).
SELECT * FROM `cycle_histories` WHERE user_id = sqlc.arg(user_id) AND period_start_date > sqlc.arg(after_date)
ORDER BY period_start_date ASC LIMIT 1;

-- name: UpdateCycleHistoryStart :exec
-- B-N3-14b (D-55): a bleeding day logged right before a period moves its start back.
UPDATE `cycle_histories` SET period_start_date = ?, bleeding_length = ?, updated_at = ? WHERE id = ?;
