-- Doctor report share links (bloom B-N6-04; internal/sharelinks, D-65). Owner queries are scoped by user_id in the
-- query itself (IDOR); the public read goes by the token's SHA-256 only. `payload` is ciphertext (token-derived key).

-- name: InsertShareLink :execlastid
INSERT INTO `health_share_links` (user_id, token_hash, payload, sections, range_from, range_to, expires_at, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(payload), sqlc.arg(sections), sqlc.arg(range_from), sqlc.arg(range_to),
        sqlc.arg(expires_at), sqlc.arg(now), sqlc.arg(now));

-- name: CountActiveShareLinks :one
SELECT COUNT(*) FROM `health_share_links`
WHERE user_id = sqlc.arg(user_id) AND kind = 'report' AND revoked_at IS NULL AND expires_at > sqlc.arg(now);

-- name: ListShareLinks :many
SELECT id, sections, range_from, range_to, expires_at, revoked_at, view_count, last_viewed_at, created_at
FROM `health_share_links`
WHERE user_id = sqlc.arg(user_id) AND kind = 'report' AND created_at >= sqlc.arg(since)
ORDER BY created_at DESC, id DESC
LIMIT 50;

-- name: GetShareLinkMeta :one
SELECT id, sections, range_from, range_to, expires_at, revoked_at, view_count, last_viewed_at, created_at
FROM `health_share_links`
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND kind = 'report' LIMIT 1;

-- name: RevokeShareLink :execrows
UPDATE `health_share_links`
SET revoked_at = COALESCE(revoked_at, sqlc.arg(now)), payload = NULL, code_payload = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND kind = 'report';

-- name: GetShareLinkByHash :one
SELECT id, payload, expires_at, revoked_at, created_at, range_from, range_to
FROM `health_share_links`
WHERE token_hash = sqlc.arg(token_hash) LIMIT 1;

-- name: CountShareLinkView :exec
UPDATE `health_share_links`
SET view_count = view_count + 1, last_viewed_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: PurgeExpiredShareLinks :execrows
UPDATE `health_share_links`
SET payload = NULL, code_payload = NULL, updated_at = sqlc.arg(now)
WHERE expires_at <= sqlc.arg(now) AND (payload IS NOT NULL OR code_payload IS NOT NULL);

-- name: DeleteStaleShareLinks :execrows
DELETE FROM `health_share_links` WHERE expires_at < sqlc.arg(before);
