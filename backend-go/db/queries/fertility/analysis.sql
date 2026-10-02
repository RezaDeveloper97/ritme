-- TTC analysis (B-N3-11, internal/analysis ttc.go). Every query is scoped by user_id.

-- name: ListFertilitySignals :many
-- The user's fertility_logs LH tests and cervical mucus from `from` to `to` (inclusive), oldest first.
-- (QUESTIONS #80: fertility_logs is not synced into health_log_entries; the TTC analysis merges both.)
SELECT log_date, lh_test, cervical_mucus
FROM `fertility_logs`
WHERE user_id = sqlc.arg(user_id)
  AND log_date >= sqlc.arg(from_date)
  AND log_date <= sqlc.arg(to_date)
  AND (lh_test IS NOT NULL OR cervical_mucus IS NOT NULL)
ORDER BY log_date;

-- name: FirstFertilityLogDate :many
-- The user's first fertility_logs day (no row when none).
SELECT log_date FROM `fertility_logs` WHERE user_id = sqlc.arg(user_id) ORDER BY log_date LIMIT 1;

-- name: FirstFertilityEntryDate :many
-- The user's first taxonomy v2 day with a TTC measurement (BBT, LH or pregnancy test; legacy BBT is
-- synced into these rows). No row when none.
SELECT log_date
FROM `health_log_entries`
WHERE user_id = sqlc.arg(user_id)
  AND category = 'measurements'
  AND param IN ('bbt', 'lh_test', 'pregnancy_test')
ORDER BY log_date
LIMIT 1;
