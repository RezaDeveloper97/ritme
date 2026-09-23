-- Placeholder so the `profile` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/profile/.

-- name: SampleProfileUserProfile :one
SELECT * FROM `user_profiles` WHERE id = ? LIMIT 1;
