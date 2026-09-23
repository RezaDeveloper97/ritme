-- Placeholder so the `healthlog` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/healthlog/.

-- name: SampleHealthlogDailyLog :one
SELECT * FROM `daily_health_logs` WHERE id = ? LIMIT 1;
