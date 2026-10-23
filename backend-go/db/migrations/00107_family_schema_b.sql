-- 00107_family_schema_b.sql — family schema v6, part B: proposals, share requests, nudges, member alerts, member
-- notices, appointment attendance, and the link column of user_notifications (T-M11-03; SPEC v6 §1.7;
-- docs/engine-v6/family.md §6–§8; gap 05 §1 rows 9, 13–15, 21, 23, §5.4, §5.5; DECISIONS.md F-06, F-09, F-12;
-- deviation D-191). Schema only: every writer and reader is a later task (T-M11-08 … T-M11-13). Family consents use the
-- generic consent_events ledger (T-M10-23), so there is no family consent table.
--
-- family_proposals       a member's write that waits for the approver (the owner, or the other guardian of a child):
--                        subject_type owner | child + subject_id, kind create_med | create_appointment |
--                        update_appointment | child_delete | guardian_remove, domain (family.md §2), payload_sealed +
--                        payload_key_id = the AES-256-GCM sealed proposed record (never plaintext health data), status
--                        pending | approved | rejected | expired | cancelled, expires_at = created + 7 days (hourly sweep
--                        and lazy expiry on read / approve), decided_by / decided_at, applied_ref = the record the
--                        approval created (`reminder:<id>` …). link_id is NULL for a co-guardian proposal without a
--                        link. `pending_key` = dedupe_key while pending, NULL otherwise, so UNIQUE (pending_key) allows
--                        one pending proposal per dedupe key (a second child_delete for the same child fails) and any
--                        number of decided ones (NULLs never collide); a NULL dedupe_key never dedupes.
--                        KEYs: (approver_id, status, created_at) the approver's pending list, (actor_id, status,
--                        created_at) the member's own list and the quota count, (status, expires_at) the sweep.
-- family_share_requests  a member asks the owner for more access (row 9): domains JSON (requested domain → level);
--                        nothing applies until the owner accepts. status pending | accepted | declined | expired.
--                        KEYs: (to_user_id, status, created_at) the owner's inbox, (from_user_id, to_user_id,
--                        created_at) the per-pair throttle.
-- family_nudges          one sent nudge (§6, F-06): target_type dose | appointment + target_id (the reminder), local_date
--                        = the owner-local day, slot = the dose time HH:MM ('' for an appointment, never NULL so the
--                        key always applies). UNIQUE (owner_id, target_type, target_id, local_date, slot) = 1 per dose /
--                        appointment and idempotent against double taps; (member_id, owner_id, sent_at) for the 3 per
--                        local day per pair count; (owner_id, sent_at) for the owner's history and the KPI join.
-- family_member_alerts   the neutral member copy of an owner's urgent / follow-up alert (row 21, F-09): source_alert_id
--                        + source_hit (the source row's hit_count, so a re-fired alert relays again) are UNIQUE per
--                        member (idempotent fan-out job). No medical detail is stored, only the level. seen_at, and
--                        sms_sent_at for the optional owner-controlled neutral SMS.
-- family_member_notices  the member notice outbox (row 23): kind, dedupe_key UNIQUE per member, due_at, sent_at (NULL
--                        = not sent yet), deferred_reason (quiet hours …), notification_id = the user_notifications row
--                        it produced. KEY (sent_at, due_at) for the dispatcher, (link_id) for the purge.
-- family_attendance      «میام» on the owner's appointment: reminder_id → reminders (appointments are reminder rows,
--                        type appointment), UNIQUE per member.
-- user_notifications     + link_id (FK companions, cascade) so a link's member notices are purged by an indexed key
--                        instead of JSON scans (FAM-L4). NULL for every non-family row.
--
-- Every link_id is an FK to companions ON DELETE CASCADE: hard-deleting a link (the owner's account deletion) removes
-- its proposals, share requests, nudges, member alerts, notices and linked notifications. A revoke keeps the row
-- (status = revoked), so the revoke path purges by link_id with the statements in db/queries/companion.
--
-- Twin of backend/database/migrations/2026_10_23_000107_add_family_schema_b.php (Laravel owns the prod schema until
-- T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables; IF [NOT]
-- EXISTS everywhere so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `family_proposals` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `subject_type` varchar(8) NOT NULL,
  `subject_id` bigint(20) unsigned NOT NULL,
  `link_id` bigint(20) unsigned DEFAULT NULL,
  `actor_id` bigint(20) unsigned NOT NULL,
  `approver_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(24) NOT NULL,
  `domain` varchar(24) NOT NULL,
  `payload_sealed` mediumblob DEFAULT NULL,
  `payload_key_id` tinyint(3) unsigned DEFAULT NULL,
  `status` varchar(12) NOT NULL DEFAULT 'pending',
  `dedupe_key` varchar(80) DEFAULT NULL,
  `pending_key` varchar(80) GENERATED ALWAYS AS (if(`status` = 'pending',`dedupe_key`,NULL)) STORED,
  `expires_at` timestamp NOT NULL,
  `decided_by` bigint(20) unsigned DEFAULT NULL,
  `decided_at` timestamp NULL DEFAULT NULL,
  `applied_ref` varchar(64) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `family_proposals_pending_key_unique` (`pending_key`),
  KEY `family_proposals_approver_id_status_created_at_index` (`approver_id`,`status`,`created_at`),
  KEY `family_proposals_actor_id_status_created_at_index` (`actor_id`,`status`,`created_at`),
  KEY `family_proposals_status_expires_at_index` (`status`,`expires_at`),
  KEY `family_proposals_link_id_foreign` (`link_id`),
  CONSTRAINT `family_proposals_actor_id_foreign` FOREIGN KEY (`actor_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_proposals_approver_id_foreign` FOREIGN KEY (`approver_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_proposals_link_id_foreign` FOREIGN KEY (`link_id`) REFERENCES `companions` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `family_share_requests` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `from_user_id` bigint(20) unsigned NOT NULL,
  `to_user_id` bigint(20) unsigned NOT NULL,
  `link_id` bigint(20) unsigned DEFAULT NULL,
  `domains` json NOT NULL,
  `status` varchar(10) NOT NULL DEFAULT 'pending',
  `expires_at` timestamp NOT NULL,
  `decided_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `family_share_requests_to_user_id_status_created_at_index` (`to_user_id`,`status`,`created_at`),
  KEY `family_share_requests_from_user_id_to_user_id_created_at_index` (`from_user_id`,`to_user_id`,`created_at`),
  KEY `family_share_requests_link_id_foreign` (`link_id`),
  CONSTRAINT `family_share_requests_from_user_id_foreign` FOREIGN KEY (`from_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_share_requests_link_id_foreign` FOREIGN KEY (`link_id`) REFERENCES `companions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_share_requests_to_user_id_foreign` FOREIGN KEY (`to_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `family_nudges` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `owner_id` bigint(20) unsigned NOT NULL,
  `member_id` bigint(20) unsigned NOT NULL,
  `link_id` bigint(20) unsigned NOT NULL,
  `target_type` varchar(12) NOT NULL,
  `target_id` bigint(20) unsigned NOT NULL,
  `local_date` date NOT NULL,
  `slot` varchar(5) NOT NULL DEFAULT '',
  `sent_at` timestamp NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `family_nudges_target_slot_unique` (`owner_id`,`target_type`,`target_id`,`local_date`,`slot`),
  KEY `family_nudges_member_id_owner_id_sent_at_index` (`member_id`,`owner_id`,`sent_at`),
  KEY `family_nudges_owner_id_sent_at_index` (`owner_id`,`sent_at`),
  KEY `family_nudges_link_id_foreign` (`link_id`),
  CONSTRAINT `family_nudges_link_id_foreign` FOREIGN KEY (`link_id`) REFERENCES `companions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_nudges_member_id_foreign` FOREIGN KEY (`member_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_nudges_owner_id_foreign` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `family_member_alerts` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `member_id` bigint(20) unsigned NOT NULL,
  `owner_id` bigint(20) unsigned NOT NULL,
  `link_id` bigint(20) unsigned NOT NULL,
  `source_alert_id` bigint(20) unsigned NOT NULL,
  `source_hit` int(10) unsigned NOT NULL,
  `level` varchar(12) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `seen_at` timestamp NULL DEFAULT NULL,
  `sms_sent_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `family_member_alerts_member_id_source_alert_id_source_hit_unique` (`member_id`,`source_alert_id`,`source_hit`),
  KEY `family_member_alerts_member_id_created_at_index` (`member_id`,`created_at`),
  KEY `family_member_alerts_link_id_index` (`link_id`),
  KEY `family_member_alerts_owner_id_foreign` (`owner_id`),
  CONSTRAINT `family_member_alerts_link_id_foreign` FOREIGN KEY (`link_id`) REFERENCES `companions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_member_alerts_member_id_foreign` FOREIGN KEY (`member_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_member_alerts_owner_id_foreign` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `family_member_notices` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `member_id` bigint(20) unsigned NOT NULL,
  `owner_id` bigint(20) unsigned NOT NULL,
  `link_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(24) NOT NULL,
  `dedupe_key` varchar(80) NOT NULL,
  `due_at` timestamp NOT NULL,
  `sent_at` timestamp NULL DEFAULT NULL,
  `deferred_reason` varchar(24) DEFAULT NULL,
  `notification_id` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `family_member_notices_member_id_dedupe_key_unique` (`member_id`,`dedupe_key`),
  KEY `family_member_notices_sent_at_due_at_index` (`sent_at`,`due_at`),
  KEY `family_member_notices_link_id_index` (`link_id`),
  KEY `family_member_notices_owner_id_foreign` (`owner_id`),
  CONSTRAINT `family_member_notices_link_id_foreign` FOREIGN KEY (`link_id`) REFERENCES `companions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_member_notices_member_id_foreign` FOREIGN KEY (`member_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_member_notices_owner_id_foreign` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `family_attendance` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `reminder_id` bigint(20) unsigned NOT NULL,
  `member_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `family_attendance_reminder_id_member_id_unique` (`reminder_id`,`member_id`),
  KEY `family_attendance_member_id_foreign` (`member_id`),
  CONSTRAINT `family_attendance_member_id_foreign` FOREIGN KEY (`member_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `family_attendance_reminder_id_foreign` FOREIGN KEY (`reminder_id`) REFERENCES `reminders` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `user_notifications`
  ADD COLUMN IF NOT EXISTS `link_id` bigint(20) unsigned DEFAULT NULL,
  ADD KEY IF NOT EXISTS `user_notifications_link_id_index` (`link_id`),
  ADD CONSTRAINT `user_notifications_link_id_foreign` FOREIGN KEY IF NOT EXISTS (`link_id`) REFERENCES `companions` (`id`) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE `user_notifications` DROP FOREIGN KEY IF EXISTS `user_notifications_link_id_foreign`;
ALTER TABLE `user_notifications` DROP INDEX IF EXISTS `user_notifications_link_id_index`;
ALTER TABLE `user_notifications` DROP COLUMN IF EXISTS `link_id`;

DROP TABLE IF EXISTS `family_attendance`;
DROP TABLE IF EXISTS `family_member_notices`;
DROP TABLE IF EXISTS `family_member_alerts`;
DROP TABLE IF EXISTS `family_nudges`;
DROP TABLE IF EXISTS `family_share_requests`;
DROP TABLE IF EXISTS `family_proposals`;
