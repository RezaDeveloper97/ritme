-- Family invites v6 and share requests (T-M11-08; SPEC §1.7–§1.8; docs/engine-v6/family.md §7; DECISIONS.md F-11,
-- F-12, F-13, F-17; D-196). internal/family/invites and internal/family/sharerequests. Every statement is scoped by the
-- owner, the member or the invite in the statement itself.
--
-- Link statuses: invited → active, or invited → awaiting_owner → active (F-11: a code-only invite with child write,
-- urgent alert or vitals waits for the owner's confirmation; the member sees nothing before it, since every member
-- read only considers `active`), or → revoked with revoked_by owner | companion | declined | expired (F-12).
-- A phone invite is bound to an account (the one with that mobile at invite time, or the one created later by the
-- signup hook) through companion_invites.used_by_id while used_at stays NULL: a *pending acceptance* the member answers
-- without the code (GET /family/invites/pending). Owner views never read used_by_id, so a binding tells the owner
-- nothing before the member accepts.

-- name: CreateFamilyLink :execlastid
-- A v6 link: an invited one (POST /family/members) or the member's own active reverse link of a mutual answer.
INSERT INTO `companions` (owner_id, companion_user_id, type, status, role, template, display_name, expires_at,
  urgent_sms, invited_at, accepted_at, created_at, updated_at)
VALUES (sqlc.arg(owner_id), sqlc.arg(companion_user_id), sqlc.arg(type), sqlc.arg(status), sqlc.arg(role),
  sqlc.arg(template), sqlc.arg(display_name), sqlc.arg(expires_at), sqlc.arg(urgent_sms), sqlc.arg(now),
  sqlc.arg(accepted_at), sqlc.arg(now), sqlc.arg(now));

-- name: AcceptFamilyLink :execrows
-- invited → active | awaiting_owner for the accepting member. 0 rows = no longer invited.
UPDATE `companions`
SET companion_user_id = sqlc.arg(companion_user_id), status = sqlc.arg(status), accepted_at = sqlc.arg(now),
  updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND status = 'invited';

-- name: ConfirmFamilyLink :execrows
-- The owner confirms an accepted high-impact code invite (F-11): awaiting_owner → active, only while the
-- confirmation window is open (accepted after stale_before) and the link has not reached its end date.
UPDATE `companions` SET status = 'active', owner_confirmed_at = sqlc.arg(now), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id) AND status = 'awaiting_owner'
  AND accepted_at > sqlc.arg(stale_before) AND (expires_at IS NULL OR expires_at > sqlc.arg(now));

-- name: GetOpenLinkForPair :one
-- The active link owner → member (share requests apply to it).
SELECT * FROM `companions`
WHERE owner_id = sqlc.arg(owner_id) AND companion_user_id = sqlc.arg(member_id) AND status = 'active'
ORDER BY id LIMIT 1;

-- name: FamilyUserIDByMobile :one
-- The account of an invited mobile (bind a phone invite at creation). users.mobile is unique.
SELECT id FROM `users` WHERE mobile = sqlc.arg(mobile) LIMIT 1;

-- name: GetFamilyUserProfile :one
-- users.name (preview: the owner's first name) and the member's sex (male owner rule of a mutual link, F-20).
SELECT u.id, u.name, lp.gender FROM `users` u
LEFT JOIN `user_life_profiles` lp ON lp.user_id = u.id
WHERE u.id = sqlc.arg(id) LIMIT 1;

-- name: FamilyOwnerDates :one
-- Carer time-box inputs (F-13): the active pregnancy's EDD and the delivery date (both one row per user).
SELECT u.id, pp.estimated_due_date AS edd, pf.birth_date AS delivery FROM `users` u
LEFT JOIN `pregnancy_profiles` pp ON pp.user_id = u.id AND pp.pregnancy_mode = 1
LEFT JOIN `postpartum_profiles` pf ON pf.user_id = u.id
WHERE u.id = sqlc.arg(owner_id) LIMIT 1;

-- name: BindInviteToUser :execrows
-- Pending acceptance: an open phone invite bound to one account (used_by_id while used_at stays NULL).
UPDATE `companion_invites` SET used_by_id = sqlc.arg(user_id), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND used_at IS NULL AND revoked_at IS NULL AND used_by_id IS NULL;

-- name: ListUnboundPhoneInvites :many
-- Signup hook: the open, unbound phone invites of a mobile (not the new account's own).
SELECT i.id FROM `companion_invites` i
JOIN `companions` c ON c.id = i.companion_id
WHERE i.phone = sqlc.arg(phone) AND i.used_at IS NULL AND i.revoked_at IS NULL AND i.used_by_id IS NULL
  AND i.expires_at > sqlc.arg(now) AND c.status = 'invited' AND c.owner_id <> sqlc.arg(user_id)
ORDER BY i.id
LIMIT 50;

-- name: ListBoundInvites :many
-- GET /family/invites/pending: the open invites bound to the member, newest first. Index (used_by_id).
SELECT i.id, i.companion_id, i.owner_id, i.phone, i.expires_at FROM `companion_invites` i
JOIN `companions` c ON c.id = i.companion_id
WHERE i.used_by_id = sqlc.arg(user_id) AND i.used_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > sqlc.arg(now)
  AND c.status = 'invited'
ORDER BY i.id DESC
LIMIT 50;

-- name: GetBoundInvite :one
-- Accept / decline without the code: the newest open invite of a link bound to the member (no row = not hers).
SELECT * FROM `companion_invites`
WHERE companion_id = sqlc.arg(companion_id) AND used_by_id = sqlc.arg(user_id) AND used_at IS NULL AND revoked_at IS NULL
ORDER BY id DESC LIMIT 1;

-- name: GetLinkInviteIntent :one
-- The intent of an invite (mutual request, children level, consent version) kept on its `invited` activity row.
SELECT diff_json FROM `companion_audit_logs`
WHERE companion_id = sqlc.arg(companion_id) AND action = 'invited'
ORDER BY id DESC LIMIT 1;

-- name: ListExpiredInvitedLinks :many
-- family.invites_sweep: invited links without an open, unexpired invite (F-12).
SELECT c.id, c.owner_id, c.display_name FROM `companions` c
WHERE c.status = 'invited'
  AND NOT EXISTS (SELECT 1 FROM `companion_invites` i WHERE i.companion_id = c.id AND i.used_at IS NULL
    AND i.revoked_at IS NULL AND i.expires_at > sqlc.arg(now))
ORDER BY c.id
LIMIT ?;

-- name: ListExpiredActiveLinks :many
-- family.invites_sweep: active links past expires_at (carer time-box, F-13). Index (status, expires_at).
SELECT id, owner_id, companion_user_id, display_name FROM `companions`
WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at <= sqlc.arg(now)
ORDER BY id
LIMIT ?;

-- name: ListStaleAwaitingLinks :many
-- family.invites_sweep: awaiting_owner links the owner did not confirm in time (accepted at or before stale_before)
-- or past their end date.
SELECT id, owner_id, display_name FROM `companions`
WHERE status = 'awaiting_owner'
  AND (accepted_at <= sqlc.arg(stale_before) OR (expires_at IS NOT NULL AND expires_at <= sqlc.arg(now)))
ORDER BY id
LIMIT ?;

-- name: ListFamilyShareRequestDomainsForPairSince :many
-- Share-request throttle: one per domain per pair per 7 days. Index (from_user_id, to_user_id, created_at).
SELECT domains FROM `family_share_requests`
WHERE from_user_id = sqlc.arg(from_user_id) AND to_user_id = sqlc.arg(to_user_id) AND created_at > sqlc.arg(since);

-- name: ListFamilyShareRequestsFrom :many
-- GET /family/share-requests?box=out: the caller's own requests, newest first; expired = 1 for a lapsed pending row.
SELECT id, from_user_id, to_user_id, link_id, domains,
  status, CAST(status = 'pending' AND expires_at <= sqlc.arg(now) AS SIGNED) AS expired,
  expires_at, decided_at, created_at
FROM `family_share_requests`
WHERE from_user_id = sqlc.arg(from_user_id)
ORDER BY created_at DESC, id DESC
LIMIT ?;
