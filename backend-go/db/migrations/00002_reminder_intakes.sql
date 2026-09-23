-- 00002_reminder_intakes.sql — one row per medication dose taken (care reminders, T-M3-01).
--
-- Twin of backend/database/migrations/2026_09_23_000001_create_reminder_intakes_table.php
-- (Laravel owns the prod schema until T-M2-27); `make schema-diff` proves both build the same table.
-- Spelled the way mariadb-dump prints the Laravel-built table. IF NOT EXISTS: a database whose
-- Laravel half already created the table (prod at cutover, T-M2-27: baseline stamped, then
-- migrations after 00001 applied) must not fail here.

-- +goose Up
CREATE TABLE IF NOT EXISTS `reminder_intakes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `reminder_id` bigint(20) unsigned NOT NULL,
  `intake_date` date NOT NULL,
  `slot` char(5) NOT NULL,
  `taken_at` datetime NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `reminder_intakes_reminder_id_intake_date_slot_unique` (`reminder_id`,`intake_date`,`slot`),
  KEY `reminder_intakes_user_id_intake_date_index` (`user_id`,`intake_date`),
  CONSTRAINT `reminder_intakes_reminder_id_foreign` FOREIGN KEY (`reminder_id`) REFERENCES `reminders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `reminder_intakes_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `reminder_intakes`;
