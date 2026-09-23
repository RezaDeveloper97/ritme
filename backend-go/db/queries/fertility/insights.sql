-- Fertility insights (T-M5-03, docs/fertility-ttc/README.md). Every query is scoped by user_id.

-- name: ListLHTests :many
-- The user's LH test results from `from` to `to` (inclusive), oldest first.
SELECT log_date, lh_test
FROM `fertility_logs`
WHERE user_id = sqlc.arg(user_id)
  AND log_date >= sqlc.arg(from_date)
  AND log_date <= sqlc.arg(to_date)
  AND lh_test IS NOT NULL
ORDER BY log_date;
