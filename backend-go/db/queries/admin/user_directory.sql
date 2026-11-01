-- Admin user directory projection and saved segments (migration 00105, T-M10-25; SPEC v6 §5.2, DECISIONS.md A-08).
-- admin_user_activity is rebuilt nightly (T-M10-32); the list endpoint filters on its indexed columns.

-- name: UpsertAdminUserActivity :exec
INSERT INTO `admin_user_activity` (`user_id`, `last_activity_at`, `log_days_30`, `log_days_recent14`, `log_days_prior14`,
                                   `churn`, `mode`, `plan`, `data_quality`, `main_warning`, `updated_on`)
VALUES (sqlc.arg(user_id), sqlc.narg(last_activity_at), sqlc.arg(log_days_30), sqlc.arg(log_days_recent14),
        sqlc.arg(log_days_prior14), sqlc.arg(churn), sqlc.arg(mode), sqlc.arg(plan), sqlc.narg(data_quality),
        sqlc.narg(main_warning), sqlc.arg(updated_on))
ON DUPLICATE KEY UPDATE `last_activity_at` = VALUES(`last_activity_at`), `log_days_30` = VALUES(`log_days_30`),
  `log_days_recent14` = VALUES(`log_days_recent14`), `log_days_prior14` = VALUES(`log_days_prior14`),
  `churn` = VALUES(`churn`), `mode` = VALUES(`mode`), `plan` = VALUES(`plan`), `data_quality` = VALUES(`data_quality`),
  `main_warning` = VALUES(`main_warning`), `updated_on` = VALUES(`updated_on`);

-- name: GetAdminUserActivity :one
SELECT * FROM `admin_user_activity` WHERE `user_id` = ? LIMIT 1;

-- name: ListAdminUserActivityByIDs :many
SELECT * FROM `admin_user_activity` WHERE `user_id` IN (sqlc.slice(user_ids));

-- name: DeleteStaleAdminUserActivity :execrows
-- Rows the nightly rebuild did not touch (users no longer in scope).
DELETE FROM `admin_user_activity` WHERE `updated_on` < sqlc.arg(before_day);

-- ---------------------------------------------------------------------------
-- Saved segments

-- name: ListAdminSegments :many
SELECT * FROM `admin_segments` ORDER BY `name`;

-- name: GetAdminSegment :one
SELECT * FROM `admin_segments` WHERE `id` = ? LIMIT 1;

-- name: CreateAdminSegment :execlastid
INSERT INTO `admin_segments` (`name`, `filters`, `created_by`, `created_at`, `updated_at`)
VALUES (sqlc.arg(name), sqlc.arg(filters), sqlc.narg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: UpdateAdminSegment :execrows
UPDATE `admin_segments` SET `name` = sqlc.arg(name), `filters` = sqlc.arg(filters), `updated_at` = sqlc.arg(now)
WHERE `id` = sqlc.arg(id);

-- name: DeleteAdminSegment :execrows
DELETE FROM `admin_segments` WHERE `id` = ?;

-- ---------------------------------------------------------------------------
-- T-M10-32 (D-181; SPEC v6 §5.2, A-08, A-09): the nightly projection writer, the directory read and the user-detail
-- engine card. Nothing here returns a health value: the projection carries codes (mode, plan, churn, data_quality,
-- the main warning) and counts of logged days, never what was logged.

-- name: ListAdminActivitySources :many
-- One page of users (id > after_id) with what the projection row needs, in one query: signup, the stored life mode,
-- the Plus tier at now (a covering active / canceled subscription → plus; else a running trial → trial — plus.tierOf),
-- the newest activity day, logged days in [from30, to] / [recent_from, to] / [prior_from, prior_to] from
-- user_activity_days (KEY user_id, day), and the newest engine snapshot on or after snapshot_since (today's row on the
-- user's own day; yesterday's covers a zone behind Tehran).
SELECT
  u.id AS user_id,
  u.created_at,
  lp.life_mode,
  CAST(EXISTS (
    SELECT 1 FROM `plus_subscriptions` s
    WHERE s.user_id = u.id AND s.status IN ('active', 'canceled')
      AND s.starts_at <= sqlc.arg(now) AND s.ends_at > sqlc.arg(now)
  ) AS UNSIGNED) AS plus_active,
  CAST(EXISTS (
    SELECT 1 FROM `plus_trials` t
    WHERE t.user_id = u.id AND t.started_at <= sqlc.arg(now) AND t.ends_at > sqlc.arg(now)
  ) AS UNSIGNED) AS trial_running,
  (SELECT MAX(d.day) FROM `user_activity_days` d WHERE d.user_id = u.id AND d.day <= sqlc.arg(to_day)) AS last_day,
  (SELECT CAST(COUNT(*) AS SIGNED) FROM `user_activity_days` d WHERE d.user_id = u.id AND d.logged = 1
    AND d.day >= sqlc.arg(from30) AND d.day <= sqlc.arg(to_day)) AS log_days_30,
  (SELECT CAST(COUNT(*) AS SIGNED) FROM `user_activity_days` d WHERE d.user_id = u.id AND d.logged = 1
    AND d.day >= sqlc.arg(recent_from) AND d.day <= sqlc.arg(to_day)) AS log_days_recent14,
  (SELECT CAST(COUNT(*) AS SIGNED) FROM `user_activity_days` d WHERE d.user_id = u.id AND d.logged = 1
    AND d.day >= sqlc.arg(prior_from) AND d.day <= sqlc.arg(prior_to)) AS log_days_prior14,
  es.mode AS snapshot_mode,
  es.data_quality AS snapshot_data_quality,
  es.warnings_mask AS snapshot_warnings_mask
FROM `users` u
LEFT JOIN `user_life_profiles` lp ON lp.user_id = u.id
LEFT JOIN `engine_daily_snapshots` es ON es.user_id = u.id AND es.day = (
  SELECT MAX(s2.day) FROM `engine_daily_snapshots` s2 WHERE s2.user_id = u.id AND s2.day >= sqlc.arg(snapshot_since)
)
WHERE u.id > sqlc.arg(after_id)
ORDER BY u.id
LIMIT ?;

-- name: CountAdminDirectory :one
-- The directory filters (segments.Filters): every one is optional (NULL = no filter) and reads an indexed column of
-- users or admin_user_activity; no aggregate per request. pattern = a LIKE pattern over name / mobile / email,
-- ref_code = an exact reference code (A-09); the status filter is CountUsers' blocked_at range (NULL = the 1971-01-01
-- sentinel, users.statusRange).
SELECT COUNT(*) FROM `users` u
LEFT JOIN `admin_user_activity` a ON a.user_id = u.id
WHERE (sqlc.narg(pattern) IS NULL OR IFNULL(u.name, '') LIKE sqlc.narg(pattern) OR IFNULL(u.mobile, '') LIKE sqlc.narg(pattern)
       OR IFNULL(u.email, '') LIKE sqlc.narg(pattern))
  AND (sqlc.narg(ref_code) IS NULL OR u.ref_code = sqlc.narg(ref_code))
  AND COALESCE(u.blocked_at, CAST('1971-01-01 00:00:00' AS DATETIME)) >= sqlc.arg(blocked_from)
  AND COALESCE(u.blocked_at, CAST('1971-01-01 00:00:00' AS DATETIME)) <= sqlc.arg(blocked_to)
  AND (sqlc.narg(created_from) IS NULL OR u.created_at >= sqlc.narg(created_from))
  AND (sqlc.narg(created_to) IS NULL OR u.created_at < sqlc.narg(created_to))
  AND (sqlc.narg(plan) IS NULL OR a.plan = sqlc.narg(plan))
  AND (sqlc.narg(churn) IS NULL OR a.churn = sqlc.narg(churn))
  AND (sqlc.narg(mode) IS NULL OR a.mode = sqlc.narg(mode))
  AND (sqlc.narg(data_quality) IS NULL OR a.data_quality = sqlc.narg(data_quality))
  AND (sqlc.narg(main_warning) IS NULL OR a.main_warning = sqlc.narg(main_warning))
  AND (sqlc.narg(active_since) IS NULL OR a.last_activity_at >= sqlc.narg(active_since))
  AND (sqlc.narg(inactive_before) IS NULL OR a.last_activity_at IS NULL OR a.last_activity_at < sqlc.narg(inactive_before));

-- name: ListAdminDirectory :many
-- CountAdminDirectory's filters, one page newest signup first (KEY users.created_at).
SELECT u.id, u.ref_code, u.mobile, u.blocked_at, u.created_at,
       a.user_id AS projected_user_id, a.last_activity_at, a.log_days_30, a.churn, a.mode, a.plan, a.data_quality,
       a.main_warning
FROM `users` u
LEFT JOIN `admin_user_activity` a ON a.user_id = u.id
WHERE (sqlc.narg(pattern) IS NULL OR IFNULL(u.name, '') LIKE sqlc.narg(pattern) OR IFNULL(u.mobile, '') LIKE sqlc.narg(pattern)
       OR IFNULL(u.email, '') LIKE sqlc.narg(pattern))
  AND (sqlc.narg(ref_code) IS NULL OR u.ref_code = sqlc.narg(ref_code))
  AND COALESCE(u.blocked_at, CAST('1971-01-01 00:00:00' AS DATETIME)) >= sqlc.arg(blocked_from)
  AND COALESCE(u.blocked_at, CAST('1971-01-01 00:00:00' AS DATETIME)) <= sqlc.arg(blocked_to)
  AND (sqlc.narg(created_from) IS NULL OR u.created_at >= sqlc.narg(created_from))
  AND (sqlc.narg(created_to) IS NULL OR u.created_at < sqlc.narg(created_to))
  AND (sqlc.narg(plan) IS NULL OR a.plan = sqlc.narg(plan))
  AND (sqlc.narg(churn) IS NULL OR a.churn = sqlc.narg(churn))
  AND (sqlc.narg(mode) IS NULL OR a.mode = sqlc.narg(mode))
  AND (sqlc.narg(data_quality) IS NULL OR a.data_quality = sqlc.narg(data_quality))
  AND (sqlc.narg(main_warning) IS NULL OR a.main_warning = sqlc.narg(main_warning))
  AND (sqlc.narg(active_since) IS NULL OR a.last_activity_at >= sqlc.narg(active_since))
  AND (sqlc.narg(inactive_before) IS NULL OR a.last_activity_at IS NULL OR a.last_activity_at < sqlc.narg(inactive_before))
ORDER BY u.created_at DESC, u.id DESC
LIMIT ? OFFSET ?;

-- name: GetSegmentSubject :one
-- The user's directory row for the `segment` message fact: the projection (NULL columns when the nightly job has not
-- run for her yet) plus the account columns the filters read.
SELECT u.id, u.blocked_at, u.created_at,
       a.user_id AS projected_user_id, a.last_activity_at, a.churn, a.mode, a.plan, a.data_quality, a.main_warning
FROM `users` u
LEFT JOIN `admin_user_activity` a ON a.user_id = u.id
WHERE u.id = ?
LIMIT 1;

-- ---------------------------------------------------------------------------
-- User detail (GET /users/:id/overview). Every query is scoped by user_id.

-- name: GetOverviewAccount :one
SELECT u.created_at, u.mobile_verified_at, u.blocked_at, u.updated_at, u.ref_code, p.user_goal,
       (SELECT CAST(COUNT(*) AS SIGNED) FROM `companions` c WHERE c.owner_id = u.id AND c.status = 'active') AS companions,
       (SELECT CAST(COUNT(DISTINCT lpg.course_id) AS SIGNED) FROM `learning_progress` lpg WHERE lpg.user_id = u.id) AS courses
FROM `users` u
LEFT JOIN `user_profiles` p ON p.id = (SELECT MIN(p2.id) FROM `user_profiles` p2 WHERE p2.user_id = u.id)
WHERE u.id = ?
LIMIT 1;

-- name: GetLatestEngineSnapshot :one
-- Today's engine card: the newest snapshot on or before today (codes only).
SELECT `day`, main_phase, subphase, confidence, data_quality, warnings_mask, days_late, `mode`
FROM `engine_daily_snapshots`
WHERE user_id = sqlc.arg(user_id) AND `day` <= sqlc.arg(today)
ORDER BY `day` DESC
LIMIT 1;

-- name: ListUserPeriodStarts :many
-- The newest logged (not estimated) period starts; the caller turns them into cycle lengths and drops the dates.
SELECT period_start_date FROM `cycle_histories`
WHERE user_id = ? AND is_estimated = 0
ORDER BY period_start_date DESC
LIMIT ?;

-- name: ListOverviewSubscriptions :many
SELECT s.status, s.source, s.starts_at, s.ends_at, s.auto_renew,
       pl.title AS plan_title
FROM `plus_subscriptions` s
LEFT JOIN `plus_plans` pl ON pl.id = s.plan_id
WHERE s.user_id = ?
ORDER BY s.id DESC
LIMIT ?;

-- name: ListOverviewConsents :many
SELECT consent, granted, version, granted_at, revoked_at FROM `user_consents`
WHERE user_id = ?
ORDER BY consent
LIMIT ?;

-- name: ListOverviewSupportReports :many
-- Never the report text: it is free-form and may hold health data.
SELECT id, status, app_version, created_at FROM `support_reports`
WHERE user_id = ?
ORDER BY id DESC
LIMIT ?;

-- name: ListOverviewChallenges :many
SELECT e.challenge_id, ch.title, e.status, e.started_on
FROM `challenge_enrollments` e
JOIN `challenges` ch ON ch.id = e.challenge_id
WHERE e.user_id = ?
ORDER BY e.id DESC
LIMIT ?;

-- name: ListOverviewMessages :many
-- Messages the user was shown, with the rule that triggered them (id, key, kind, title) and the version she saw.
SELECT e.rule_id, r.rule_key, r.kind, r.title, r.placement, e.slot, e.version, e.variant, e.event, e.created_at
FROM `message_events` e
JOIN `message_rules` r ON r.id = e.rule_id
WHERE e.user_id = ? AND e.event <> 'eligible'
ORDER BY e.id DESC
LIMIT ?;
