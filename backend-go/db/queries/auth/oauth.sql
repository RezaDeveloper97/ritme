-- Passport tables (oauth_access_tokens, oauth_clients), shared with Laravel.

-- name: GetAccessToken :one
SELECT id, user_id, client_id, revoked, expires_at FROM `oauth_access_tokens` WHERE id = ? LIMIT 1;

-- name: GetAccessTokenWithUser :one
-- The auth hot path (T-M9-09): the token row (id = jti) and the user (id = the JWT `sub`) in ONE round-trip,
-- replacing GetAccessToken + GetUserByID. Both are primary-key lookups, so the join is two const reads. No row means
-- the token or the user is missing: the verifier then falls back to the separate reads to answer the exact 401 code.
-- Never cached (DECISIONS.md X-02): revocation must be read fresh on every request.
SELECT t.id AS token_id, t.user_id AS token_user_id, t.client_id AS token_client_id, t.revoked AS token_revoked,
       t.expires_at AS token_expires_at, sqlc.embed(u)
FROM `oauth_access_tokens` t
CROSS JOIN `users` u
WHERE t.id = sqlc.arg(token_id) AND u.id = sqlc.arg(user_id)
LIMIT 1;

-- name: InsertAccessToken :exec
-- AccessTokenRepository::persistNewAccessToken + PersonalAccessTokenFactory (name, scopes '[]').
INSERT INTO `oauth_access_tokens` (id, user_id, client_id, name, scopes, revoked, created_at, updated_at, expires_at)
VALUES (?, ?, ?, 'auth_token', '[]', 0, ?, ?, ?);

-- name: RevokeAccessToken :execrows
-- AccessToken::revoke() (Eloquent update also touches updated_at).
UPDATE `oauth_access_tokens` SET revoked = 1, updated_at = ? WHERE id = ?;

-- name: RevokeUserAccessTokens :execrows
-- $user->tokens()->update(['revoked' => true]) — admin block / account deletion.
UPDATE `oauth_access_tokens` SET revoked = 1, updated_at = ? WHERE user_id = ? AND revoked = 0;

-- name: GetClient :one
-- ClientRepository::find($aud) (findActive checks `revoked`, the guard checks `provider`).
SELECT id, provider, revoked FROM `oauth_clients` WHERE id = ? LIMIT 1;

-- name: ListClients :many
-- Every client for the verifier's in-process cache (T-M9-09): loaded on first use, refreshed every 5 min, reloaded
-- once on an unknown id. The table holds a handful of rows.
SELECT id, provider, revoked FROM `oauth_clients`;

-- name: ListPersonalAccessClientCandidates :many
-- ClientRepository::personalAccessClient('users'): non-revoked, provider NULL or 'users',
-- latest first; the caller picks the first whose grant_types contains "personal_access".
SELECT id, grant_types FROM `oauth_clients`
WHERE revoked = 0 AND (provider IS NULL OR provider = 'users')
ORDER BY created_at DESC;
