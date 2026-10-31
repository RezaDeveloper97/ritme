-- SPEC v6 §5.6 KPIs computed by the `admin_metrics:snapshot` job (T-M10-30, D-179; docs/engine-v6/kpi.md). Every
-- query is an aggregate or a per-user fact list the job folds into counts; nothing here leaves internal/metrics
-- with a user id. The KPI endpoint never runs these: it reads the stored snapshot.

-- name: CountLoggersAtLeast :one
-- Users with a data log on at least min_days days of [from_day, to_day] (loggers ≥ 15 days in the month).
SELECT COUNT(*) FROM (
  SELECT `user_id` FROM `user_activity_days`
  WHERE `day` BETWEEN sqlc.arg(from_day) AND sqlc.arg(to_day) AND `logged` = 1
  GROUP BY `user_id`
  HAVING COUNT(*) >= CAST(sqlc.arg(min_days) AS SIGNED)
) AS l;

-- name: ListSignupFacts :many
-- Activation funnel: one row per user who signed up (= verified the OTP) in [from_at, to_at), with the server facts
-- of each step. Day offsets are user-local activity days minus the Tehran signup day.
SELECT u.`id`,
       DATE(u.`created_at`) AS signup_day,
       CAST(EXISTS(SELECT 1 FROM `analytics_events` e WHERE e.`user_id` = u.`id` AND e.`name` = 'app_open') AS SIGNED) AS opt_in_open,
       CAST(EXISTS(SELECT 1 FROM `user_life_profiles` lp WHERE lp.`user_id` = u.`id` AND lp.`onboarding_completed_at` IS NOT NULL) AS SIGNED) AS onboarded,
       CAST(EXISTS(SELECT 1 FROM `user_activity_days` a WHERE a.`user_id` = u.`id` AND a.`logged` = 1) AS SIGNED) AS first_log,
       CAST(EXISTS(SELECT 1 FROM `user_activity_days` a WHERE a.`user_id` = u.`id` AND a.`logged` = 1
         AND a.`day` BETWEEN DATE_ADD(DATE(u.`created_at`), INTERVAL 6 DAY) AND DATE_ADD(DATE(u.`created_at`), INTERVAL 8 DAY)) AS SIGNED) AS logged_d7,
       CAST(EXISTS(SELECT 1 FROM `user_activity_days` a WHERE a.`user_id` = u.`id` AND a.`logged` = 1
         AND a.`day` BETWEEN DATE_ADD(DATE(u.`created_at`), INTERVAL 29 DAY) AND DATE_ADD(DATE(u.`created_at`), INTERVAL 31 DAY)) AS SIGNED) AS logged_d30
FROM `users` u
WHERE u.`created_at` >= sqlc.arg(from_at) AND u.`created_at` < sqlc.arg(to_at);

-- name: ListSignupDays :many
-- Retention cohorts: the users who signed up in [from_at, to_at) with their Tehran signup day.
SELECT `id`, DATE(`created_at`) AS signup_day FROM `users`
WHERE `created_at` >= sqlc.arg(from_at) AND `created_at` < sqlc.arg(to_at);

-- name: ListCohortActivityDays :many
-- Retention cohorts: the activity days of the users who signed up since from_at, up to 62 days after the signup day.
SELECT a.`user_id`, a.`day`, a.`logged` FROM `user_activity_days` a
JOIN `users` u ON u.`id` = a.`user_id`
WHERE u.`created_at` >= sqlc.arg(from_at)
  AND a.`day` BETWEEN DATE(u.`created_at`) AND DATE_ADD(DATE(u.`created_at`), INTERVAL 62 DAY);

-- name: ListResolvedFirstPredictions :many
-- Engine MAE / ±2-day accuracy (A-05): the `first` prediction of each cycle built from ≥ 2 real cycles whose next real
-- start fell in [from_day, to_day], minus the cycles during which a daily snapshot shows pregnancy or postpartum mode
-- or a warning of excluded_mask (hormonal contraception).
SELECT p.`user_id`, p.`error_days` FROM `engine_prediction_log` p
WHERE p.`kind` = 'first' AND p.`basis_cycles` >= 2 AND p.`error_days` IS NOT NULL
  AND p.`actual_start` BETWEEN sqlc.arg(from_day) AND sqlc.arg(to_day)
  AND NOT EXISTS (
    SELECT 1 FROM `engine_daily_snapshots` s
    WHERE s.`user_id` = p.`user_id` AND s.`day` BETWEEN p.`cycle_start` AND p.`actual_start`
      AND (s.`mode` IN ('pregnancy', 'postpartum') OR (s.`warnings_mask` & sqlc.arg(excluded_mask)) <> 0)
  );

-- name: LatestEngineSnapshotDay :one
-- The newest engine snapshot day on or before to_day (no row when none).
SELECT `day` FROM `engine_daily_snapshots` WHERE `day` <= sqlc.arg(to_day) ORDER BY `day` DESC LIMIT 1;

-- name: CountEngineSnapshotClasses :many
-- Confidence × data_quality of the active users snapshotted on one day (A-06).
SELECT `confidence`, `data_quality`, COUNT(*) AS users FROM `engine_daily_snapshots`
WHERE `day` = sqlc.arg(day)
GROUP BY `confidence`, `data_quality`;

-- name: ListUserInputSnapshots :many
-- requires_user_input episodes: the snapshots in [from_day, to_day] of every user who had a warning of rui_mask in
-- [rui_from_day, rui_to_day], in (user, day) order.
SELECT s.`user_id`, s.`day`, s.`warnings_mask`, s.`engine_rev` FROM `engine_daily_snapshots` s
WHERE s.`day` >= sqlc.arg(from_day) AND s.`day` <= sqlc.arg(to_day)
  AND s.`user_id` IN (
    SELECT r.`user_id` FROM `engine_daily_snapshots` r
    WHERE r.`day` >= sqlc.arg(rui_from_day) AND r.`day` <= sqlc.arg(rui_to_day) AND (r.`warnings_mask` & sqlc.arg(rui_mask)) <> 0
  )
ORDER BY s.`user_id`, s.`day`;

-- name: SumAICost :one
-- AI cost of the lab analysis and the assistant since since_at (USD micros) and the users behind it.
SELECT CAST(COALESCE(SUM(`cost_micros`), 0) AS SIGNED) AS cost_micros, COUNT(DISTINCT `user_id`) AS users
FROM `ai_usage_logs`
WHERE `feature` IN ('lab_analysis', 'assistant') AND `created_at` >= sqlc.arg(since_at);

-- name: SumPlusRevenue :one
-- Plus revenue (IRR, excluding VAT) of the invoices paid in [from_at, to_at) and the paying users.
SELECT CAST(COALESCE(SUM(`subtotal_rials` - `discount_rials`), 0) AS SIGNED) AS revenue_rials, COUNT(DISTINCT `user_id`) AS payers
FROM `plus_invoices`
WHERE `status` = 'paid' AND `paid_at` >= sqlc.arg(from_at) AND `paid_at` < sqlc.arg(to_at);

-- name: GetKPISetting :one
-- An admin-set KPI setting (plus_settings row; e.g. the USD → IRR rate of the AI cost share).
SELECT `value` FROM `plus_settings` WHERE `key` = sqlc.arg(setting_key) LIMIT 1;

-- name: ListMessageVersionCTR :many
-- Message CTR per rule version from the daily rollup (T-M10-06) in [since_day, until_day].
SELECT `rule_id`, `version`,
       CAST(COALESCE(SUM(CASE WHEN `event` = 'impression' THEN `count` END), 0) AS SIGNED) AS shown,
       CAST(COALESCE(SUM(CASE WHEN `event` = 'click' THEN `count` END), 0) AS SIGNED) AS clicked
FROM `message_event_daily`
WHERE `day` BETWEEN sqlc.arg(since_day) AND sqlc.arg(until_day)
GROUP BY `rule_id`, `version`
ORDER BY shown DESC, `rule_id`, `version`;

-- name: GetMetricSnapshotOnOrBefore :one
-- The newest metrics snapshot on or before day (the previous period of the KPI endpoint).
SELECT `day`, `payload`, `computed_at` FROM `metric_snapshots` WHERE `day` <= sqlc.arg(day)
ORDER BY `day` DESC LIMIT 1;

-- name: GetLatestMetricSnapshot :one
SELECT `day`, `payload`, `computed_at` FROM `metric_snapshots` ORDER BY `day` DESC LIMIT 1;

-- name: GetActivityUserTimezone :one
-- The stored zone of a user, read by the activity writer (internal/auth/activity) off the request path.
SELECT `timezone` FROM `user_profiles` WHERE `user_id` = sqlc.arg(user_id) LIMIT 1;
