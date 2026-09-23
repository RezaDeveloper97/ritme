-- Placeholder so the `admin` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/admin/.

-- name: SampleAdminAdmin :one
SELECT * FROM `admins` WHERE id = ? LIMIT 1;
