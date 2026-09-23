-- 00004_fertility_logs.sql — per-day fertility signals no daily_health_logs column stores: LH test strip,
-- cervical mucus and the time the basal temperature was taken (TTC tracking, T-M5-01,
-- docs/fertility-ttc/README.md). BBT, intercourse, symptoms and the note stay in daily_health_logs.
--
-- Twin of backend/database/migrations/2026_09_25_000001_create_fertility_logs_table.php
-- (Laravel owns the prod schema until T-M2-27); `make schema-diff` proves both build the same table.
-- Spelled the way mariadb-dump prints the Laravel-built table. IF NOT EXISTS: a database whose
-- Laravel half already created the table (prod at cutover, T-M2-27: baseline stamped, then
-- migrations after 00001 applied) must not fail here.

-- +goose Up
CREATE TABLE IF NOT EXISTS `fertility_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `lh_test` varchar(16) DEFAULT NULL,
  `cervical_mucus` varchar(16) DEFAULT NULL,
  `bbt_time` time DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `fertility_logs_user_id_log_date_unique` (`user_id`,`log_date`),
  CONSTRAINT `fertility_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `fertility_logs`;
