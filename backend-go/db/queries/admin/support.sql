-- Admin support reports inbox (B-N1-12b): «گزارش مشکل» rows from POST /api/v1/support/reports, newest first.
-- The status filter is a LIKE pattern ('%' = all, 'open' / 'resolved' = exact), like the other admin lists.

-- name: CountSupportReports :one
SELECT COUNT(*) FROM `support_reports` WHERE `status` LIKE sqlc.arg(status);

-- name: CountSupportReportsByStatus :many
SELECT `status`, COUNT(*) AS total FROM `support_reports` GROUP BY `status`;

-- name: ListSupportReports :many
SELECT r.id, r.user_id, LEFT(r.message, 160) AS preview, r.screenshot_path IS NOT NULL AS has_screenshot,
       r.app_version, r.status, r.created_at, r.updated_at, u.name AS user_name, u.mobile AS user_mobile
FROM `support_reports` r
JOIN `users` u ON u.id = r.user_id
WHERE r.status LIKE sqlc.arg(status)
ORDER BY r.created_at DESC, r.id DESC
LIMIT ? OFFSET ?;

-- name: GetSupportReport :one
SELECT r.id, r.user_id, r.message, r.screenshot_path, r.app_version, r.user_agent, r.status, r.created_at,
       r.updated_at, u.name AS user_name, u.mobile AS user_mobile
FROM `support_reports` r
JOIN `users` u ON u.id = r.user_id
WHERE r.id = ? LIMIT 1;

-- name: SetSupportReportStatus :execresult
UPDATE `support_reports` SET `status` = sqlc.arg(status), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);
