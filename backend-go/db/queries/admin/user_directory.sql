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
