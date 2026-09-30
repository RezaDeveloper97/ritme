-- 00011_notification_preferences.sql — per-user notification settings (B-N1-11, bloom/N1): which reminder
-- categories may be pushed, a quiet-hours window and «متن خنثی» (neutral lock-screen copy). One row per user,
-- created on the first save; no row = the defaults in internal/notifications (Defaults()).
--
-- categories   JSON object {"<category code>": true|false}; only explicit choices are stored, a missing key falls
--              back to the category default, so new categories need no data migration
-- quiet_*      the window in Tehran wall-clock; start > end wraps midnight (23:00 → 08:00)
-- neutral_copy 1 = push title/body never carry health data (the default)
--
-- Twin of backend/database/migrations/2026_10_01_000002_create_notification_preferences_table.php (Laravel owns
-- the prod schema until T-M2-27; schema only, no model/routes), so `make schema-diff` stays green. Spelled the way
-- mariadb-dump prints the Laravel-built table. IF NOT EXISTS: a database whose Laravel half already created the
-- table (prod at cutover) must not fail here.

-- +goose Up
CREATE TABLE IF NOT EXISTS `notification_preferences` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `categories` json DEFAULT NULL,
  `quiet_hours_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `quiet_start` time NOT NULL DEFAULT '23:00:00',
  `quiet_end` time NOT NULL DEFAULT '08:00:00',
  `neutral_copy` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `notification_preferences_user_id_unique` (`user_id`),
  CONSTRAINT `notification_preferences_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `notification_preferences`;
