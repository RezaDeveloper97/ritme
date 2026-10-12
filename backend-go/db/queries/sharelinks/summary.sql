-- Record sharing (canvas-build CB-REC-03, D-71): the 24h summary links with a short code (kind = 'summary') and the
-- access log of every link kind. Owner queries are scoped by user_id in the query itself (IDOR); the public reads go
-- by the token's SHA-256 or the code's HMAC only. `payload` / `code_payload` are ciphertext.

-- name: InsertSummaryLink :execlastid
INSERT INTO `health_share_links` (user_id, kind, label, token_hash, code_hash, code_payload, payload, sections,
                                  include_losses, range_from, range_to, expires_at, created_at, updated_at)
VALUES (sqlc.arg(user_id), 'summary', sqlc.arg(label), sqlc.arg(token_hash), sqlc.arg(code_hash), sqlc.arg(code_payload),
        sqlc.arg(payload), sqlc.arg(sections), sqlc.arg(include_losses), sqlc.arg(range_from), sqlc.arg(range_to),
        sqlc.arg(expires_at), sqlc.arg(now), sqlc.arg(now));

-- name: CountActiveSummaryLinks :one
SELECT COUNT(*) FROM `health_share_links`
WHERE user_id = sqlc.arg(user_id) AND kind = 'summary' AND revoked_at IS NULL AND expires_at > sqlc.arg(now);

-- name: ListSummaryLinks :many
SELECT id, label, sections, include_losses, range_from, range_to, expires_at, revoked_at, view_count, last_viewed_at, created_at
FROM `health_share_links`
WHERE user_id = sqlc.arg(user_id) AND kind = 'summary' AND created_at >= sqlc.arg(since)
ORDER BY created_at DESC, id DESC
LIMIT 50;

-- name: GetSummaryLinkMeta :one
SELECT id, label, sections, include_losses, range_from, range_to, expires_at, revoked_at, view_count, last_viewed_at, created_at
FROM `health_share_links`
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND kind = 'summary' LIMIT 1;

-- name: RevokeSummaryLink :execrows
UPDATE `health_share_links`
SET revoked_at = COALESCE(revoked_at, sqlc.arg(now)), payload = NULL, code_payload = NULL, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND kind = 'summary';

-- name: GetShareLinkForOpen :one
-- include_losses: the public read re-applies the owner's per-share loss choice to the decrypted snapshot.
SELECT id, user_id, kind, payload, include_losses, expires_at, revoked_at
FROM `health_share_links`
WHERE token_hash = sqlc.arg(token_hash) LIMIT 1;

-- name: GetShareLinkByCode :one
SELECT id, user_id, code_payload, expires_at, revoked_at
FROM `health_share_links`
WHERE code_hash = sqlc.arg(code_hash) AND kind = 'summary' LIMIT 1;

-- name: RecentShareLinkView :one
-- CB-REC-06 (L3): the same viewer (via, device, browser) already logged on this link since `since` — a reload or a
-- re-render then neither counts nor logs again.
SELECT EXISTS (
  SELECT 1 FROM `health_share_link_views`
  WHERE share_link_id = sqlc.arg(share_link_id) AND via = sqlc.arg(via) AND device = sqlc.arg(device)
    AND browser = sqlc.arg(browser) AND viewed_at > sqlc.arg(since)
) AS seen;

-- name: InsertShareLinkView :exec
INSERT INTO `health_share_link_views` (share_link_id, user_id, via, device, browser, viewed_at)
VALUES (sqlc.arg(share_link_id), sqlc.arg(user_id), sqlc.arg(via), sqlc.arg(device), sqlc.arg(browser), sqlc.arg(viewed_at));

-- name: ListShareAccess :many
-- The owner's access log «سابقه دسترسی», newest first, with the link it went through.
SELECT v.id, v.share_link_id, v.via, v.device, v.browser, v.viewed_at, l.kind, l.label, l.expires_at, l.revoked_at
FROM `health_share_link_views` v
JOIN `health_share_links` l ON l.id = v.share_link_id AND l.user_id = v.user_id
WHERE v.user_id = sqlc.arg(user_id)
ORDER BY v.viewed_at DESC, v.id DESC
LIMIT ?;

-- name: ListShareLinkAccess :many
-- The access log of one link of the owner.
SELECT v.id, v.share_link_id, v.via, v.device, v.browser, v.viewed_at, l.kind, l.label, l.expires_at, l.revoked_at
FROM `health_share_link_views` v
JOIN `health_share_links` l ON l.id = v.share_link_id AND l.user_id = v.user_id
WHERE v.user_id = sqlc.arg(user_id) AND v.share_link_id = sqlc.arg(share_link_id)
ORDER BY v.viewed_at DESC, v.id DESC
LIMIT ?;

-- name: ShareLinkOwned :one
-- Whether the link (either kind) belongs to the owner.
SELECT COUNT(*) FROM `health_share_links` WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);
