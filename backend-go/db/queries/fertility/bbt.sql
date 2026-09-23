-- BBT chart (T-M5-02, docs/fertility-ttc/README.md). Every query is scoped by user_id.

-- name: ListBBTReadings :many
-- The user's basal-temperature readings from `from` to `to` (inclusive), oldest first.
-- The value is decimal(4,2) text ("36.55").
SELECT log_date, basal_body_temperature
FROM `daily_health_logs`
WHERE user_id = sqlc.arg(user_id)
  AND log_date >= sqlc.arg(from_date)
  AND log_date <= sqlc.arg(to_date)
  AND basal_body_temperature IS NOT NULL
ORDER BY log_date;
