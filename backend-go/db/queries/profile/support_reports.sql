-- Support reports (B-N1-12, goose 00012): «گزارش مشکل» from the Support screen, Go only.

-- name: CreateSupportReport :execlastid
INSERT INTO `support_reports` (user_id, message, screenshot_path, app_version, user_agent, status, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(message), sqlc.narg(screenshot_path), sqlc.narg(app_version),
        sqlc.narg(user_agent), 'open', sqlc.arg(now), sqlc.arg(now));

-- name: CountRecentSupportReports :one
-- Per-user flood guard: reports created since `since`.
SELECT COUNT(*) FROM `support_reports` WHERE user_id = ? AND created_at >= ?;

-- name: ListUserSupportScreenshots :many
-- Screenshot files to remove with the account (the rows go by ON DELETE CASCADE).
SELECT screenshot_path FROM `support_reports` WHERE user_id = ? AND screenshot_path IS NOT NULL;
