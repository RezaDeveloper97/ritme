-- 00099_health_targets_confirmation_test_kind.sql — storage for three M10 health tasks (T-M10-17; SPEC v6 §3.2–§3.4,
-- docs/engine-v6/DECISIONS.md H-08, H-10, H-11). Schema plus one backfill; no behaviour reads the new columns yet
-- (T-M10-19, T-M10-20, T-M10-21).
--
-- vital_targets    one row per (user, glucose context: fasting | pre_meal | post_meal | bedtime): the diabetic
--                  glucose target range in mg/dL (low/high, either may be open) and who set it (doctor | user), with
--                  an optional note. Health data, user-scoped, cascades with the user. The diabetes flag itself is
--                  user_life_profiles.chronic_illnesses (no column here).
-- lab_markers      +confirmed_at   when the user confirmed this value (H-08: low-confidence values are confirmed one
--                                  by one). Backfill: every marker of a report with verified_at gets
--                                  confirmed_at = verified_at (today's all-or-nothing confirmation); markers of
--                                  unverified reports stay NULL.
-- checkup_records  +test_kind      the kind of a cervical screening test: cytology | hpv | cotest (H-10). NULL = not
--                                  a cervical test / not given.
-- checkup_types    +informational  1 = an informational catalog item without a schedule (H-11). Existing rows read 0.
--
-- The new columns are appended (no AFTER), so the table order is the order sqlc generates the row structs in.
--
-- Safe on populated tables: nullable columns / a NOT NULL column with a constant default (MariaDB adds them in place),
-- one new empty table and an UPDATE that only fills NULL confirmed_at values. IF [NOT] EXISTS everywhere, so a
-- database whose Laravel twin already ran is a no-op and a partial run can run again. Down drops what Up added (the
-- targets, confirmations, test kinds and informational flags stored since are lost; nothing older is touched).
--
-- Twin of backend/database/migrations/2026_10_21_000099_add_health_targets_confirmation_test_kind.php (Laravel owns
-- the prod schema until T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the
-- Laravel-built tables.

-- +goose Up
CREATE TABLE IF NOT EXISTS `vital_targets` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `context` varchar(16) NOT NULL,
  `low_mgdl` smallint(5) unsigned DEFAULT NULL,
  `high_mgdl` smallint(5) unsigned DEFAULT NULL,
  `set_by` varchar(8) NOT NULL,
  `note` varchar(160) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `vital_targets_user_id_context_unique` (`user_id`,`context`),
  CONSTRAINT `vital_targets_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `lab_markers` ADD COLUMN IF NOT EXISTS `confirmed_at` timestamp NULL DEFAULT NULL;

UPDATE `lab_markers` m
JOIN `lab_reports` r ON r.id = m.lab_id
SET m.confirmed_at = r.verified_at
WHERE r.verified_at IS NOT NULL AND m.confirmed_at IS NULL;

ALTER TABLE `checkup_records` ADD COLUMN IF NOT EXISTS `test_kind` varchar(12) DEFAULT NULL;

ALTER TABLE `checkup_types` ADD COLUMN IF NOT EXISTS `informational` tinyint(1) NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE `checkup_types` DROP COLUMN IF EXISTS `informational`;
ALTER TABLE `checkup_records` DROP COLUMN IF EXISTS `test_kind`;
ALTER TABLE `lab_markers` DROP COLUMN IF EXISTS `confirmed_at`;
DROP TABLE IF EXISTS `vital_targets`;
