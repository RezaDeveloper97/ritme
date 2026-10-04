-- Consents (B-N1-12, goose 00012): one row per (user, consent) once answered, Go only.
-- Read and written by GET/PUT /profile/consents. Always scoped by user_id.

-- name: ListUserConsents :many
SELECT consent, granted, version, granted_at, revoked_at FROM `user_consents` WHERE user_id = ? ORDER BY id;

-- name: GrantUserConsent :exec
-- A grant keeps revoked_at (the last withdrawal), stamps granted_at and the version of the consent text in force
-- (B-N6-05, internal/consent); a repeated grant of the same version leaves the row untouched.
INSERT INTO `user_consents` (user_id, consent, granted, version, granted_at, revoked_at, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(consent), 1, sqlc.arg(version), sqlc.arg(now), NULL, sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  granted_at = IF(granted = 1 AND version <=> VALUES(version), granted_at, VALUES(granted_at)),
  updated_at = IF(granted = 1 AND version <=> VALUES(version), updated_at, VALUES(updated_at)),
  version = VALUES(version),
  granted = 1;

-- name: RevokeUserConsent :exec
-- A withdrawal keeps granted_at (the last grant) and stamps revoked_at; revoking what was never granted stores
-- an explicit «no» without a revoked_at.
INSERT INTO `user_consents` (user_id, consent, granted, granted_at, revoked_at, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(consent), 0, NULL, NULL, sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  revoked_at = IF(granted = 1, VALUES(created_at), revoked_at),
  updated_at = IF(granted = 1, VALUES(updated_at), updated_at),
  granted = 0;
