-- Pregnancy alert rule inputs (internal/messages/pregnancyalerts): the legend / contact rows of message_contents and the
-- weekly logs the detectors read. The alerts themselves are message_inbox rows since T-M10-08 (D-165; alerts.sql).
-- Every user-scoped query filters by user_id.

-- name: ListLiveMessageGroup :many
-- Every live (active + approved) row of one message_contents group, all locales (rules + legend).
SELECT item_key, locale, payload FROM `message_contents`
WHERE `group` = sqlc.arg(message_group) AND is_active = 1 AND is_approved = 1
ORDER BY item_key, locale;

-- name: ListV2WeeklyLogsSince :many
-- Weekly logs (weight / BP / sugar) logged on or after `date_from`.
SELECT * FROM `pregnancy_weekly_logs`
WHERE user_id = sqlc.arg(user_id) AND log_date >= sqlc.arg(date_from)
ORDER BY log_date, pregnancy_week;
