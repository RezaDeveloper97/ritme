-- Message fact registry v6 (T-M10-03, D-161): the small read-only queries behind the lazy fact producers
-- (internal/messages/facts). Every query is scoped by user_id (IDOR) and runs at most once per request.

-- name: GetMessageFactProfile :one
-- account, contraception, postpartum, logs and the vitals context in one row: the user (name, signup), her first
-- user_profiles row (hasOne, the same row GetMessageProfile reads), the life profile (stored life mode, onboarding
-- medications, chronic illnesses), the active contraception method (a row exists only while tracking, I-03), the
-- postpartum birth date, whether a pregnancy is active (pregnancy_mode = 1: the pregnancy profile is read only then),
-- the log activity (newest log day up to today, distinct log days in [since, today]) and the Plus tier at now (a
-- covering active / canceled subscription → plus; else a running trial → trial; else free — plus.tierOf). The joined
-- tables hold at most one row per user, so nothing fans out.
SELECT
  u.id AS user_id,
  u.name,
  u.created_at,
  p.birthday,
  p.user_goal,
  p.subscription_type,
  lp.life_mode,
  lp.medications,
  lp.chronic_illnesses,
  cm.method AS contraception_method,
  pp.birth_date AS postpartum_birth_date,
  CAST(EXISTS (
    SELECT 1 FROM `pregnancy_profiles` pg WHERE pg.user_id = u.id AND pg.pregnancy_mode = 1
  ) AS UNSIGNED) AS pregnancy_active,
  (SELECT MAX(e.log_date) FROM `health_log_entries` e
    WHERE e.user_id = u.id AND e.log_date <= sqlc.arg(today)) AS last_log_date,
  (SELECT COUNT(DISTINCT e.log_date) FROM `health_log_entries` e
    WHERE e.user_id = u.id AND e.log_date >= sqlc.arg(since) AND e.log_date <= sqlc.arg(today)) AS log_days_since,
  CAST(EXISTS (
    SELECT 1 FROM `plus_subscriptions` s
    WHERE s.user_id = u.id AND s.status IN ('active', 'canceled')
      AND s.starts_at <= sqlc.arg(now) AND s.ends_at > sqlc.arg(now)
  ) AS UNSIGNED) AS plus_active,
  CAST(EXISTS (
    SELECT 1 FROM `plus_trials` t
    WHERE t.user_id = u.id AND t.started_at <= sqlc.arg(now) AND t.ends_at > sqlc.arg(now)
  ) AS UNSIGNED) AS trial_running
FROM `users` u
LEFT JOIN `user_profiles` p ON p.id = (SELECT MIN(p2.id) FROM `user_profiles` p2 WHERE p2.user_id = u.id)
LEFT JOIN `user_life_profiles` lp ON lp.user_id = u.id
LEFT JOIN `contraception_methods` cm ON cm.user_id = u.id
LEFT JOIN `postpartum_profiles` pp ON pp.user_id = u.id
WHERE u.id = sqlc.arg(user_id);

-- name: GetMessageFactLogStats :one
-- logs: the newest log day up to today and the number of distinct log days in [since, today] (logs_30d) — the
-- profile row carries the same numbers for the request's own today; this is the fallback for another window.
SELECT
  MAX(log_date) AS last_log_date,
  COUNT(DISTINCT CASE WHEN log_date >= sqlc.arg(since) THEN log_date END) AS days_since
FROM `health_log_entries`
WHERE user_id = sqlc.arg(user_id) AND log_date <= sqlc.arg(today);

-- name: ListMessageFactSymptomEntries :many
-- symptom_history: the symptom items (pain locations, digestive and general symptoms) logged in [date_from, date_to].
SELECT log_date, category, param, item, value_code, value_num
FROM `health_log_entries`
WHERE user_id = sqlc.arg(user_id)
  AND ((category = 'pain' AND param = 'location') OR (category = 'symptoms' AND param IN ('digestive', 'general')))
  AND item <> ''
  AND log_date >= sqlc.arg(date_from) AND log_date <= sqlc.arg(date_to)
ORDER BY log_date;

-- name: ListMessageFactVitalReadings :many
-- vitals: the timed blood pressure and glucose readings of [date_from, date_to), newest first (vitals ListReadings).
SELECT * FROM `vital_readings`
WHERE user_id = sqlc.arg(user_id) AND type IN ('bp', 'glucose')
  AND measured_at >= sqlc.arg(date_from) AND measured_at < sqlc.arg(date_to)
ORDER BY measured_at DESC, id DESC
LIMIT 500;

-- name: ListMessageFactLogMeasurements :many
-- vitals: the log-sheet day values of [date_from, date_to], merged into the timed readings exactly as the vitals
-- views merge them (vitals ListLogMeasurements, internal/vitals/merge.go).
SELECT log_date, param, value_num FROM `health_log_entries`
WHERE user_id = sqlc.arg(user_id) AND category = 'measurements'
  AND param IN ('bp_systolic', 'bp_diastolic', 'heart_rate', 'blood_sugar')
  AND item = '' AND value_num IS NOT NULL
  AND log_date >= sqlc.arg(date_from) AND log_date <= sqlc.arg(date_to)
ORDER BY log_date DESC;

-- name: ListMessageFactVitalTargets :many
-- vitals: a diabetic user's glucose targets by context (vitals ListVitalTargets).
SELECT * FROM `vital_targets` WHERE user_id = sqlc.arg(user_id) ORDER BY context;
