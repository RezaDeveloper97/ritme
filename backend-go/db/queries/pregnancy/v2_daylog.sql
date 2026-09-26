-- Pregnancy v2 day log (T-M7-03). Every query is scoped by user_id.

-- name: GetV2LastWeight :one
-- The newest weekly log with a weight (value + the day it was logged).
SELECT log_date, pregnancy_week, weight FROM `pregnancy_weekly_logs`
WHERE user_id = ? AND weight IS NOT NULL
ORDER BY log_date DESC, pregnancy_week DESC
LIMIT 1;

-- name: ListV2WeightsRange :many
-- Weighed weekly logs whose log_date is within from..to inclusive (doctor report).
SELECT log_date, pregnancy_week, weight FROM `pregnancy_weekly_logs`
WHERE user_id = sqlc.arg(user_id) AND weight IS NOT NULL
  AND log_date >= sqlc.arg(date_from) AND log_date <= sqlc.arg(date_to)
ORDER BY log_date, pregnancy_week;

-- name: ListV2SymptomLogsRange :many
SELECT * FROM `pregnancy_symptom_logs`
WHERE user_id = sqlc.arg(user_id) AND log_date >= sqlc.arg(date_from) AND log_date <= sqlc.arg(date_to)
ORDER BY log_date;
