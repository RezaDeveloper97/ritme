-- Family v6 storage (T-M11-02, 00106; docs/engine-v6/family.md §5, §7; gap 05 §5.3). Read by the family.Authorizer
-- (internal/family, T-M11-05): the snapshot statements also carry the owner's mode inputs (stored life mode, active
-- pregnancy, goal — enums.ResolveLifeMode) and sex (F-20), so check() and the lazy SyncMode need one statement.
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
  u.blocked_at AS owner_blocked_at,
  lp.life_mode AS owner_life_mode, lp.gender AS owner_gender,
  CAST(EXISTS(SELECT 1 FROM `pregnancy_profiles` pp WHERE pp.user_id = c.owner_id AND pp.pregnancy_mode = 1) AS SIGNED)
    AS owner_pregnancy_active,
  up.user_goal AS owner_user_goal
FROM `companions` c
JOIN `users` u ON u.id = c.owner_id
LEFT JOIN `companion_grants` g ON g.companion_id = c.id
LEFT JOIN `family_owner_state` s ON s.owner_id = c.owner_id
LEFT JOIN `user_life_profiles` lp ON lp.user_id = c.owner_id
LEFT JOIN `user_profiles` up ON up.user_id = c.owner_id
WHERE c.owner_id = sqlc.arg(owner_id) AND c.companion_user_id = sqlc.arg(member_id) AND c.status = 'active'
ORDER BY c.id, g.section;

-- name: FamilySnapshotForMember :many
-- Every owner of one member (member home). Index companions_companion_user_id_status_owner_id_index.
SELECT c.id AS link_id, c.owner_id, c.companion_user_id AS member_id, c.type, c.role, c.template, c.status,
  c.expires_at, c.urgent_sms, c.owner_confirmed_at,
  g.section, g.level, g.mode_epoch AS grant_mode_epoch,
  s.mode AS owner_mode, s.mode_epoch AS owner_mode_epoch, s.review_pending,
  u.blocked_at AS owner_blocked_at,
  lp.life_mode AS owner_life_mode, lp.gender AS owner_gender,
  CAST(EXISTS(SELECT 1 FROM `pregnancy_profiles` pp WHERE pp.user_id = c.owner_id AND pp.pregnancy_mode = 1) AS SIGNED)
    AS owner_pregnancy_active,
  up.user_goal AS owner_user_goal
FROM `companions` c
JOIN `users` u ON u.id = c.owner_id
LEFT JOIN `companion_grants` g ON g.companion_id = c.id
LEFT JOIN `family_owner_state` s ON s.owner_id = c.owner_id
LEFT JOIN `user_life_profiles` lp ON lp.user_id = c.owner_id
LEFT JOIN `user_profiles` up ON up.user_id = c.owner_id
WHERE c.companion_user_id = sqlc.arg(member_id) AND c.status = 'active'
ORDER BY c.owner_id, c.id, g.section;

-- name: FamilyLockPair :many
-- RecheckTx (CMP-L4): the pair snapshot read with shared locks inside a write transaction, so a revoke or grant
-- change committed meanwhile wins (a locking read sees the latest committed rows) and cannot commit until we do.
SELECT c.id AS link_id, c.owner_id, c.companion_user_id AS member_id, c.type, c.role, c.status, c.expires_at,
  g.section, g.level, g.mode_epoch AS grant_mode_epoch,
  s.mode AS owner_mode, s.mode_epoch AS owner_mode_epoch,
  u.blocked_at AS owner_blocked_at, lp.gender AS owner_gender
FROM `companions` c
JOIN `users` u ON u.id = c.owner_id
LEFT JOIN `companion_grants` g ON g.companion_id = c.id
LEFT JOIN `family_owner_state` s ON s.owner_id = c.owner_id
LEFT JOIN `user_life_profiles` lp ON lp.user_id = c.owner_id
WHERE c.owner_id = sqlc.arg(owner_id) AND c.companion_user_id = sqlc.arg(member_id) AND c.status = 'active'
ORDER BY c.id, g.section
LOCK IN SHARE MODE;

-- name: FamilyGuardiansOfUser :many
-- The actor's guardian rows (any status; Decide reads status) for a set of children: one statement per request.
SELECT child_id, level, sub_scopes, status FROM `child_guardians`
WHERE user_id = sqlc.arg(user_id) AND child_id IN (sqlc.slice(child_ids))
ORDER BY child_id;

-- name: FamilyLockGuardiansOfUser :many
-- RecheckTx: the same rows with shared locks.
SELECT child_id, level, sub_scopes, status FROM `child_guardians`
WHERE user_id = sqlc.arg(user_id) AND child_id IN (sqlc.slice(child_ids))
ORDER BY child_id
LOCK IN SHARE MODE;

-- name: FamilyOwnerModeInputs :one
-- SyncMode outside a snapshot (after a mode write): the inputs of enums.ResolveLifeMode and the stored state.
SELECT u.id, lp.life_mode, lp.gender,
  CAST(EXISTS(SELECT 1 FROM `pregnancy_profiles` pp WHERE pp.user_id = u.id AND pp.pregnancy_mode = 1) AS SIGNED)
    AS pregnancy_active,
  up.user_goal, s.mode AS state_mode, s.mode_epoch AS state_mode_epoch,
  s.review_pending AS state_review_pending
FROM `users` u
LEFT JOIN `user_life_profiles` lp ON lp.user_id = u.id
LEFT JOIN `user_profiles` up ON up.user_id = u.id
LEFT JOIN `family_owner_state` s ON s.owner_id = u.id
WHERE u.id = sqlc.arg(owner_id) LIMIT 1;

-- name: GetFamilyOwnerState :one
SELECT owner_id, mode, mode_epoch, review_pending, updated_at FROM `family_owner_state` WHERE owner_id = ? LIMIT 1;

-- name: UpsertFamilyGrant :exec
-- Grants.Apply: one v6 grant stamped with the owner's current mode epoch (NULL for an epoch-free domain is fine too).
INSERT INTO `companion_grants` (companion_id, section, level, mode_epoch, created_at, updated_at)
VALUES (sqlc.arg(companion_id), sqlc.arg(section), sqlc.arg(level), sqlc.arg(mode_epoch), sqlc.arg(now), sqlc.arg(now))
ON DUPLICATE KEY UPDATE level = VALUES(level), mode_epoch = VALUES(mode_epoch), updated_at = VALUES(updated_at);

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

-- ---------------------------------------------------------------------------------------------------------------------
-- Family v6 storage, part B (T-M11-03, 00107; family.md §6–§8; gap 05 §5.4, §5.5). No reader or writer yet: proposals
-- (T-M11-09/11), share requests (T-M11-08), nudges (T-M11-10), member alerts and notices (T-M11-12/13) wire them up.
-- Expiry is lazy (every read and transition compares expires_at with `now`) so correctness never waits for a sweep.
-- Idempotent inserts use ON DUPLICATE KEY UPDATE id = id (rows affected 0 = already there); unlike INSERT IGNORE it
-- still fails on a missing link / user (FK) or a bad value.

-- name: InsertFamilyProposal :execlastid
-- A pending proposal. A second pending one with the same dedupe_key fails with 1062 (UNIQUE pending_key): the caller
-- answers with the existing one. expires_at = now + 7 days (family.md §7).
INSERT INTO `family_proposals` (subject_type, subject_id, link_id, actor_id, approver_id, kind, domain, payload_sealed,
  payload_key_id, status, dedupe_key, expires_at, created_at, updated_at)
VALUES (sqlc.arg(subject_type), sqlc.arg(subject_id), sqlc.arg(link_id), sqlc.arg(actor_id), sqlc.arg(approver_id),
  sqlc.arg(kind), sqlc.arg(domain), sqlc.arg(payload_sealed), sqlc.arg(payload_key_id), 'pending', sqlc.arg(dedupe_key),
  sqlc.arg(expires_at), sqlc.arg(now), sqlc.arg(now));

-- name: GetFamilyProposal :one
-- One proposal. expired = 1 for a pending row past expires_at (lazy expiry): the caller treats it as expired.
SELECT id, subject_type, subject_id, link_id, actor_id, approver_id, kind, domain, payload_sealed, payload_key_id,
  status, CAST(status = 'pending' AND expires_at <= sqlc.arg(now) AS SIGNED) AS expired,
  dedupe_key, expires_at, decided_by, decided_at, applied_ref, created_at, updated_at
FROM `family_proposals` WHERE id = sqlc.arg(id) LIMIT 1;

-- name: LockFamilyProposal :one
-- Approve / reject / cancel: lock the row, then DecideFamilyProposal. The caller treats a pending row past expires_at as
-- expired (ExpireFamilyProposal) and never applies it.
SELECT id, subject_type, subject_id, link_id, actor_id, approver_id, kind, domain, payload_sealed, payload_key_id, status,
  dedupe_key, expires_at, decided_by, decided_at, applied_ref, created_at, updated_at
FROM `family_proposals` WHERE id = sqlc.arg(id) LIMIT 1 FOR UPDATE;

-- name: ListPendingFamilyProposalsForApprover :many
-- The approver's waiting list, newest first; expired rows are left out. Index (approver_id, status, created_at).
SELECT id, subject_type, subject_id, link_id, actor_id, approver_id, kind, domain, payload_sealed, payload_key_id, status,
  dedupe_key, expires_at, decided_by, decided_at, applied_ref, created_at, updated_at
FROM `family_proposals`
WHERE approver_id = sqlc.arg(approver_id) AND status = 'pending' AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC, id DESC
LIMIT ?;

-- name: ListFamilyProposalsByActor :many
-- The member's own proposals (outcome cards), newest first; expired = 1 marks a lapsed pending row. Index (actor_id, …).
SELECT id, subject_type, subject_id, link_id, actor_id, approver_id, kind, domain,
  status, CAST(status = 'pending' AND expires_at <= sqlc.arg(now) AS SIGNED) AS expired,
  expires_at, decided_at, applied_ref, created_at
FROM `family_proposals`
WHERE actor_id = sqlc.arg(actor_id)
ORDER BY created_at DESC, id DESC
LIMIT ?;

-- name: CountPendingFamilyProposalsByActor :one
-- Quota: the member's live pending proposals. Index (actor_id, status, created_at).
SELECT COUNT(*) FROM `family_proposals`
WHERE actor_id = sqlc.arg(actor_id) AND status = 'pending' AND expires_at > sqlc.arg(now);

-- name: DecideFamilyProposal :execrows
-- pending → approved | rejected | cancelled, only while still live. 0 rows = already decided or expired (the caller
-- re-reads). applied_ref = the record an approval created.
UPDATE `family_proposals`
SET status = sqlc.arg(status), decided_by = sqlc.arg(decided_by), decided_at = sqlc.arg(decided_at),
  applied_ref = sqlc.arg(applied_ref), updated_at = sqlc.arg(decided_at)
WHERE id = sqlc.arg(id) AND status = 'pending' AND expires_at > sqlc.arg(now);

-- name: ExpireFamilyProposal :execrows
-- Lazy expiry of one row found past expires_at on read or approve. updated_at = now (a nullable column, hence its own
-- parameter; the same holds for decided_at in the Decide statements).
UPDATE `family_proposals` SET status = 'expired', updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND status = 'pending' AND expires_at <= sqlc.arg(now);

-- name: ExpireFamilyProposals :execrows
-- Hourly sweep (idempotent; batches until 0). Index (status, expires_at).
UPDATE `family_proposals` SET status = 'expired', updated_at = sqlc.arg(updated_at)
WHERE status = 'pending' AND expires_at <= sqlc.arg(now)
ORDER BY expires_at
LIMIT ?;

-- name: CancelPendingFamilyProposalsForLink :execrows
-- Revoke / expiry of a link: its pending proposals can no longer be applied (rows kept for the activity log).
UPDATE `family_proposals` SET status = 'cancelled', updated_at = sqlc.arg(now)
WHERE link_id = sqlc.arg(link_id) AND status = 'pending';

-- name: InsertFamilyShareRequest :execlastid
-- A member asks for more access; domains = {domain: level}. Nothing applies until the owner accepts.
INSERT INTO `family_share_requests` (from_user_id, to_user_id, link_id, domains, status, expires_at, created_at)
VALUES (sqlc.arg(from_user_id), sqlc.arg(to_user_id), sqlc.arg(link_id), sqlc.arg(domains), 'pending',
  sqlc.arg(expires_at), sqlc.arg(now));

-- name: GetFamilyShareRequest :one
-- One request asked of to_user_id (the data owner: approve / decline path, IDOR-scoped in the query); expired = 1 for
-- a pending row past expires_at (lazy expiry).
SELECT id, from_user_id, to_user_id, link_id, domains,
  status, CAST(status = 'pending' AND expires_at <= sqlc.arg(now) AS SIGNED) AS expired,
  expires_at, decided_at, created_at
FROM `family_share_requests` WHERE id = sqlc.arg(id) AND to_user_id = sqlc.arg(to_user_id) LIMIT 1;

-- name: ListPendingFamilyShareRequestsForOwner :many
-- The owner's inbox of live requests, newest first. Index (to_user_id, status, created_at).
SELECT id, from_user_id, to_user_id, link_id, domains, status, expires_at, decided_at, created_at
FROM `family_share_requests`
WHERE to_user_id = sqlc.arg(to_user_id) AND status = 'pending' AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC, id DESC
LIMIT ?;

-- name: CountFamilyShareRequestsForPairSince :one
-- Throttle: requests one member sent one owner since a time. Index (from_user_id, to_user_id, created_at).
SELECT COUNT(*) FROM `family_share_requests`
WHERE from_user_id = sqlc.arg(from_user_id) AND to_user_id = sqlc.arg(to_user_id) AND created_at >= sqlc.arg(since);

-- name: DecideFamilyShareRequest :execrows
-- pending → accepted | declined by the owner it was sent to, only while live. 0 rows = decided, expired or not hers.
UPDATE `family_share_requests` SET status = sqlc.arg(status), decided_at = sqlc.arg(decided_at)
WHERE id = sqlc.arg(id) AND to_user_id = sqlc.arg(to_user_id) AND status = 'pending' AND expires_at > sqlc.arg(now);

-- name: ExpirePendingFamilyShareRequestsForLink :execrows
-- Revoke / expiry of a link: its open requests end.
UPDATE `family_share_requests` SET status = 'expired', decided_at = sqlc.arg(now)
WHERE link_id = sqlc.arg(link_id) AND status = 'pending';

-- name: InsertFamilyNudge :execrows
-- One nudge per (owner, target, owner-local day, slot); slot = dose HH:MM, '' for an appointment. 1 = sent now, 0 =
-- already nudged (double tap). Run inside the link-row lock together with CountFamilyNudgesForPair (F-06).
INSERT INTO `family_nudges` (owner_id, member_id, link_id, target_type, target_id, local_date, slot, sent_at)
VALUES (sqlc.arg(owner_id), sqlc.arg(member_id), sqlc.arg(link_id), sqlc.arg(target_type), sqlc.arg(target_id),
  sqlc.arg(local_date), sqlc.arg(slot), sqlc.arg(sent_at))
ON DUPLICATE KEY UPDATE id = id;

-- name: CountFamilyNudgesForPair :one
-- Quota: nudges one member sent one owner in [from, to) — the owner-local day (3 per day, F-06). Index (member_id,
-- owner_id, sent_at).
SELECT COUNT(*) FROM `family_nudges`
WHERE member_id = sqlc.arg(member_id) AND owner_id = sqlc.arg(owner_id)
  AND sent_at >= sqlc.arg(from_at) AND sent_at < sqlc.arg(to_at);

-- name: ListFamilyNudgesForOwnerDay :many
-- Which of the owner's targets were already nudged on one local day (greys out the member's nudge buttons).
SELECT target_type, target_id, slot, member_id, sent_at FROM `family_nudges`
WHERE owner_id = sqlc.arg(owner_id) AND local_date = sqlc.arg(local_date);

-- name: InsertFamilyMemberAlert :execrows
-- Fan-out of one owner alert hit to one member (job family_member_alert, retried): 1 = new, 0 = already relayed.
INSERT INTO `family_member_alerts` (member_id, owner_id, link_id, source_alert_id, source_hit, level, created_at)
VALUES (sqlc.arg(member_id), sqlc.arg(owner_id), sqlc.arg(link_id), sqlc.arg(source_alert_id), sqlc.arg(source_hit),
  sqlc.arg(level), sqlc.arg(now))
ON DUPLICATE KEY UPDATE id = id;

-- name: ListFamilyMemberAlerts :many
-- The member's alert slot, newest first. Index (member_id, created_at).
SELECT id, member_id, owner_id, link_id, source_alert_id, source_hit, level, created_at, seen_at, sms_sent_at
FROM `family_member_alerts`
WHERE member_id = sqlc.arg(member_id)
ORDER BY created_at DESC, id DESC
LIMIT ?;

-- name: CountUnseenFamilyMemberAlerts :one
SELECT COUNT(*) FROM `family_member_alerts` WHERE member_id = sqlc.arg(member_id) AND seen_at IS NULL;

-- name: MarkFamilyMemberAlertSeen :execrows
UPDATE `family_member_alerts` SET seen_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND member_id = sqlc.arg(member_id) AND seen_at IS NULL;

-- name: MarkFamilyMemberAlertSmsSent :execrows
-- The optional neutral urgent SMS (F-09) goes once per alert row: 0 rows = already sent.
UPDATE `family_member_alerts` SET sms_sent_at = sqlc.arg(now) WHERE id = sqlc.arg(id) AND sms_sent_at IS NULL;

-- name: DeleteFamilyMemberAlertsForLink :execrows
-- Revoke / expiry: the member loses the link's alerts. Index (link_id).
DELETE FROM `family_member_alerts` WHERE link_id = sqlc.arg(link_id);

-- name: InsertFamilyMemberNotice :execrows
-- Queue one member notice; dedupe_key is unique per member (e.g. `appt_tomorrow:<reminder>:<date>`). 1 = queued, 0 =
-- already queued.
INSERT INTO `family_member_notices` (member_id, owner_id, link_id, kind, dedupe_key, due_at, created_at)
VALUES (sqlc.arg(member_id), sqlc.arg(owner_id), sqlc.arg(link_id), sqlc.arg(kind), sqlc.arg(dedupe_key),
  sqlc.arg(due_at), sqlc.arg(now))
ON DUPLICATE KEY UPDATE id = id;

-- name: ListDueFamilyMemberNotices :many
-- Dispatcher: unsent notices that are due, oldest first. Index (sent_at, due_at).
SELECT id, member_id, owner_id, link_id, kind, dedupe_key, due_at, sent_at, deferred_reason, notification_id, created_at
FROM `family_member_notices`
WHERE sent_at IS NULL AND due_at <= sqlc.arg(now)
ORDER BY due_at, id
LIMIT ?;

-- name: MarkFamilyMemberNoticeSent :execrows
-- notification_id = the user_notifications row it produced (InsertLinkedUserNotification). 0 rows = already sent.
UPDATE `family_member_notices` SET sent_at = sqlc.arg(now), notification_id = sqlc.arg(notification_id),
  deferred_reason = NULL
WHERE id = sqlc.arg(id) AND sent_at IS NULL;

-- name: DeferFamilyMemberNotice :execrows
-- Quiet hours / preferences (notifications.Decide): move due_at and record why.
UPDATE `family_member_notices` SET due_at = sqlc.arg(due_at), deferred_reason = sqlc.arg(deferred_reason)
WHERE id = sqlc.arg(id) AND sent_at IS NULL;

-- name: DeleteFamilyMemberNoticesForLink :execrows
-- Revoke / expiry. Index (link_id).
DELETE FROM `family_member_notices` WHERE link_id = sqlc.arg(link_id);

-- name: InsertLinkedUserNotification :execlastid
-- A member inbox row tied to its link (neutral copy only, SPEC §1.8), so DeleteUserNotificationsForLink can purge it.
INSERT INTO `user_notifications` (user_id, type, title, body, action_url, data, link_id, created_at, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(type), sqlc.arg(title), sqlc.arg(body), sqlc.arg(action_url), sqlc.arg(data),
  sqlc.arg(link_id), sqlc.arg(now), sqlc.arg(now));

-- name: DeleteUserNotificationsForLink :execrows
-- Revoke / expiry: every member notice of the link, by the indexed link_id (FAM-L4; no JSON scan).
DELETE FROM `user_notifications` WHERE link_id = sqlc.arg(link_id);

-- name: InsertFamilyAttendance :execrows
-- «میام» on an owner appointment (reminders row). 1 = new, 0 = already marked.
INSERT INTO `family_attendance` (reminder_id, member_id, created_at)
VALUES (sqlc.arg(reminder_id), sqlc.arg(member_id), sqlc.arg(now))
ON DUPLICATE KEY UPDATE id = id;

-- name: DeleteFamilyAttendance :execrows
DELETE FROM `family_attendance` WHERE reminder_id = sqlc.arg(reminder_id) AND member_id = sqlc.arg(member_id);

-- name: ListFamilyAttendanceForReminder :many
-- Who is coming to one appointment. Index (reminder_id, member_id).
SELECT member_id, created_at FROM `family_attendance` WHERE reminder_id = sqlc.arg(reminder_id) ORDER BY id;

-- name: DeleteFamilyAttendanceForPair :execrows
-- Revoke / expiry: the member's «میام» marks on the owner's appointments go.
DELETE a FROM `family_attendance` a
JOIN `reminders` r ON r.id = a.reminder_id
WHERE r.user_id = sqlc.arg(owner_id) AND a.member_id = sqlc.arg(member_id);
