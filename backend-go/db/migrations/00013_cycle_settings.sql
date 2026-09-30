-- 00013_cycle_settings.sql — cycle settings & reminder preferences (B-N1-09, bloom/N1).
--
--  1. `notification_preferences.schedule` — when each cycle reminder fires, extending the B-N1-11 row (the
--     category switches stay in `categories`, a new `pill` category included):
--     {"before_period":{"days_before":2,"time":"09:00"},"pms":{"time":"09:00"},"fertile_window":{"time":"09:00"},
--      "daily_log":{"time":"22:00"},"pill":{"time":"21:00"}}. Only explicit choices are stored; a missing key falls
--     back to the defaults in internal/notifications (DefaultSchedule()), so NULL = all defaults.
--  2. `cycle_preferences` — one row per user once the cycle settings are saved: `lengths_auto` = «خودکار از
--     داده‌ها» (1: the engine's medians; 0: the manual cycle_duration / period_duration of user_profiles).
--     No row = automatic.
--
-- Twin of backend/database/migrations/2026_10_01_000004_create_cycle_settings.php (Laravel owns the prod schema
-- until T-M2-27; schema only, no models/routes), so `make schema-diff` stays green. Spelled the way mariadb-dump
-- prints the Laravel-built tables. IF NOT EXISTS: a database whose Laravel half already ran must not fail here.

-- +goose Up
ALTER TABLE `notification_preferences`
  ADD COLUMN IF NOT EXISTS `schedule` json DEFAULT NULL AFTER `categories`;

CREATE TABLE IF NOT EXISTS `cycle_preferences` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `lengths_auto` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `cycle_preferences_user_id_unique` (`user_id`),
  CONSTRAINT `cycle_preferences_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `cycle_preferences`;
ALTER TABLE `notification_preferences` DROP COLUMN IF EXISTS `schedule`;
