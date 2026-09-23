-- Placeholder so the `messages` sqlc package exists and compiles before its domain task lands (T-M2-03).
-- Delete this file when the first real query is added to db/queries/messages/.

-- name: SampleMessagesContent :one
SELECT * FROM `message_contents` WHERE id = ? LIMIT 1;
