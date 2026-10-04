-- Versioned consents (B-N6-05, goose 00012 + 00032): the AI feature consent gate and GET/PUT /api/v1/consents*.
-- Same table as the B-N1-12 privacy toggles (db/queries/profile/consents.sql); always scoped by user_id.

-- name: GetConsent :one
SELECT consent, granted, version, granted_at, revoked_at FROM `user_consents` WHERE user_id = ? AND consent = ?;

-- name: ListConsents :many
SELECT consent, granted, version, granted_at, revoked_at FROM `user_consents` WHERE user_id = ? ORDER BY id;

-- name: GrantConsent :exec
-- Accepting a text stamps its version; granted_at moves only when the grant is new or the version changes, so a
-- repeated accept of the same text leaves the row untouched. revoked_at keeps the last withdrawal.
-- (MariaDB applies the assignments left to right: granted / version are read before they are overwritten.)
INSERT INTO `user_consents` (user_id, consent, granted, version, granted_at, revoked_at, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(consent), 1, sqlc.arg(version), sqlc.arg(now), NULL, sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  granted_at = IF(granted = 1 AND version <=> VALUES(version), granted_at, VALUES(granted_at)),
  updated_at = IF(granted = 1 AND version <=> VALUES(version), updated_at, VALUES(updated_at)),
  version = VALUES(version),
  granted = 1;

-- name: RevokeConsent :exec
-- A withdrawal keeps granted_at and version (what was last accepted) and stamps revoked_at; revoking what was
-- never granted stores an explicit «no» without a revoked_at.
INSERT INTO `user_consents` (user_id, consent, granted, version, granted_at, revoked_at, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(consent), 0, NULL, NULL, NULL, sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  revoked_at = IF(granted = 1, VALUES(created_at), revoked_at),
  updated_at = IF(granted = 1, VALUES(updated_at), updated_at),
  granted = 0;
