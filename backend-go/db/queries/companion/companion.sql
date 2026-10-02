-- Companion «همدم» & family (bloom B-N4-01, internal/companion). Every statement is scoped by the owner and/or the
-- companion user in the statement itself; access checks only ever consider `active` links.

-- name: LockOwner :one
-- Per-owner mutex for invite / accept / revoke (the uniqueness rules live in Go, not in a unique key).
SELECT id FROM `users` WHERE id = ? FOR UPDATE;

-- name: GetUserMobile :one
SELECT mobile FROM `users` WHERE id = ?;

-- name: CountOpenCompanions :one
-- Non-revoked links of an owner, optionally of one type ('' = any).
SELECT COUNT(*) FROM `companions`
WHERE owner_id = sqlc.arg(owner_id) AND status <> 'revoked'
  AND (sqlc.arg(type) = '' OR type = sqlc.arg(type));

-- name: CountOpenLinks :one
-- Non-revoked links between an owner and one companion account.
SELECT COUNT(*) FROM `companions`
WHERE owner_id = ? AND companion_user_id = ? AND status <> 'revoked';

-- name: CreateCompanion :execlastid
INSERT INTO `companions` (owner_id, type, status, display_name, invited_at, created_at, updated_at)
VALUES (sqlc.arg(owner_id), sqlc.arg(type), 'invited', sqlc.arg(display_name), sqlc.arg(now), sqlc.arg(now), sqlc.arg(now));

-- name: GetCompanion :one
SELECT * FROM `companions` WHERE id = ? LIMIT 1;

-- name: ListOwnerCompanions :many
-- The owner's invited and active links, oldest first.
SELECT * FROM `companions` WHERE owner_id = ? AND status <> 'revoked' ORDER BY id;

-- name: ListCompanionLinks :many
-- The active links in which the user is the companion, oldest first.
SELECT * FROM `companions` WHERE companion_user_id = ? AND status = 'active' ORDER BY id;

-- name: ActivateCompanion :execrows
UPDATE `companions`
SET companion_user_id = sqlc.arg(companion_user_id), status = 'active', accepted_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'invited';

-- name: RevokeCompanion :execrows
UPDATE `companions`
SET status = 'revoked', revoked_at = sqlc.arg(now), revoked_by = sqlc.arg(revoked_by), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status <> 'revoked';

-- name: CreateInvite :execlastid
INSERT INTO `companion_invites` (companion_id, owner_id, phone, code_hash, attempts, expires_at, created_at, updated_at)
VALUES (sqlc.arg(companion_id), sqlc.arg(owner_id), sqlc.arg(phone), sqlc.arg(code_hash), 0, sqlc.arg(expires_at),
  sqlc.arg(now), sqlc.arg(now));

-- name: GetInviteByHash :one
SELECT * FROM `companion_invites` WHERE code_hash = ? LIMIT 1;

-- name: GetInvite :one
SELECT * FROM `companion_invites` WHERE id = ? LIMIT 1;

-- name: IncrementInviteAttempts :exec
UPDATE `companion_invites` SET attempts = attempts + 1, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND attempts < 255;

-- name: MarkInviteUsed :execrows
-- One-time: only an unused, unrevoked invite flips.
UPDATE `companion_invites` SET used_at = sqlc.arg(now), used_by_id = sqlc.arg(used_by_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND used_at IS NULL AND revoked_at IS NULL;

-- name: RevokeOpenInvites :exec
UPDATE `companion_invites` SET revoked_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE companion_id = sqlc.arg(companion_id) AND used_at IS NULL AND revoked_at IS NULL;

-- name: ListOpenInvitesForOwner :many
-- Unused, unrevoked, unexpired invites of an owner's links (the pending card shows the expiry), newest first.
SELECT * FROM `companion_invites`
WHERE owner_id = sqlc.arg(owner_id) AND used_at IS NULL AND revoked_at IS NULL AND expires_at > sqlc.arg(now)
ORDER BY id DESC;

-- name: ListLinkInvites :many
-- Every invite of one of the owner's links, newest first.
SELECT * FROM `companion_invites` WHERE owner_id = ? AND companion_id = ? ORDER BY id DESC;

-- name: UpsertGrant :exec
INSERT INTO `companion_grants` (companion_id, section, level, created_at, updated_at)
VALUES (sqlc.arg(companion_id), sqlc.arg(section), sqlc.arg(level), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE level = VALUES(level), updated_at = VALUES(updated_at);

-- name: DeleteGrant :exec
DELETE FROM `companion_grants` WHERE companion_id = ? AND section = ?;

-- name: DeleteGrants :exec
DELETE FROM `companion_grants` WHERE companion_id = ?;

-- name: ListGrants :many
SELECT * FROM `companion_grants` WHERE companion_id = ? ORDER BY id;

-- name: ListOwnerGrants :many
-- Grants of every non-revoked link of an owner.
SELECT g.* FROM `companion_grants` g
JOIN `companions` c ON c.id = g.companion_id
WHERE c.owner_id = ? AND c.status <> 'revoked'
ORDER BY g.id;

-- name: ListCompanionUserGrants :many
-- Grants of every active link in which the user is the companion.
SELECT g.* FROM `companion_grants` g
JOIN `companions` c ON c.id = g.companion_id
WHERE c.companion_user_id = ? AND c.status = 'active'
ORDER BY g.id;

-- name: GetAccess :one
-- The viewer's level on one section of the owner's data through an active link (no row = none).
SELECT c.id AS companion_id, g.level FROM `companions` c
JOIN `companion_grants` g ON g.companion_id = c.id AND g.section = sqlc.arg(section)
WHERE c.owner_id = sqlc.arg(owner_id) AND c.companion_user_id = sqlc.arg(viewer_id) AND c.status = 'active'
ORDER BY c.id
LIMIT 1;

-- name: CreateFamily :execlastid
INSERT INTO `families` (owner_id, companion_id, created_at, updated_at)
VALUES (sqlc.arg(owner_id), sqlc.arg(companion_id), sqlc.arg(now), sqlc.arg(now));

-- name: GetFamilyByCompanion :one
SELECT * FROM `families` WHERE companion_id = ? LIMIT 1;

-- name: SetFamilySpouse :exec
UPDATE `families` SET spouse_user_id = sqlc.arg(spouse_user_id), updated_at = sqlc.arg(now)
WHERE companion_id = sqlc.arg(companion_id);

-- name: DeleteFamilyByCompanion :exec
DELETE FROM `families` WHERE companion_id = ?;

-- name: ListUserFamilies :many
-- Families the user belongs to (as owner or as accepted spouse) through an active spouse link.
SELECT f.* FROM `families` f
JOIN `companions` c ON c.id = f.companion_id
WHERE (f.owner_id = sqlc.arg(owner_id) OR f.spouse_user_id = sqlc.arg(spouse_user_id)) AND c.status = 'active'
ORDER BY f.id;

-- name: AddFamilyChild :exec
INSERT IGNORE INTO `family_children` (family_id, child_id, created_at, updated_at)
VALUES (sqlc.arg(family_id), sqlc.arg(child_id), sqlc.arg(now), sqlc.arg(now));

-- name: DeleteFamilyChildren :exec
DELETE FROM `family_children` WHERE family_id = ?;

-- name: ListFamilyChildIDs :many
SELECT child_id FROM `family_children` WHERE family_id = ? ORDER BY child_id;

-- name: InsertAudit :exec
INSERT INTO `companion_audit_logs` (owner_id, actor_id, companion_id, section, action, created_at)
VALUES (sqlc.arg(owner_id), sqlc.arg(actor_id), sqlc.arg(companion_id), sqlc.arg(section), sqlc.arg(action), sqlc.arg(now));

-- name: ListOwnerAudit :many
-- The owner's audit trail, newest first.
SELECT * FROM `companion_audit_logs` WHERE owner_id = ? ORDER BY id DESC LIMIT ?;
