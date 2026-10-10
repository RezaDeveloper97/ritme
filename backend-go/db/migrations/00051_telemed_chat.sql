-- 00051_telemed_chat.sql — doctor chat, e-prescription and video link (bloom B-N7-04, D-82). Domain logic:
-- internal/telemed (chat*.go, prescription.go, video.go). The doctor answers from admin-web (B-N7-08) through
-- /api/admin/v1/telemed/doctor/* as the admin linked by telemed_doctors.admin_id.
--
-- telemed_messages            one chat message of a booking: sender user | doctor (validated in Go), admin_id = the
--                             admin who wrote as the doctor. body = the text sealed with PRIVATE_NOTE_KEY (AES-256-GCM,
--                             "v1:<kid>:<base64>", additional data binds booking and sender) — never plain text.
--                             file_id = an attachment of the user's (files purpose telemed_attachment, encrypted by
--                             internal/files). read_at = when the other party first listed it.
-- telemed_prescription_items  the doctor's e-prescription lines of a booking: drug, dose, form (care medication
--                             forms), times ("08:00,20:00", 1–4 slots), duration_days (NULL = ongoing), instructions.
--                             reminder_id / added_at = the care medication reminder the user created from it (one tap).
--
-- Twin of backend/database/migrations/2026_10_08_000051_create_telemed_chat_tables.php (Laravel owns the prod schema
-- until T-M2-27), so `make schema-diff` stays green. IF NOT EXISTS so a database whose Laravel half already ran does
-- not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `telemed_messages` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `booking_id` bigint(20) unsigned NOT NULL,
  `sender` varchar(8) NOT NULL,
  `admin_id` bigint(20) unsigned DEFAULT NULL,
  `body` text NOT NULL,
  `file_id` bigint(20) unsigned DEFAULT NULL,
  `read_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `telemed_messages_admin_id_foreign` (`admin_id`),
  KEY `telemed_messages_file_id_foreign` (`file_id`),
  KEY `telemed_messages_booking_id_id_index` (`booking_id`,`id`),
  CONSTRAINT `telemed_messages_admin_id_foreign` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`) ON DELETE SET NULL,
  CONSTRAINT `telemed_messages_booking_id_foreign` FOREIGN KEY (`booking_id`) REFERENCES `telemed_bookings` (`id`) ON DELETE CASCADE,
  CONSTRAINT `telemed_messages_file_id_foreign` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `telemed_prescription_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `booking_id` bigint(20) unsigned NOT NULL,
  `drug` varchar(120) NOT NULL,
  `dose` varchar(50) DEFAULT NULL,
  `form` varchar(16) NOT NULL,
  `times` varchar(32) NOT NULL,
  `duration_days` smallint(5) unsigned DEFAULT NULL,
  `instructions` varchar(500) DEFAULT NULL,
  `reminder_id` bigint(20) unsigned DEFAULT NULL,
  `added_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `telemed_prescription_items_booking_id_foreign` (`booking_id`),
  KEY `telemed_prescription_items_reminder_id_foreign` (`reminder_id`),
  CONSTRAINT `telemed_prescription_items_booking_id_foreign` FOREIGN KEY (`booking_id`) REFERENCES `telemed_bookings` (`id`) ON DELETE CASCADE,
  CONSTRAINT `telemed_prescription_items_reminder_id_foreign` FOREIGN KEY (`reminder_id`) REFERENCES `reminders` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `telemed_prescription_items`;
DROP TABLE IF EXISTS `telemed_messages`;
