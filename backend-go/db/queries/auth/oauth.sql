-- Passport tables (oauth_access_tokens, oauth_clients), shared with Laravel.

-- name: GetAccessToken :one
SELECT id, user_id, client_id, revoked, expires_at FROM `oauth_access_tokens` WHERE id = ? LIMIT 1;

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

-- name: ListPersonalAccessClientCandidates :many
-- ClientRepository::personalAccessClient('users'): non-revoked, provider NULL or 'users',
-- latest first; the caller picks the first whose grant_types contains "personal_access".
SELECT id, grant_types FROM `oauth_clients`
WHERE revoked = 0 AND (provider IS NULL OR provider = 'users')
ORDER BY created_at DESC;
