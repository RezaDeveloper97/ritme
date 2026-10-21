-- 00096_user_profiles_user_id_unique.sql — one profile per user (T-M9-23, D-127, DECISIONS.md I-09, gap 02 X1).
--
-- `user_profiles` had only the non-unique FK index on user_id and the first save was a plain INSERT, so two concurrent
-- first saves could leave two rows for one user. Readers took the lowest id; some writers updated every row of the
-- user, others one id. This migration dedupes, then adds UNIQUE(user_id); the first save is now
-- INSERT … ON DUPLICATE KEY UPDATE (db/queries/profile/profile.sql InsertProfile).
--
-- Dedupe, for every user_id with more than one row (deterministic: ids only, no clock or row order):
--   * kept:    the row with the lowest id (the one every reader already returned);
--   * backup:  every other row is copied as is into `user_profiles_dedupe_backup` (same columns + removed_at);
--   * merge:   each NULL column of the kept row is filled from the newest removed row (highest id) that has a value for
--              it. NOT NULL columns (user_goal, subscription_type, calculation_status, calculation_version) keep the
--              kept row's value, and nothing else of the kept row changes (updated_at included);
--   * delete:  the removed rows (only ids that are in the backup table are deleted).
-- Then UNIQUE KEY user_profiles_user_id_unique (user_id) is added and the old KEY user_profiles_user_id_foreign is
-- dropped (the unique key serves the foreign key). A clean table gets the keys and an empty backup table.
--
-- Re-run safe: every step can run again after a partial run (MariaDB commits each DDL statement). The backup table is
-- IF NOT EXISTS with the original id as its primary key (INSERT IGNORE keeps the first copy and its removed_at), the
-- merge only fills NULLs, the delete only removes backed-up ids, and the index changes use IF [NOT] EXISTS.
--
-- Down: re-adds the non-unique FK index, drops the unique key, puts every backed-up row back with its original id
-- (skipped when the id exists again or the user has been deleted since), then drops the backup table. The NULLs the
-- merge filled on the kept row are not emptied again (that row keeps the merged values).
--
-- Twin of backend/database/migrations/2026_10_21_000096_add_unique_user_id_to_user_profiles.php (same statements), so
-- `make schema-diff` stays green. Timestamps are Tehran wall-clock (the session time_zone is never set).

-- +goose Up
-- The backup table is an operational record, not app schema: it is created through PREPARE so sqlc (which parses
-- the Up sections) never generates a model for it in every store package. No Go code reads it.
PREPARE `user_profiles_backup` FROM 'CREATE TABLE IF NOT EXISTS `user_profiles_dedupe_backup` (
  `id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `birthday` date DEFAULT NULL,
  `weight` decimal(5,2) DEFAULT NULL,
  `height` smallint(5) unsigned DEFAULT NULL,
  `period_duration` tinyint(3) unsigned DEFAULT NULL,
  `cycle_duration` tinyint(3) unsigned DEFAULT NULL,
  `last_period_start` date DEFAULT NULL,
  `user_goal` varchar(255) NOT NULL DEFAULT ''non_ttc'',
  `pregnancy_intention` varchar(255) DEFAULT NULL,
  `chronic_conditions` json DEFAULT NULL,
  `subscription_type` varchar(255) NOT NULL DEFAULT ''free'',
  `calculation_status` varchar(255) NOT NULL DEFAULT ''pending'',
  `calculation_started_at` timestamp NULL DEFAULT NULL,
  `calculation_completed_at` timestamp NULL DEFAULT NULL,
  `calculation_version` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `removed_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_profiles_dedupe_backup_user_id_index` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci';
EXECUTE `user_profiles_backup`;
DEALLOCATE PREPARE `user_profiles_backup`;

INSERT IGNORE INTO `user_profiles_dedupe_backup` (`id`, `user_id`, `birthday`, `weight`, `height`, `period_duration`,
  `cycle_duration`, `last_period_start`, `user_goal`, `pregnancy_intention`, `chronic_conditions`, `subscription_type`,
  `calculation_status`, `calculation_started_at`, `calculation_completed_at`, `calculation_version`, `created_at`,
  `updated_at`, `removed_at`)
SELECT p.`id`, p.`user_id`, p.`birthday`, p.`weight`, p.`height`, p.`period_duration`, p.`cycle_duration`,
  p.`last_period_start`, p.`user_goal`, p.`pregnancy_intention`, p.`chronic_conditions`, p.`subscription_type`,
  p.`calculation_status`, p.`calculation_started_at`, p.`calculation_completed_at`, p.`calculation_version`,
  p.`created_at`, p.`updated_at`, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
FROM `user_profiles` p
JOIN (SELECT `user_id`, MIN(`id`) AS `kept_id` FROM `user_profiles` GROUP BY `user_id` HAVING COUNT(*) > 1) d
  ON d.`user_id` = p.`user_id` AND p.`id` <> d.`kept_id`
ORDER BY p.`id`;

UPDATE `user_profiles` k SET
  k.`birthday` = COALESCE(k.`birthday`, (SELECT b.`birthday` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`birthday` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`weight` = COALESCE(k.`weight`, (SELECT b.`weight` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`weight` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`height` = COALESCE(k.`height`, (SELECT b.`height` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`height` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`period_duration` = COALESCE(k.`period_duration`, (SELECT b.`period_duration` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`period_duration` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`cycle_duration` = COALESCE(k.`cycle_duration`, (SELECT b.`cycle_duration` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`cycle_duration` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`last_period_start` = COALESCE(k.`last_period_start`, (SELECT b.`last_period_start` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`last_period_start` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`pregnancy_intention` = COALESCE(k.`pregnancy_intention`, (SELECT b.`pregnancy_intention` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`pregnancy_intention` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`chronic_conditions` = COALESCE(k.`chronic_conditions`, (SELECT b.`chronic_conditions` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`chronic_conditions` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`calculation_started_at` = COALESCE(k.`calculation_started_at`, (SELECT b.`calculation_started_at` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`calculation_started_at` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`calculation_completed_at` = COALESCE(k.`calculation_completed_at`, (SELECT b.`calculation_completed_at` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`calculation_completed_at` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`created_at` = COALESCE(k.`created_at`, (SELECT b.`created_at` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`created_at` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1)),
  k.`updated_at` = COALESCE(k.`updated_at`, (SELECT b.`updated_at` FROM `user_profiles_dedupe_backup` b
    WHERE b.`user_id` = k.`user_id` AND b.`id` > k.`id` AND b.`updated_at` IS NOT NULL ORDER BY b.`id` DESC LIMIT 1))
WHERE k.`id` NOT IN (SELECT `id` FROM `user_profiles_dedupe_backup`)
  AND k.`user_id` IN (SELECT `user_id` FROM `user_profiles_dedupe_backup`);

DELETE FROM `user_profiles` WHERE `id` IN (SELECT `id` FROM `user_profiles_dedupe_backup`);

-- Prepared so sqlc (which parses the Up sections) does not choke on MariaDB's IF [NOT] EXISTS index clauses.
PREPARE `user_profiles_keys` FROM 'ALTER TABLE `user_profiles` ADD UNIQUE KEY IF NOT EXISTS `user_profiles_user_id_unique` (`user_id`)';
EXECUTE `user_profiles_keys`;
PREPARE `user_profiles_keys` FROM 'ALTER TABLE `user_profiles` DROP INDEX IF EXISTS `user_profiles_user_id_foreign`';
EXECUTE `user_profiles_keys`;
DEALLOCATE PREPARE `user_profiles_keys`;

-- +goose Down
ALTER TABLE `user_profiles` ADD KEY IF NOT EXISTS `user_profiles_user_id_foreign` (`user_id`);

ALTER TABLE `user_profiles` DROP INDEX IF EXISTS `user_profiles_user_id_unique`;

INSERT INTO `user_profiles` (`id`, `user_id`, `birthday`, `weight`, `height`, `period_duration`, `cycle_duration`,
  `last_period_start`, `user_goal`, `pregnancy_intention`, `chronic_conditions`, `subscription_type`,
  `calculation_status`, `calculation_started_at`, `calculation_completed_at`, `calculation_version`, `created_at`,
  `updated_at`)
SELECT b.`id`, b.`user_id`, b.`birthday`, b.`weight`, b.`height`, b.`period_duration`, b.`cycle_duration`,
  b.`last_period_start`, b.`user_goal`, b.`pregnancy_intention`, b.`chronic_conditions`, b.`subscription_type`,
  b.`calculation_status`, b.`calculation_started_at`, b.`calculation_completed_at`, b.`calculation_version`,
  b.`created_at`, b.`updated_at`
FROM `user_profiles_dedupe_backup` b
JOIN `users` u ON u.`id` = b.`user_id`
WHERE b.`id` NOT IN (SELECT `id` FROM `user_profiles`)
ORDER BY b.`id`;

DROP TABLE IF EXISTS `user_profiles_dedupe_backup`;
