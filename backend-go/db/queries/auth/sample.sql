-- Placeholder so the `auth` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/auth/.

-- name: SampleAuthUser :one
SELECT * FROM `users` WHERE id = ? LIMIT 1;
