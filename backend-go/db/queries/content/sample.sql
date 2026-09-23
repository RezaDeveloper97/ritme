-- Placeholder so the `content` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/content/.

-- name: SampleContentArticle :one
SELECT * FROM `articles` WHERE id = ? LIMIT 1;
