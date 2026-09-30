-- Inputs of the cycle engines (CycleCalculationController, HealthDataEngine, CycleEngineCache,
-- RecommendationRepository). One query per input per request.

-- name: GetProfileByUserID :one
-- $user->profile (hasOne without ordering: the lowest id on MariaDB).
SELECT * FROM `user_profiles` WHERE user_id = ? ORDER BY id LIMIT 1;

-- name: ListCycleHistoriesNewestFirst :many
-- HealthDataEngine::getCycleHistories: CycleHistory::where('user_id')->orderBy('period_start_date', 'desc').
SELECT * FROM `cycle_histories` WHERE user_id = ? ORDER BY period_start_date DESC;

-- name: ListDailyLogsBetween :many
-- HealthDataEngine::preloadDailyLogs: whereBetween('log_date', [from, to 23:59:59]) on a DATE column.
SELECT * FROM `daily_health_logs`
WHERE user_id = sqlc.arg(user_id) AND log_date BETWEEN sqlc.arg(from_date) AND sqlc.arg(to_date)
ORDER BY log_date;

-- name: ListActiveRecommendations :many
-- RecommendationRepository::rows(): Recommendation::active()->orderBy('sort_order')->orderBy('id').
SELECT * FROM `recommendations` WHERE is_active = 1 ORDER BY sort_order, id;

-- name: AnyRecommendationExists :one
-- RecommendationRepository::hasContent(): Recommendation::query()->exists() (inactive rows count).
SELECT EXISTS (SELECT 1 FROM `recommendations`) AS present;

-- name: GetEngineProfileByUserID :one
-- GetProfileByUserID plus B-N1-09 «خودکار از داده‌ها» (goose 00013) in the same round trip: no
-- cycle_preferences row = automatic (1); 0 = the engine prefers the profile lengths.
SELECT sqlc.embed(p), COALESCE(cp.lengths_auto, 1) AS lengths_auto
FROM `user_profiles` p
LEFT JOIN `cycle_preferences` cp ON cp.user_id = p.user_id
WHERE p.user_id = ?
ORDER BY p.id
LIMIT 1;
