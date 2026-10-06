-- 00036_health_record.sql — «پرونده سلامت من» (bloom B-N6-03; artboard nbl_Record_Summary in c-health-record): the two
-- things the health record needs that no other domain stores. Everything else on the record is read from its owner
-- (user_profiles height/weight, user_life_profiles conditions, care medications, the cycle engine, vital_readings +
-- the log sheet, checkup_records, lab_reports, pregnancy_profiles / postpartum_profiles). Domain logic:
-- internal/healthrecord. Bloom owns 00034–00039 (00034 labs, 00035 vitals).
--
-- health_records             one row per user: blood_type A+|A-|B+|B-|AB+|AB-|O+|O- (NULL = not told; the record
--                            falls back to the pregnancy profile's blood_type + rh_factor), allergies = JSON list of
--                            short free-text items («پنی‌سیلین»; NULL = never answered, [] = «ندارم»). Roadmap
--                            CB-REC-01 extends this row (family history, surgeries) instead of adding a second one.
-- health_record_pregnancies  pregnancies / births the user adds by hand when the app did not track them: outcome
--                            vaginal|cesarean|ended (no loss type, no note — a pregnancy that ended is only «ended»),
--                            ended_on = approximate (the client sends the first day of the chosen year), baby_count
--                            1–4 for a birth.
--
-- Health data: owner-only (no companion section reads these), never logged. Codes are varchar validated in Go. Twin of
-- backend/database/migrations/2026_10_06_000036_create_health_record_tables.php (Laravel owns the prod schema until
-- T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables; IF NOT
-- EXISTS so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `health_records` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `blood_type` varchar(4) DEFAULT NULL,
  `allergies` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `health_records_user_id_unique` (`user_id`),
  CONSTRAINT `health_records_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `health_record_pregnancies` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `outcome` varchar(16) NOT NULL,
  `ended_on` date DEFAULT NULL,
  `baby_count` tinyint(3) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `health_record_pregnancies_user_id_ended_on_index` (`user_id`,`ended_on`),
  CONSTRAINT `health_record_pregnancies_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `health_record_pregnancies`;
DROP TABLE IF EXISTS `health_records`;
