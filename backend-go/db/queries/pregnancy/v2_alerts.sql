-- Pregnancy v2 alert rules (T-M7-04, internal/messages/pregnancyalerts). v2 rows live in pregnancy_alerts
-- with alert_type = 'v2:<rule_key>' and their v2 metadata (level4, dedupe key, placeholder values) in
-- trigger_symptoms. Every user-scoped query filters by user_id.

-- name: ListLiveMessageGroup :many
-- Every live (active + approved) row of one message_contents group, all locales (rules + legend).
SELECT item_key, locale, payload FROM `message_contents`
WHERE `group` = sqlc.arg(message_group) AND is_active = 1 AND is_approved = 1
ORDER BY item_key, locale;

-- name: ListV2AlertsSince :many
-- The user's v2 alerts created at or after `since`, newest first (dismissed = acked ones included).
SELECT * FROM `pregnancy_alerts`
WHERE user_id = sqlc.arg(user_id) AND alert_type LIKE 'v2:%' AND created_at >= sqlc.arg(since)
ORDER BY created_at DESC, id DESC;

-- name: CountV2AlertDedupe :one
-- Dedupe: how many alerts of one rule with the same dedupe key exist since `since`.
SELECT COUNT(*) FROM `pregnancy_alerts`
WHERE user_id = sqlc.arg(user_id) AND alert_type = sqlc.arg(alert_type) AND created_at >= sqlc.arg(since)
  AND JSON_UNQUOTE(JSON_EXTRACT(trigger_symptoms, '$.dedupe')) = CAST(sqlc.arg(dedupe) AS CHAR);

-- name: GetV2Alert :one
SELECT * FROM `pregnancy_alerts`
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id) AND alert_type LIKE 'v2:%' LIMIT 1;

-- name: AckV2Alert :exec
-- Action ack («دیدم، ممنون»): read + dismissed (hidden from the v1 active list too).
UPDATE `pregnancy_alerts`
SET is_read = 1, read_at = COALESCE(read_at, sqlc.arg(now)), is_dismissed = 1,
    dismissed_at = COALESCE(dismissed_at, sqlc.arg(now)), updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id);

-- name: MarkV2AlertRead :exec
UPDATE `pregnancy_alerts` SET is_read = 1, read_at = COALESCE(read_at, sqlc.arg(now)), updated_at = sqlc.arg(now)
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(id);

-- name: ListV2WeeklyLogsSince :many
-- Weekly logs (weight / BP / sugar) logged on or after `date_from`.
SELECT * FROM `pregnancy_weekly_logs`
WHERE user_id = sqlc.arg(user_id) AND log_date >= sqlc.arg(date_from)
ORDER BY log_date, pregnancy_week;
