-- Family v6 storage (T-M11-02, 00106; docs/engine-v6/family.md §5, §7; gap 05 §5.3). No reader or writer yet: the
-- family.Authorizer, SyncMode and the activity writer arrive with T-M11-05 and later tasks.
--
-- Snapshot = links ⋈ grants ⋈ family_owner_state ⋈ users.blocked_at (the owner's), one row per (link, grant); a link
-- without grants yields one row with NULL grant columns, an owner without a state row NULL state columns (SyncMode
-- creates it lazily). Only active links: every other status denies (family.md §5 step 3), expiry is checked in Go.

-- name: FamilySnapshotForPair :many
-- One owner and one member (check() on a member request): a one-row ref on either new composite companions key.
SELECT c.id AS link_id, c.owner_id, c.companion_user_id AS member_id, c.type, c.role, c.template, c.status,
  c.expires_at, c.urgent_sms, c.owner_confirmed_at,
  g.section, g.level, g.mode_epoch AS grant_mode_epoch,
  s.mode AS owner_mode, s.mode_epoch AS owner_mode_epoch, s.review_pending,
  u.blocked_at AS owner_blocked_at
FROM `companions` c
JOIN `users` u ON u.id = c.owner_id
LEFT JOIN `companion_grants` g ON g.companion_id = c.id
LEFT JOIN `family_owner_state` s ON s.owner_id = c.owner_id
WHERE c.owner_id = sqlc.arg(owner_id) AND c.companion_user_id = sqlc.arg(member_id) AND c.status = 'active'
ORDER BY c.id, g.section;

-- name: FamilySnapshotForMember :many
-- Every owner of one member (member home). Index companions_companion_user_id_status_owner_id_index.
SELECT c.id AS link_id, c.owner_id, c.companion_user_id AS member_id, c.type, c.role, c.template, c.status,
  c.expires_at, c.urgent_sms, c.owner_confirmed_at,
  g.section, g.level, g.mode_epoch AS grant_mode_epoch,
  s.mode AS owner_mode, s.mode_epoch AS owner_mode_epoch, s.review_pending,
  u.blocked_at AS owner_blocked_at
FROM `companions` c
JOIN `users` u ON u.id = c.owner_id
LEFT JOIN `companion_grants` g ON g.companion_id = c.id
LEFT JOIN `family_owner_state` s ON s.owner_id = c.owner_id
WHERE c.companion_user_id = sqlc.arg(member_id) AND c.status = 'active'
ORDER BY c.owner_id, c.id, g.section;

-- name: UpsertFamilyOwnerState :execrows
-- SyncMode: store the owner's effective mode. A new owner starts at epoch 1, nothing to review; a different mode bumps
-- the epoch, sets review_pending (F-23: owner-data domains pause until she reviews) and stamps updated_at; the same mode
-- changes nothing. MariaDB evaluates the assignments left to right, so `mode` is compared before it is overwritten.
-- Rows affected (no CLIENT_FOUND_ROWS): 1 = created, 2 = mode changed (epoch bumped), 0 = unchanged.
INSERT INTO `family_owner_state` (owner_id, mode, mode_epoch, review_pending, updated_at)
VALUES (sqlc.arg(owner_id), sqlc.arg(mode), 1, 0, sqlc.arg(now))
ON DUPLICATE KEY UPDATE
  mode_epoch = IF(mode = VALUES(mode), mode_epoch, mode_epoch + 1),
  review_pending = IF(mode = VALUES(mode), review_pending, 1),
  updated_at = IF(mode = VALUES(mode), updated_at, VALUES(updated_at)),
  mode = VALUES(mode);

-- name: InsertFamilyActivity :execlastid
-- One family activity row (companion_audit_logs) with the v6 columns. payload_sealed is AES-256-GCM ciphertext (never
-- plaintext health data); diff_json carries levels only.
INSERT INTO `companion_audit_logs` (owner_id, actor_id, companion_id, section, action, subject_type, subject_id,
  diff_json, payload_sealed, payload_key_id, undo_deadline, proposal_id, created_at)
VALUES (sqlc.arg(owner_id), sqlc.arg(actor_id), sqlc.arg(companion_id), sqlc.arg(section), sqlc.arg(action),
  sqlc.arg(subject_type), sqlc.arg(subject_id), sqlc.arg(diff_json), sqlc.arg(payload_sealed), sqlc.arg(payload_key_id),
  sqlc.arg(undo_deadline), sqlc.arg(proposal_id), sqlc.arg(now));
