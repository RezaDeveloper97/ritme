-- Placeholder so the `home` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/home/.

-- name: SampleHomeTaskTemplate :one
SELECT * FROM `task_templates` WHERE id = ? LIMIT 1;
