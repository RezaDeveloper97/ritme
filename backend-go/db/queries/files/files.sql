-- Generic encrypted file storage (CB-CORE-05; internal/files). Owner-facing queries are scoped by user_id in the query
-- itself (IDOR). GetFile reads by id only: it serves the signed-URL and public downloads, which prove their right to
-- the row with an HMAC over the row's owner and path, or with the public purpose + the random file name.

-- name: CreateFile :execlastid
INSERT INTO `files` (user_id, purpose, visibility, mime, size_bytes, sha256, path, created_at, updated_at)
VALUES (sqlc.narg(user_id), sqlc.arg(purpose), sqlc.arg(visibility), sqlc.arg(mime), sqlc.arg(size_bytes),
        sqlc.arg(sha256), sqlc.arg(path), sqlc.arg(now), sqlc.arg(now));

-- name: GetFile :one
SELECT * FROM `files` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: GetOwnedFile :one
SELECT * FROM `files` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) LIMIT 1;

-- name: GetPlatformFile :one
SELECT * FROM `files` WHERE id = sqlc.arg(id) AND user_id IS NULL LIMIT 1;

-- name: DeleteOwnedFile :execrows
DELETE FROM `files` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: DeletePlatformFile :execrows
DELETE FROM `files` WHERE id = sqlc.arg(id) AND user_id IS NULL;

-- name: LockOwner :one
-- First statement of a user's upload transaction: the user row lock serialises that user's uploads, so the usage
-- reads below never race (two gap locks on an empty range would otherwise deadlock on insert).
SELECT id FROM `users` WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: LockOwnerUsage :one
-- Inside the upload transaction: the owner's files of one purpose, locked (next-key locks on the (user_id, purpose)
-- range) so two concurrent uploads cannot both pass the quota.
SELECT COUNT(*) AS files, CAST(COALESCE(SUM(size_bytes), 0) AS UNSIGNED) AS bytes
FROM `files` WHERE user_id = sqlc.arg(user_id) AND purpose = sqlc.arg(purpose) FOR UPDATE;

-- name: ListOwnerPaths :many
-- The orphan sweep: the stored paths of one owner and purpose (blobs on disk without one are removed).
SELECT path FROM `files` WHERE user_id = sqlc.arg(user_id) AND purpose = sqlc.arg(purpose);

-- name: ListPlatformPaths :many
SELECT path FROM `files` WHERE user_id IS NULL AND purpose = sqlc.arg(purpose);

-- name: LockPlatformUsage :one
SELECT COUNT(*) AS files, CAST(COALESCE(SUM(size_bytes), 0) AS UNSIGNED) AS bytes
FROM `files` WHERE user_id IS NULL AND purpose = sqlc.arg(purpose) FOR UPDATE;
