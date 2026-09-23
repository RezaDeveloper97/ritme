-- PregnancyAlert (App\Models\PregnancyAlert). Scopes: active = is_dismissed 0, unread = is_read 0.
-- Ordering mirrors Laravel's orderBy('created_at', 'desc') (no tie-breaker).

-- name: ListActiveAlerts :many
SELECT * FROM `pregnancy_alerts`
WHERE user_id = sqlc.arg(user_id) AND is_dismissed = 0
  AND (sqlc.arg(level) = '' OR alert_level = sqlc.arg(level))
  AND (sqlc.arg(unread_only) = 0 OR is_read = 0)
ORDER BY created_at DESC;

-- name: CountActiveAlerts :one
SELECT
  COUNT(*) AS total,
  CAST(COALESCE(SUM(is_read = 0), 0) AS SIGNED) AS unread,
  CAST(COALESCE(SUM(alert_level = 'emergency'), 0) AS SIGNED) AS emergency,
  CAST(COALESCE(SUM(alert_level = 'emergency' AND is_read = 0), 0) AS SIGNED) AS unread_emergency,
  CAST(COALESCE(SUM(alert_level = 'warning'), 0) AS SIGNED) AS warning,
  CAST(COALESCE(SUM(alert_level = 'info'), 0) AS SIGNED) AS info
FROM `pregnancy_alerts` WHERE user_id = ? AND is_dismissed = 0;

-- name: LatestUnreadEmergencyAlert :one
SELECT * FROM `pregnancy_alerts`
WHERE user_id = ? AND alert_level = 'emergency' AND is_read = 0 AND is_dismissed = 0
ORDER BY created_at DESC LIMIT 1;

-- name: GetAlert :one
SELECT * FROM `pregnancy_alerts` WHERE user_id = ? AND id = ? LIMIT 1;

-- name: InsertAlert :execlastid
INSERT INTO `pregnancy_alerts` (
  user_id, alert_level, alert_type, title, message, pregnancy_week, trigger_symptoms, medical_history_flags,
  recommended_actions, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: MarkAlertRead :exec
UPDATE `pregnancy_alerts` SET is_read = 1, read_at = ?, updated_at = ? WHERE id = ?;

-- name: DismissAlert :exec
UPDATE `pregnancy_alerts` SET is_dismissed = 1, dismissed_at = ?, updated_at = ? WHERE id = ?;

-- name: MarkAllAlertsRead :execrows
UPDATE `pregnancy_alerts` SET is_read = 1, read_at = ?, updated_at = ? WHERE user_id = ? AND is_read = 0;
