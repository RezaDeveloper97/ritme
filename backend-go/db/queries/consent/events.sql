-- The consent ledger (T-M10-23, goose 00101): append-only `consent_events`; `user_consents` is its current-state
-- projection (consents.sql). There is deliberately no UPDATE or DELETE here — the table's triggers refuse both, and
-- only a user delete (FK cascade) removes rows. Always scoped by user_id.

-- name: LockConsentOwner :one
-- Serialises one user's consent writes: the event and the projection row change together under this lock.
SELECT id FROM `users` WHERE id = ? FOR UPDATE;

-- name: InsertConsentEvent :exec
INSERT INTO `consent_events` (user_id, consent, version, action, scope, source, created_at)
VALUES (sqlc.arg(user_id), sqlc.arg(consent), sqlc.arg(version), sqlc.arg(action), sqlc.narg(scope), sqlc.arg(source),
        sqlc.arg(now));

-- name: ListConsentEvents :many
-- The user's own events, newest first.
SELECT id, consent, version, action, scope, source, created_at FROM `consent_events`
WHERE user_id = sqlc.arg(user_id) ORDER BY id DESC LIMIT ?;
