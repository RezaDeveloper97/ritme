-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportChallengeEnrollments :many
SELECT `id`, `user_id`, `challenge_id`, `started_on`, `status`, `completed_at`, `created_at`, `updated_at`
FROM `challenge_enrollments`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportChallengeTaskCompletions :many
SELECT `id`, `enrollment_id`, `task_id`, `day_number`, `completed_at`
FROM `challenge_task_completions`
WHERE enrollment_id IN (SELECT id FROM `challenge_enrollments` WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportChallengeRewardGrants :many
SELECT `id`, `user_id`, `challenge_id`, `enrollment_id`, `badge_code`, `badge_title`, `plus_days`, `subscription_id`, `granted_at`
FROM `challenge_reward_grants`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportUserChallengeCompletions :many
SELECT `id`, `user_id`, `challenge_id`, `completion_date`, `completed_at`, `created_at`, `updated_at`
FROM `user_challenge_completions`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportUserTaskCompletions :many
SELECT `id`, `user_id`, `task_template_id`, `completion_date`, `completed_at`, `created_at`, `updated_at`
FROM `user_task_completions`
WHERE user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
