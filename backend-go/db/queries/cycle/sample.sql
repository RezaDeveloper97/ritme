-- Placeholder so the `cycle` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/cycle/.

-- name: SampleCycleHistory :one
SELECT * FROM `cycle_histories` WHERE id = ? LIMIT 1;
