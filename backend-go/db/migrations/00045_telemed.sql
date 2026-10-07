-- 00045_telemed.sql — doctors directory «پزشکان و ماماها» (bloom B-N7-02, D-67; artboards nbl_v17_Doctors /
-- nbl_v17_DoctorProfile in d-doctor-assistant). Admin-managed profiles, visit types, weekly availability, time off and
-- reviews. Domain logic: internal/telemed (slot generation in slots.go). Booking (B-N7-03), chat (B-N7-04) and the admin
-- console (B-N7-08) build on these tables.
--
-- telemed_doctors           one doctor or midwife (kind doctor | midwife, validated in Go). name / headline / bio are
--                           translatable JSON (keys = language codes, only the default language required). specialty,
--                           city = catalog_items codes of the groups telemed_specialties / telemed_cities (labels are
--                           admin content, not code). licence_no = the medical / midwifery council number. photo_path =
--                           the public disk (STORAGE_PATH/app/public/doctors/…). response_minutes = the «پاسخ‌گویی»
--                           figure (admin-set until B-N7-04 measures it). visits_count = completed visits (B-N7-03
--                           increments it). rating_sum / rating_count / positive_count = the visible reviews (count of
--                           ratings ≥ 4 → «رضایت» percent), kept in step with telemed_reviews in one transaction.
--                           admin_id = the admin account that answers as this doctor (B-N7-08 `doctor` role).
-- telemed_doctor_insurers   the insurers a doctor accepts: catalog_items codes of the group telemed_insurers.
-- telemed_visit_types       per doctor at most one per mode (video | phone | in_person): duration in minutes, price in
--                           integer rials (the payments adapter currency; clients show toman = rials / 10), an optional
--                           translatable note («با نسخه الکترونیک», «شنبه تا چهارشنبه») and address (in_person).
-- telemed_availability_rules weekly windows in Tehran wall-clock: weekday (Go time.Weekday: 0 = Sunday … 6 = Saturday),
--                           start_minute / end_minute (minutes after midnight, end exclusive), slot_minutes = the step
--                           of the slot grid; modes NULL = every mode the doctor offers, else a JSON list of modes.
-- telemed_time_off          absences (Tehran wall-clock datetimes, end exclusive): no slot overlaps one.
-- telemed_reviews           one review per completed visit (booking_id unique; NULL until B-N7-03 supplies bookings):
--                           rating 1–5, optional plain-text body, is_visible (admins hide abusive ones). Reviewers are
--                           shown by the first letter of their name only.
--
-- catalog_items telemed_specialties (fa, en): the common specialties as a starting list, editable / extendable in the
-- admin catalog; cities and insurers are left to the admins (no placeholder data). No doctor rows are seeded.
--
-- Twin of backend/database/migrations/2026_10_07_000045_create_telemed_tables.php (Laravel owns the prod schema until
-- T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables;
-- IF NOT EXISTS / INSERT IGNORE so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `telemed_doctors` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `kind` varchar(12) NOT NULL,
  `name` json NOT NULL,
  `headline` json DEFAULT NULL,
  `bio` json DEFAULT NULL,
  `specialty` varchar(64) NOT NULL,
  `city` varchar(64) DEFAULT NULL,
  `licence_no` varchar(32) NOT NULL,
  `experience_years` tinyint(3) unsigned DEFAULT NULL,
  `photo_path` varchar(255) DEFAULT NULL,
  `response_minutes` smallint(5) unsigned DEFAULT NULL,
  `visits_count` int(10) unsigned NOT NULL DEFAULT 0,
  `rating_sum` int(10) unsigned NOT NULL DEFAULT 0,
  `rating_count` int(10) unsigned NOT NULL DEFAULT 0,
  `positive_count` int(10) unsigned NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `admin_id` bigint(20) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `telemed_doctors_admin_id_unique` (`admin_id`),
  KEY `telemed_doctors_is_active_sort_order_index` (`is_active`,`sort_order`),
  KEY `telemed_doctors_specialty_index` (`specialty`),
  KEY `telemed_doctors_city_index` (`city`),
  CONSTRAINT `telemed_doctors_admin_id_foreign` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `telemed_doctor_insurers` (
  `doctor_id` bigint(20) unsigned NOT NULL,
  `insurer` varchar(64) NOT NULL,
  PRIMARY KEY (`doctor_id`,`insurer`),
  KEY `telemed_doctor_insurers_insurer_index` (`insurer`),
  CONSTRAINT `telemed_doctor_insurers_doctor_id_foreign` FOREIGN KEY (`doctor_id`) REFERENCES `telemed_doctors` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `telemed_visit_types` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `doctor_id` bigint(20) unsigned NOT NULL,
  `mode` varchar(12) NOT NULL,
  `duration_minutes` smallint(5) unsigned NOT NULL,
  `price_rials` bigint(20) unsigned NOT NULL,
  `note` json DEFAULT NULL,
  `address` json DEFAULT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `telemed_visit_types_doctor_id_mode_unique` (`doctor_id`,`mode`),
  CONSTRAINT `telemed_visit_types_doctor_id_foreign` FOREIGN KEY (`doctor_id`) REFERENCES `telemed_doctors` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `telemed_availability_rules` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `doctor_id` bigint(20) unsigned NOT NULL,
  `weekday` tinyint(3) unsigned NOT NULL,
  `start_minute` smallint(5) unsigned NOT NULL,
  `end_minute` smallint(5) unsigned NOT NULL,
  `slot_minutes` smallint(5) unsigned NOT NULL,
  `modes` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `telemed_availability_rules_doctor_id_weekday_index` (`doctor_id`,`weekday`),
  CONSTRAINT `telemed_availability_rules_doctor_id_foreign` FOREIGN KEY (`doctor_id`) REFERENCES `telemed_doctors` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `telemed_time_off` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `doctor_id` bigint(20) unsigned NOT NULL,
  `starts_at` datetime NOT NULL,
  `ends_at` datetime NOT NULL,
  `note` varchar(190) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `telemed_time_off_doctor_id_ends_at_index` (`doctor_id`,`ends_at`),
  CONSTRAINT `telemed_time_off_doctor_id_foreign` FOREIGN KEY (`doctor_id`) REFERENCES `telemed_doctors` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `telemed_reviews` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `doctor_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `booking_id` bigint(20) unsigned DEFAULT NULL,
  `rating` tinyint(3) unsigned NOT NULL,
  `body` varchar(1000) DEFAULT NULL,
  `is_visible` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `telemed_reviews_booking_id_unique` (`booking_id`),
  KEY `telemed_reviews_doctor_id_is_visible_created_at_index` (`doctor_id`,`is_visible`,`created_at`),
  KEY `telemed_reviews_user_id_index` (`user_id`),
  CONSTRAINT `telemed_reviews_doctor_id_foreign` FOREIGN KEY (`doctor_id`) REFERENCES `telemed_doctors` (`id`) ON DELETE CASCADE,
  CONSTRAINT `telemed_reviews_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('telemed_specialties', 'gynecology', 1, 1, NULL, '{"fa":"زنان و زایمان","en":"Obstetrics & gynaecology"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('telemed_specialties', 'midwifery', 2, 1, NULL, '{"fa":"ماما","en":"Midwife"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('telemed_specialties', 'dermatology', 3, 1, NULL, '{"fa":"پوست","en":"Dermatology"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('telemed_specialties', 'nutrition', 4, 1, NULL, '{"fa":"تغذیه","en":"Nutrition"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('telemed_specialties', 'psychology', 5, 1, NULL, '{"fa":"روان‌شناسی","en":"Psychology"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('telemed_specialties', 'general', 6, 1, NULL, '{"fa":"پزشک عمومی","en":"General practice"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
-- Removes the seeded specialties only while nobody edited them (updated_at = created_at), like 00043's copy rows.
DELETE FROM `catalog_items`
 WHERE `group` = 'telemed_specialties'
   AND `code` IN ('gynecology', 'midwifery', 'dermatology', 'nutrition', 'psychology', 'general')
   AND `updated_at` <=> `created_at`;
DROP TABLE IF EXISTS `telemed_reviews`;
DROP TABLE IF EXISTS `telemed_time_off`;
DROP TABLE IF EXISTS `telemed_availability_rules`;
DROP TABLE IF EXISTS `telemed_visit_types`;
DROP TABLE IF EXISTS `telemed_doctor_insurers`;
DROP TABLE IF EXISTS `telemed_doctors`;
