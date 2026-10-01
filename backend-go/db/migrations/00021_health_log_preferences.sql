-- 00021_health_log_preferences.sql — log preferences (B-N3-02, bloom/N3, nbl_Log_Customize): per user and
-- life-stage mode the category order, the hidden categories and the pinned quick tiles, plus the user's custom
-- items. Health data: user-scoped (FK cascade on account deletion).
--
-- health_log_preferences   one row per (user, mode), written only when she customises that mode. Each JSON list
--                          is NULL until she changes it (the default applies: registry order, nothing hidden,
--                          tiles by mode and cycle phase — internal/healthlog/taxonomy/prefs.go).
--   category_order         JSON list of category codes, her order (categories she never ordered follow)
--   hidden                 JSON list of hidden category codes
--   pinned                 JSON list of ≤ 8 tile keys ("pain", "measurements.weight", …)
-- health_log_custom_items  a user-defined item inside a category's custom param (taxonomy Param.Custom: custom.items,
--                          symptoms.general, mood.moods, …). Its item code in health_log_entries is
--                          "custom_<id>"; a deleted item is soft-deleted (deleted_at) so logged days keep its label
--                          and only new input refuses it.
--
-- Twin of backend/database/migrations/2026_10_01_000021_create_health_log_preferences_tables.php (Laravel owns the
-- prod schema until T-M2-27; schema only), so `make schema-diff` stays green. Spelled the way mariadb-dump prints
-- the Laravel-built tables. IF NOT EXISTS: a database whose Laravel half already ran must not fail here.

-- +goose Up
CREATE TABLE IF NOT EXISTS `health_log_custom_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `category` varchar(32) NOT NULL,
  `param` varchar(32) NOT NULL,
  `label` varchar(40) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `health_log_custom_items_user_id_deleted_at_index` (`user_id`,`deleted_at`),
  CONSTRAINT `health_log_custom_items_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `health_log_preferences` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `mode` varchar(16) NOT NULL,
  `category_order` json DEFAULT NULL,
  `hidden` json DEFAULT NULL,
  `pinned` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `health_log_preferences_user_id_mode_unique` (`user_id`,`mode`),
  CONSTRAINT `health_log_preferences_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `health_log_preferences`;
DROP TABLE IF EXISTS `health_log_custom_items`;
