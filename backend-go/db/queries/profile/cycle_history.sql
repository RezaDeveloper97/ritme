-- The cycle-history reads of ProfileController (show, export, cycle settings). Onboarding no longer writes a seed row
-- here: the declared LMP lives in user_profiles.onboarding_last_period_start (T-M9-27, I-01, D-123).

-- name: ListCycleHistories :many
-- CycleHistory::where('user_id', …)->get() / ->orderBy('period_start_date') (unique-index order).
SELECT * FROM `cycle_histories` WHERE user_id = ? ORDER BY period_start_date, id;
