-- 00097_postpartum_recovery_logs.sql — postpartum red flags and the postpartum visit log (T-M9-06, D-135; SPEC v6
-- §3.1). The recovery log itself has no table (00030: it is log taxonomy v2 slots in health_log_entries), so the four
-- red-flag answers of the recovery sheet and «I had my postpartum visit today» get their own per-day row here:
--
-- postpartum_recovery_logs   one row per (user, day): temperature_c (°C, one decimal), pad_soaked_within_hour,
--                            severe_pain, breathless (NULL = not answered) and postpartum_visit (1 = she had a
--                            postpartum check-up that day; the visits card of GET /postpartum marks the contact /
--                            6-week / comprehensive visit done when such a day falls in its window). Health data:
--                            user-scoped, never logged, not shared with companions.
--
-- Twin of backend/database/migrations/2026_10_21_000097_create_postpartum_recovery_logs_table.php (Laravel owns the
-- prod schema until T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built
-- table; IF NOT EXISTS so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `postpartum_recovery_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `temperature_c` decimal(3,1) DEFAULT NULL,
  `pad_soaked_within_hour` tinyint(1) DEFAULT NULL,
  `severe_pain` tinyint(1) DEFAULT NULL,
  `breathless` tinyint(1) DEFAULT NULL,
  `postpartum_visit` tinyint(1) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `postpartum_recovery_logs_user_id_log_date_unique` (`user_id`,`log_date`),
  CONSTRAINT `postpartum_recovery_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `postpartum_recovery_logs`;
