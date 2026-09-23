-- PeriodLogController (backend/app/Http/Controllers/Api/V1/PeriodLogController.php).

-- name: GetOngoingPeriod :one
-- ongoingPeriod(): whereNull('period_end_date')->orderBy('period_start_date', 'desc')->first().
SELECT * FROM `cycle_histories`
WHERE user_id = ? AND period_end_date IS NULL
ORDER BY period_start_date DESC LIMIT 1;

-- name: GetLatestPeriod :one
-- CycleHistory::where('user_id')->orderBy('period_start_date', 'desc')->first().
SELECT * FROM `cycle_histories` WHERE user_id = ? ORDER BY period_start_date DESC LIMIT 1;

-- name: GetPeriodByStart :one
-- whereDate('period_start_date', $start)->first().
SELECT * FROM `cycle_histories` WHERE user_id = ? AND period_start_date = ? LIMIT 1;

-- name: GetPreviousPeriod :one
-- The period before a new start: whereDate('period_start_date', '<', $start)->orderBy(desc)->first().
SELECT * FROM `cycle_histories`
WHERE user_id = ? AND period_start_date < ?
ORDER BY period_start_date DESC LIMIT 1;

-- name: GetBlockingOpenPeriod :one
-- blockingOpenPeriod(): an open, non-estimated period that started before $start but within the
-- hard cap (period_start_date >= today - 12 days).
SELECT * FROM `cycle_histories`
WHERE user_id = sqlc.arg(user_id)
  AND period_end_date IS NULL
  AND is_estimated = 0
  AND period_start_date < sqlc.arg(before_date)
  AND period_start_date >= sqlc.arg(since_date)
ORDER BY period_start_date DESC LIMIT 1;

-- name: GetUserPeriod :one
-- CycleHistory::where('user_id', $u)->find($period).
SELECT * FROM `cycle_histories` WHERE user_id = ? AND id = ? LIMIT 1;

-- name: ListLoggedPeriods :many
-- history() and overlapsExistingPeriod(): the non-estimated periods, newest first.
SELECT * FROM `cycle_histories` WHERE user_id = ? AND is_estimated = 0 ORDER BY period_start_date DESC;

-- name: ListPeriodsOldestFirst :many
-- recomputeCycleLengths(): CycleHistory::where('user_id')->orderBy('period_start_date')->get().
SELECT * FROM `cycle_histories` WHERE user_id = ? ORDER BY period_start_date;

-- name: InsertPeriod :execlastid
-- CycleHistory::create([...]); columns the controller does not set take the table defaults.
INSERT INTO `cycle_histories`
    (user_id, period_start_date, period_end_date, cycle_length, bleeding_length, is_confirmed, is_estimated,
     source, data_quality_flags, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePeriod :exec
-- $period->update([...]) when at least one attribute is dirty (every column written; the clean ones
-- keep their value).
UPDATE `cycle_histories` SET
    period_start_date = ?, period_end_date = ?, cycle_length = ?, bleeding_length = ?, is_confirmed = ?,
    is_estimated = ?, source = ?, data_quality_flags = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteEstimatesExcept :exec
-- CycleHistory::where('user_id')->where('is_estimated', true)->where('id', '!=', $keep)->delete().
DELETE FROM `cycle_histories` WHERE user_id = ? AND is_estimated = 1 AND id <> ?;

-- name: DeletePeriod :exec
DELETE FROM `cycle_histories` WHERE id = ?;

-- name: UpdateProfileLastPeriodStart :exec
-- $profile->update(['last_period_start' => …]) when dirty.
UPDATE `user_profiles` SET last_period_start = ?, updated_at = ? WHERE id = ?;
