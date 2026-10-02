-- Fertility insights (T-M5-03, docs/fertility-ttc/README.md). Every query is scoped by user_id.

-- name: ListLHTests :many
-- The user's LH test results from `from` to `to` (inclusive), oldest first: the /fertility log's
-- (fertility_logs) and the log sheet's (health_log_entries measurements.lh_test, B-N3-14b); a day
-- logged in both comes twice and the service keeps the stronger result.
SELECT f.log_date, f.lh_test
FROM `fertility_logs` f
WHERE f.user_id = sqlc.arg(user_id)
  AND f.log_date >= sqlc.arg(from_date)
  AND f.log_date <= sqlc.arg(to_date)
  AND f.lh_test IS NOT NULL
UNION ALL
SELECT e.log_date, e.value_code AS lh_test
FROM `health_log_entries` e
WHERE e.user_id = sqlc.arg(user_id)
  AND e.category = 'measurements'
  AND e.param = 'lh_test'
  AND e.log_date >= sqlc.arg(from_date)
  AND e.log_date <= sqlc.arg(to_date)
  AND e.value_code IS NOT NULL
ORDER BY log_date;
