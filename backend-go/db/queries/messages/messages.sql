-- Queries of the smart-message system (backend/app/Services/MessageSystem, MessageController).

-- name: ListLiveMessageContents :many
-- MessageContentRepository::rows(): every live (active + approved) row of one locale.
SELECT `group`, item_key, payload FROM `message_contents`
WHERE is_active = 1 AND is_approved = 1 AND locale = ?;

-- name: GetMessageProfile :one
-- $user->profile (hasOne: the first row in index order) — the columns the message system and
-- the legacy HealthDataEngine read.
SELECT id, birthday, period_duration, cycle_duration, last_period_start, user_goal, subscription_type
FROM `user_profiles` WHERE user_id = ? ORDER BY id LIMIT 1;

-- name: ListMessageCycleHistories :many
-- HealthDataEngine::getCycleHistories() (the engine sorts itself).
SELECT id, period_start_date, period_end_date, cycle_length, bleeding_length, is_confirmed, is_estimated,
       source, data_quality_flags
FROM `cycle_histories` WHERE user_id = ? ORDER BY period_start_date, id;

-- name: GetMessageDailyLog :one
-- HealthDataEngine::dailyLogFor(): $user->dailyHealthLogs()->whereDate('log_date', $date)->first().
-- Only the columns MessageManager::extractSymptoms can read exist here (see messages/manager).
SELECT id, log_date, energy_level, sleep_quality FROM `daily_health_logs`
WHERE user_id = sqlc.arg(user_id) AND DATE(log_date) = sqlc.arg(log_date) LIMIT 1;

-- name: ListMessageRecentLogs :many
-- MessageManager::buildContext(): the last 90 days of logs, newest first (PatternLayer input).
SELECT id, log_date, energy_level, sleep_quality FROM `daily_health_logs`
WHERE user_id = sqlc.arg(user_id) AND log_date >= sqlc.arg(date_from) AND log_date <= sqlc.arg(date_to)
ORDER BY log_date DESC;

-- name: GetMessageLifeMode :one
-- B-N2-01: the stored life-stage mode (user_life_profiles.life_mode; no row / NULL = legacy detection).
SELECT life_mode FROM `user_life_profiles` WHERE user_id = ? LIMIT 1;

-- name: GetMessageAccount :one
-- T-M9-49b: the daily's account reads in one row — the companion check (GetUserGender), the stored life mode
-- (GetMessageLifeMode), $user->profile (GetMessageProfile's columns; profile_id NULL = no row) and whether any
-- pregnancy profile exists (the pregnancy profile is read only then).
SELECT
  (SELECT lp.gender FROM `user_life_profiles` lp WHERE lp.user_id = u.id LIMIT 1) AS gender,
  (SELECT lp.life_mode FROM `user_life_profiles` lp WHERE lp.user_id = u.id LIMIT 1) AS life_mode,
  p.id AS profile_id, p.birthday, p.period_duration, p.cycle_duration, p.last_period_start, p.user_goal,
  p.subscription_type,
  CAST(EXISTS (SELECT 1 FROM `pregnancy_profiles` pg WHERE pg.user_id = u.id) AS UNSIGNED) AS has_pregnancy_profile
FROM `users` u
LEFT JOIN `user_profiles` p ON p.id = (SELECT MIN(p2.id) FROM `user_profiles` p2 WHERE p2.user_id = u.id)
WHERE u.id = sqlc.arg(user_id);
