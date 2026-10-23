-- User reference codes (migration 00105, T-M10-25; SPEC v6 §5.2, DECISIONS.md A-09: `RT-` + 6 base32 chars of an HMAC
-- of the id). Stored for search; the writer fills it, a backfill pages through the users without one.

-- name: GetUserIDByRefCode :one
SELECT id FROM `users` WHERE ref_code = ? LIMIT 1;

-- name: SetUserRefCode :execrows
UPDATE `users` SET ref_code = sqlc.arg(ref_code) WHERE id = sqlc.arg(id) AND ref_code IS NULL;

-- name: ListUserIDsWithoutRefCode :many
SELECT id FROM `users` WHERE ref_code IS NULL AND id > sqlc.arg(after_id) ORDER BY id LIMIT ?;
