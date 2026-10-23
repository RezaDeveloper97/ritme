-- 00106_family_schema_a.sql — family schema v6, part A: link role / template / expiry, grant mode epoch, owner mode
-- state, activity-log columns (T-M11-02; SPEC v6 §1.7; docs/engine-v6/family.md §1–§3, §5, §7; gap 05 §1 rows 1–8,
-- §2.1, §2.12, §5.3; DECISIONS.md F-09, F-11, F-12, F-13, F-14, F-17, F-23; deviation D-190). Additive only: every
-- current reader and writer keeps working. The behaviour that reads these columns, and the legacy grant code / level
-- migration (cycle → cycle_phase, edit → view | write, symptoms removed), ship with T-M11-05.
--
-- companions             + role               partner | other_parent | carer | friend | teen_parent (NULL until
--                                             backfilled or written by the v6 invite flow; codes validated in Go)
--                        + template           cycle_buddy | meds_appointments | full_family | custom | legacy
--                        + expires_at         link expiry (carer time-box, F-13); NULL = no expiry
--                        + urgent_sms         owner-controlled optional neutral SMS for urgent alerts (F-09)
--                        + owner_confirmed_at owner confirmation of a high-impact code-only invite (F-11)
--                        status admits `awaiting_owner` (F-11) and revoked_by admits `expired | declined | system` (F-12):
--                        both are varchar(16) already (00026), so no ALTER is needed.
--                        KEYs: (companion_user_id, status, owner_id) for the member snapshot (every owner of a member)
--                        — it replaces its prefix (companion_user_id, status), which also served the companion_user_id
--                        FK —, (owner_id, companion_user_id, status) for the pair snapshot (check()), (status,
--                        expires_at) for the hourly expiry sweep.
-- companion_grants       + mode_epoch: the owner's mode epoch the grant was given (or re-stamped) in; a mode-scoped
--                        grant is live only while it equals family_owner_state.mode_epoch (family.md §5). NULL =
--                        inactive / not stamped yet. section varchar(32) and level varchar(8) already hold every v6
--                        code (longest: teen_period_week, fetal_activity; nudge, write) — unchanged.
-- family_owner_state     one row per owner: effective mode, mode_epoch (bumped on every mode change), review_pending
--                        (the owner's «دسترسی‌های جدید رو تنظیم کن» card, F-23). Written lazily by SyncMode.
-- companion_audit_logs   the family activity log: + subject_type / subject_id (child, household item, proposal …),
--                        diff_json (per-domain level diff), payload_sealed + payload_key_id (AES-256-GCM before/after
--                        JSON, purged after 30 days, F-14), undo_deadline / undone_at / undone_by (24 h undo),
--                        proposal_id (family_proposals arrives in T-M11-03; no FK on purpose). KEYs: (subject_type,
--                        subject_id, created_at), (undo_deadline) for the undo/purge sweeper, (created_at) for the
--                        12-month retention sweep.
--
-- Backfill (goose only; the Laravel twin is schema only, prod is not converted by Laravel): role from type — partner →
-- partner, spouse → partner (co-guardianship of shared children is T-M11-04), parent → teen_parent (F-17) — and
-- template = 'legacy' on every existing link. Idempotent (only NULL columns are filled). Rows created after Up by the
-- v1 code keep role / template NULL until T-M11-05 writes them; readers treat NULL like the backfill would.
--
-- Twin of backend/database/migrations/2026_10_23_000106_add_family_schema_a.php (Laravel owns the prod schema until
-- T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables; IF [NOT]
-- EXISTS everywhere so a database whose Laravel half already ran does not fail.

-- +goose Up
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `role` varchar(16) DEFAULT NULL;
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `template` varchar(24) DEFAULT NULL;
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `expires_at` timestamp NULL DEFAULT NULL;
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `urgent_sms` tinyint(1) NOT NULL DEFAULT 0;
ALTER TABLE `companions` ADD COLUMN IF NOT EXISTS `owner_confirmed_at` timestamp NULL DEFAULT NULL;
CREATE INDEX IF NOT EXISTS `companions_companion_user_id_status_owner_id_index` ON `companions` (`companion_user_id`,`status`,`owner_id`);
CREATE INDEX IF NOT EXISTS `companions_owner_id_companion_user_id_status_index` ON `companions` (`owner_id`,`companion_user_id`,`status`);
CREATE INDEX IF NOT EXISTS `companions_status_expires_at_index` ON `companions` (`status`,`expires_at`);
DROP INDEX IF EXISTS `companions_companion_user_id_status_index` ON `companions`;

ALTER TABLE `companion_grants` ADD COLUMN IF NOT EXISTS `mode_epoch` int(10) unsigned DEFAULT NULL;

CREATE TABLE IF NOT EXISTS `family_owner_state` (
  `owner_id` bigint(20) unsigned NOT NULL,
  `mode` varchar(16) NOT NULL,
  `mode_epoch` int(10) unsigned NOT NULL DEFAULT 1,
  `review_pending` tinyint(1) NOT NULL DEFAULT 0,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`owner_id`),
  CONSTRAINT `family_owner_state_owner_id_foreign` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `companion_audit_logs` ADD COLUMN IF NOT EXISTS `subject_type` varchar(16) DEFAULT NULL;
ALTER TABLE `companion_audit_logs` ADD COLUMN IF NOT EXISTS `subject_id` bigint(20) unsigned DEFAULT NULL;
ALTER TABLE `companion_audit_logs` ADD COLUMN IF NOT EXISTS `diff_json` json DEFAULT NULL;
ALTER TABLE `companion_audit_logs` ADD COLUMN IF NOT EXISTS `payload_sealed` mediumblob DEFAULT NULL;
ALTER TABLE `companion_audit_logs` ADD COLUMN IF NOT EXISTS `payload_key_id` tinyint(3) unsigned DEFAULT NULL;
ALTER TABLE `companion_audit_logs` ADD COLUMN IF NOT EXISTS `undo_deadline` timestamp NULL DEFAULT NULL;
ALTER TABLE `companion_audit_logs` ADD COLUMN IF NOT EXISTS `undone_at` timestamp NULL DEFAULT NULL;
ALTER TABLE `companion_audit_logs` ADD COLUMN IF NOT EXISTS `undone_by` bigint(20) unsigned DEFAULT NULL;
ALTER TABLE `companion_audit_logs` ADD COLUMN IF NOT EXISTS `proposal_id` bigint(20) unsigned DEFAULT NULL;
CREATE INDEX IF NOT EXISTS `companion_audit_logs_subject_type_subject_id_created_at_index` ON `companion_audit_logs` (`subject_type`,`subject_id`,`created_at`);
CREATE INDEX IF NOT EXISTS `companion_audit_logs_undo_deadline_index` ON `companion_audit_logs` (`undo_deadline`);
CREATE INDEX IF NOT EXISTS `companion_audit_logs_created_at_index` ON `companion_audit_logs` (`created_at`);

UPDATE `companions`
SET `role` = CASE `type` WHEN 'partner' THEN 'partner' WHEN 'spouse' THEN 'partner' WHEN 'parent' THEN 'teen_parent' END
WHERE `role` IS NULL AND `type` IN ('partner', 'spouse', 'parent');

UPDATE `companions` SET `template` = 'legacy' WHERE `template` IS NULL;

-- +goose Down
DROP INDEX IF EXISTS `companion_audit_logs_created_at_index` ON `companion_audit_logs`;
DROP INDEX IF EXISTS `companion_audit_logs_undo_deadline_index` ON `companion_audit_logs`;
DROP INDEX IF EXISTS `companion_audit_logs_subject_type_subject_id_created_at_index` ON `companion_audit_logs`;
ALTER TABLE `companion_audit_logs` DROP COLUMN IF EXISTS `proposal_id`;
ALTER TABLE `companion_audit_logs` DROP COLUMN IF EXISTS `undone_by`;
ALTER TABLE `companion_audit_logs` DROP COLUMN IF EXISTS `undone_at`;
ALTER TABLE `companion_audit_logs` DROP COLUMN IF EXISTS `undo_deadline`;
ALTER TABLE `companion_audit_logs` DROP COLUMN IF EXISTS `payload_key_id`;
ALTER TABLE `companion_audit_logs` DROP COLUMN IF EXISTS `payload_sealed`;
ALTER TABLE `companion_audit_logs` DROP COLUMN IF EXISTS `diff_json`;
ALTER TABLE `companion_audit_logs` DROP COLUMN IF EXISTS `subject_id`;
ALTER TABLE `companion_audit_logs` DROP COLUMN IF EXISTS `subject_type`;

DROP TABLE IF EXISTS `family_owner_state`;

ALTER TABLE `companion_grants` DROP COLUMN IF EXISTS `mode_epoch`;

-- Data is left as it is: a v6 status (awaiting_owner) or revoked_by value (expired | declined | system) written after
-- Up is fail-closed for the v1 code (access needs status = 'active'; anything but 'revoked' counts as an open link).
CREATE INDEX IF NOT EXISTS `companions_companion_user_id_status_index` ON `companions` (`companion_user_id`,`status`);
DROP INDEX IF EXISTS `companions_status_expires_at_index` ON `companions`;
DROP INDEX IF EXISTS `companions_owner_id_companion_user_id_status_index` ON `companions`;
DROP INDEX IF EXISTS `companions_companion_user_id_status_owner_id_index` ON `companions`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `owner_confirmed_at`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `urgent_sms`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `expires_at`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `template`;
ALTER TABLE `companions` DROP COLUMN IF EXISTS `role`;
