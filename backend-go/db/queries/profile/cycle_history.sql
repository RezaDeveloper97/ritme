-- ProfileController::syncOnboardingPeriodLog and export.

-- name: ListCycleHistories :many
-- CycleHistory::where('user_id', …)->get() / ->orderBy('period_start_date') (unique-index order).
SELECT * FROM `cycle_histories` WHERE user_id = ? ORDER BY period_start_date, id;

-- name: InsertOnboardingCycleHistory :exec
INSERT INTO `cycle_histories`
    (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed, is_estimated, source, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateOnboardingCycleHistory :exec
UPDATE `cycle_histories` SET
    period_start_date = ?, period_end_date = ?, bleeding_length = ?, is_confirmed = ?, is_estimated = ?,
    source = ?, updated_at = ?
WHERE id = ?;
