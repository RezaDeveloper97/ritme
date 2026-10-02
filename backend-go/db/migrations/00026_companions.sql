-- 00026_companions.sql — companion «همدم» & family schema (bloom B-N4-01, canvas boards nbl_Hamdam_*): a woman
-- (the owner) links a partner or spouse account, decides per section what that account may see or edit, and with a
-- spouse forms a family whose children are shared. Domain logic: internal/companion (B-N4-01); HTTP: B-N4-02.
--
-- Twin of backend/database/migrations/2026_10_02_000026_create_companion_tables.php (Laravel owns the prod schema
-- until T-M2-27; it creates the same tables), so `make schema-diff` stays green. Spelled the way mariadb-dump prints
-- the Laravel-built tables. IF NOT EXISTS: a database whose Laravel half already created the tables (prod at cutover)
-- must not fail here. 00025 belongs to a parallel task.
--
-- Codes (type / status / section / level / action) are varchar, validated in Go (internal/companion), not ENUMs: the
-- canvas queue adds companion type `parent` and teen-only sections (CB-TEEN-01) and IVF / loss / record-sharing scopes
-- (CB-IVF-01, CB-LOSS-01, CB-REC-03) without an ALTER.
--
-- companions             one row per link: owner_id (the sharing user), companion_user_id (NULL until the invite is
--                        accepted), type partner|spouse, status invited|active|revoked, the display name the owner
--                        gave («اسمش»), lifecycle timestamps, revoked_by owner|companion. Revoked rows are kept (audit
--                        trail); a new invite makes a new row. At most one non-revoked spouse per owner and one
--                        non-revoked link per (owner, companion user) — enforced in Go under a per-owner row lock.
-- companion_invites      one-time invite for a companion row: a 6-char code stored only as code_hash (HMAC-SHA256,
--                        hex), optional phone (the invite is then redeemable only by the account with that mobile; a
--                        mismatch counts as a failed attempt), attempts counter (locked at 5), expires_at = created +
--                        24 h, used_at / used_by_id, revoked_at (renewed or link revoked).
-- companion_grants       explicit access per (companion, section): section cycle|symptoms|meds|appointments|pregnancy,
--                        level view|edit. No row = none (default: nothing is shared). Effective only while the
--                        companion row is active; revoking deletes the rows.
-- families               owner + spouse: created with a spouse invite (holds the shared-children choice of the
--                        Hamdam_Children step), spouse_user_id set on accept, deleted when the spouse link is revoked.
-- family_children        shared-children link (placeholder): child_id has no FK yet — the `children` table arrives in
--                        bloom B-N5-02, which adds `family_children_child_id_foreign` → children(id) ON DELETE CASCADE.
-- companion_audit_logs   who (actor) read / wrote which section of whose data (owner) and when, plus link lifecycle
--                        events (invited, accepted, revoked, grants_changed). Never a health payload. Kept when the
--                        actor or the link goes (SET NULL); removed with the owner's account.

-- +goose Up
CREATE TABLE IF NOT EXISTS `companions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `owner_id` bigint(20) unsigned NOT NULL,
  `companion_user_id` bigint(20) unsigned DEFAULT NULL,
  `type` varchar(16) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'invited',
  `display_name` varchar(100) DEFAULT NULL,
  `invited_at` timestamp NULL DEFAULT NULL,
  `accepted_at` timestamp NULL DEFAULT NULL,
  `revoked_at` timestamp NULL DEFAULT NULL,
  `revoked_by` varchar(16) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `companions_owner_id_status_index` (`owner_id`,`status`),
  KEY `companions_companion_user_id_status_index` (`companion_user_id`,`status`),
  CONSTRAINT `companions_companion_user_id_foreign` FOREIGN KEY (`companion_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `companions_owner_id_foreign` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `companion_invites` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `companion_id` bigint(20) unsigned NOT NULL,
  `owner_id` bigint(20) unsigned NOT NULL,
  `phone` varchar(11) DEFAULT NULL,
  `code_hash` char(64) NOT NULL,
  `attempts` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `expires_at` timestamp NULL DEFAULT NULL,
  `used_at` timestamp NULL DEFAULT NULL,
  `used_by_id` bigint(20) unsigned DEFAULT NULL,
  `revoked_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `companion_invites_code_hash_unique` (`code_hash`),
  KEY `companion_invites_companion_id_foreign` (`companion_id`),
  KEY `companion_invites_used_by_id_foreign` (`used_by_id`),
  KEY `companion_invites_owner_id_created_at_index` (`owner_id`,`created_at`),
  CONSTRAINT `companion_invites_companion_id_foreign` FOREIGN KEY (`companion_id`) REFERENCES `companions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `companion_invites_owner_id_foreign` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `companion_invites_used_by_id_foreign` FOREIGN KEY (`used_by_id`) REFERENCES `users` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `companion_grants` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `companion_id` bigint(20) unsigned NOT NULL,
  `section` varchar(32) NOT NULL,
  `level` varchar(8) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `companion_grants_companion_id_section_unique` (`companion_id`,`section`),
  CONSTRAINT `companion_grants_companion_id_foreign` FOREIGN KEY (`companion_id`) REFERENCES `companions` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `families` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `owner_id` bigint(20) unsigned NOT NULL,
  `spouse_user_id` bigint(20) unsigned DEFAULT NULL,
  `companion_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `families_companion_id_unique` (`companion_id`),
  KEY `families_owner_id_foreign` (`owner_id`),
  KEY `families_spouse_user_id_foreign` (`spouse_user_id`),
  CONSTRAINT `families_companion_id_foreign` FOREIGN KEY (`companion_id`) REFERENCES `companions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `families_owner_id_foreign` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `families_spouse_user_id_foreign` FOREIGN KEY (`spouse_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `family_children` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `family_id` bigint(20) unsigned NOT NULL,
  `child_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `family_children_family_id_child_id_unique` (`family_id`,`child_id`),
  KEY `family_children_child_id_index` (`child_id`),
  CONSTRAINT `family_children_family_id_foreign` FOREIGN KEY (`family_id`) REFERENCES `families` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `companion_audit_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `owner_id` bigint(20) unsigned NOT NULL,
  `actor_id` bigint(20) unsigned DEFAULT NULL,
  `companion_id` bigint(20) unsigned DEFAULT NULL,
  `section` varchar(32) DEFAULT NULL,
  `action` varchar(32) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `companion_audit_logs_actor_id_foreign` (`actor_id`),
  KEY `companion_audit_logs_companion_id_foreign` (`companion_id`),
  KEY `companion_audit_logs_owner_id_created_at_index` (`owner_id`,`created_at`),
  CONSTRAINT `companion_audit_logs_actor_id_foreign` FOREIGN KEY (`actor_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `companion_audit_logs_companion_id_foreign` FOREIGN KEY (`companion_id`) REFERENCES `companions` (`id`) ON DELETE SET NULL,
  CONSTRAINT `companion_audit_logs_owner_id_foreign` FOREIGN KEY (`owner_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `companion_audit_logs`;
DROP TABLE IF EXISTS `family_children`;
DROP TABLE IF EXISTS `families`;
DROP TABLE IF EXISTS `companion_grants`;
DROP TABLE IF EXISTS `companion_invites`;
DROP TABLE IF EXISTS `companions`;
