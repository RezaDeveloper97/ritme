-- T-M12-14b: the account export reads each table with an explicit column list (an allow-list):
-- a column the export must not carry (hash, token, path, sealed blob) is simply not selected here.
-- Generated once from the schema, kept by hand; TestAccountExport_ColumnGuard fails when a table gains a column.

-- name: AcctExportFamilies :many
SELECT `id`, `owner_id`, `spouse_user_id`, `companion_id`, `created_at`, `updated_at`
FROM `families`
WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED) OR spouse_user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportFamilyChildren :many
SELECT `id`, `family_id`, `child_id`, `created_at`, `updated_at`
FROM `family_children`
WHERE child_id IN (SELECT id FROM `children` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportCompanions :many
SELECT `id`, `owner_id`, `companion_user_id`, `type`, `status`, `display_name`, `invited_at`, `accepted_at`, `revoked_at`, `revoked_by`, `created_at`, `updated_at`, `role`, `template`, `expires_at`, `urgent_sms`, `owner_confirmed_at`, `goal`, `share_pain`, `share_mood`, `share_energy`, `share_sleep`, `share_period_days`, `action_prefs`
FROM `companions`
WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED) OR companion_user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportCompanionGrants :many
SELECT `id`, `companion_id`, `section`, `level`, `created_at`, `updated_at`, `mode_epoch`
FROM `companion_grants`
WHERE companion_id IN (SELECT id FROM `companions` WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED) OR companion_user_id = CAST(sqlc.arg(uid) AS UNSIGNED))
ORDER BY id;

-- name: AcctExportCompanionInvites :many
SELECT `id`, `companion_id`, `owner_id`, `phone`, `attempts`, `expires_at`, `used_at`, `used_by_id`, `revoked_at`, `created_at`, `updated_at`
FROM `companion_invites`
WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportCompanionAuditLogs :many
SELECT `id`, `owner_id`, `actor_id`, `companion_id`, `section`, `action`, `created_at`, `subject_type`, `subject_id`, `diff_json`, `payload_key_id`, `undo_deadline`, `undone_at`, `undone_by`, `proposal_id`
FROM `companion_audit_logs`
WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportFamilyAttendance :many
SELECT `id`, `reminder_id`, `member_id`, `created_at`
FROM `family_attendance`
WHERE member_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportFamilyMemberAlerts :many
SELECT `id`, `member_id`, `owner_id`, `link_id`, `source_alert_id`, `source_hit`, `level`, `created_at`, `seen_at`, `sms_sent_at`
FROM `family_member_alerts`
WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportFamilyMemberNotices :many
SELECT `id`, `member_id`, `owner_id`, `link_id`, `kind`, `dedupe_key`, `due_at`, `sent_at`, `deferred_reason`, `notification_id`, `created_at`
FROM `family_member_notices`
WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportFamilyNudges :many
SELECT `id`, `owner_id`, `member_id`, `link_id`, `target_type`, `target_id`, `local_date`, `slot`, `sent_at`
FROM `family_nudges`
WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED) OR member_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportFamilyOwnerState :many
SELECT `owner_id`, `mode`, `mode_epoch`, `review_pending`, `updated_at`
FROM `family_owner_state`
WHERE owner_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY owner_id;

-- name: AcctExportFamilyProposals :many
SELECT `id`, `subject_type`, `subject_id`, `link_id`, `actor_id`, `approver_id`, `kind`, `domain`, `payload_key_id`, `status`, `expires_at`, `decided_by`, `decided_at`, `applied_ref`, `created_at`, `updated_at`
FROM `family_proposals`
WHERE actor_id = CAST(sqlc.arg(uid) AS UNSIGNED) OR approver_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;

-- name: AcctExportFamilyShareRequests :many
SELECT `id`, `from_user_id`, `to_user_id`, `link_id`, `domains`, `status`, `expires_at`, `decided_at`, `created_at`
FROM `family_share_requests`
WHERE from_user_id = CAST(sqlc.arg(uid) AS UNSIGNED) OR to_user_id = CAST(sqlc.arg(uid) AS UNSIGNED)
ORDER BY id;
