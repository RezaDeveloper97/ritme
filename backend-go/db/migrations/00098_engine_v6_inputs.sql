-- 00098_engine_v6_inputs.sql — storage for the engine v6 inputs and the per-user revision (T-M9-24; SPEC v6,
-- docs/engine-v6/DECISIONS.md I-01, X-07, E-15, I-04, X-02). Schema only: no row is written or moved here (the
-- onboarding seed rows move in T-M9-27, the bleeding-log bridge rows in T-M9-28).
--
-- user_profiles        +onboarding_last_period_start  the LMP declared at onboarding (I-01). The engine's declared
--                                                     anchor; nothing but onboarding writes it. NULL = not declared.
--                      +timezone                      the user's IANA zone (X-07: X-Timezone header, stored when it
--                                                     changes). NULL = unknown → Asia/Tehran.
-- cycle_skips          one row per «پریود نداشتم» (E-15): the predicted start the skip refers to. Health data,
--                      user-scoped; deleting the row undoes the skip.
-- postpartum_profiles  +breastfeeding                 asked on the postpartum screens (I-04). NULL = not answered.
-- user_revisions       one row per user (X-02): engine_rev is bumped in every engine-input write transaction,
--                      analysis_rev in every analysis-input write. A missing row reads as 0 / 0.
--
-- The new columns are appended (no AFTER), so the table order is the order sqlc generates the row structs in, and
-- Eloquent-order serialisers (internal/profile/model Attributes) match Laravel's toArray().
--
-- Safe on populated tables: nullable columns without defaults (MariaDB adds them in place, existing rows read NULL)
-- and two new empty tables. IF [NOT] EXISTS everywhere, so a database whose Laravel twin already ran is a no-op and a
-- partial run can run again. Down drops what Up added (the values stored in the new columns and tables since are lost;
-- nothing older is touched).
--
-- Twin of backend/database/migrations/2026_10_21_000098_add_engine_v6_inputs.php (Laravel owns the prod schema until
-- T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables.

-- +goose Up
ALTER TABLE `user_profiles` ADD COLUMN IF NOT EXISTS `onboarding_last_period_start` date DEFAULT NULL;
ALTER TABLE `user_profiles` ADD COLUMN IF NOT EXISTS `timezone` varchar(64) DEFAULT NULL;

ALTER TABLE `postpartum_profiles` ADD COLUMN IF NOT EXISTS `breastfeeding` tinyint(1) DEFAULT NULL;

CREATE TABLE IF NOT EXISTS `cycle_skips` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `expected_start` date NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `cycle_skips_user_id_expected_start_unique` (`user_id`,`expected_start`),
  CONSTRAINT `cycle_skips_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `user_revisions` (
  `user_id` bigint(20) unsigned NOT NULL,
  `engine_rev` bigint(20) unsigned NOT NULL DEFAULT 0,
  `analysis_rev` bigint(20) unsigned NOT NULL DEFAULT 0,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`user_id`),
  CONSTRAINT `user_revisions_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `user_revisions`;
DROP TABLE IF EXISTS `cycle_skips`;
ALTER TABLE `postpartum_profiles` DROP COLUMN IF EXISTS `breastfeeding`;
ALTER TABLE `user_profiles` DROP COLUMN IF EXISTS `timezone`;
ALTER TABLE `user_profiles` DROP COLUMN IF EXISTS `onboarding_last_period_start`;
