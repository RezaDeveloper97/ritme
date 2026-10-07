-- 00050_telemed_bookings.sql — visit booking with payment, cancellation rules and data-share consent (bloom B-N7-03,
-- D-80; artboards nbl_v17_ReviewBook / nbl_v17_Booked in d-doctor-assistant). Domain logic: internal/telemed
-- (booking*.go). Chat (B-N7-04), the doctor screens (B-N7-05) and the admin console (B-N7-08) build on these tables.
--
-- telemed_bookings          one visit: user → doctor, mode (video | phone | in_person), Tehran wall-clock
--                           starts_at / ends_at (end exclusive), status held → confirmed → completed, or cancelled /
--                           expired / failed (validated in Go). slot_key = "doctor:start" while the booking blocks its
--                           slot (held / confirmed), NULL otherwise — UNIQUE, the database backstop against
--                           double-booking (the service also locks the doctor row and checks overlaps).
--                           hold_expires_at = end of the payment hold (held only; the sweeper expires it).
--                           for_whom self | child | other: child_id = one of the user's children, patient_name = the
--                           free name of someone else. reason = a catalog_items code of the group telemed_visit_reasons,
--                           note = the user's text for the doctor (plain text, owner + doctor only).
--                           Money in integer rials: price (visit type snapshot), discount (Plus «تخفیف ویزیت», source
--                           plus), total charged. reference = the public payment reference ("TB" + 16 base32 chars);
--                           gateway / authority / ref_id / card_pan / paid_at = the payments adapter's result
--                           (UNIQUE gateway+ref_id: one bank payment settles one booking). payment_status none |
--                           pending | paid | free | refunded | refund_pending. Refunds: refund_id / refunded_rials /
--                           refunded_at. appointment_id = the care appointment (reminders row) created on
--                           confirmation. reschedules counts free moves.
-- telemed_booking_consents  scoped data-share consent per booking: scope cycle_summary | bbt_lh | assistant_summaries,
--                           granted_at, revoked_at (NULL = active). Only for this booking's doctor, only within the
--                           visit window (internal/telemed consent.go); nothing is shared without an active row.
--
-- catalog_items telemed_visit_reasons (fa, en): the «دلیل مراجعه» chips as a starting list, admin-editable.
--
-- Twin of backend/database/migrations/2026_10_07_000050_create_telemed_bookings_tables.php (Laravel owns the prod
-- schema until T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built
-- tables; IF NOT EXISTS / INSERT IGNORE so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `telemed_bookings` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `reference` varchar(24) NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `doctor_id` bigint(20) unsigned NOT NULL,
  `mode` varchar(12) NOT NULL,
  `duration_minutes` smallint(5) unsigned NOT NULL,
  `starts_at` datetime NOT NULL,
  `ends_at` datetime NOT NULL,
  `status` varchar(12) NOT NULL,
  `slot_key` varchar(48) DEFAULT NULL,
  `hold_expires_at` datetime DEFAULT NULL,
  `for_whom` varchar(8) NOT NULL,
  `child_id` bigint(20) unsigned DEFAULT NULL,
  `patient_name` varchar(64) DEFAULT NULL,
  `reason` varchar(64) DEFAULT NULL,
  `note` varchar(1000) DEFAULT NULL,
  `price_rials` bigint(20) unsigned NOT NULL,
  `discount_rials` bigint(20) unsigned NOT NULL DEFAULT 0,
  `discount_source` varchar(16) DEFAULT NULL,
  `total_rials` bigint(20) unsigned NOT NULL,
  `payment_status` varchar(16) NOT NULL DEFAULT 'none',
  `gateway` varchar(32) DEFAULT NULL,
  `authority` varchar(191) DEFAULT NULL,
  `ref_id` varchar(191) DEFAULT NULL,
  `card_pan` varchar(32) DEFAULT NULL,
  `paid_at` datetime DEFAULT NULL,
  `refund_id` varchar(191) DEFAULT NULL,
  `refunded_rials` bigint(20) unsigned NOT NULL DEFAULT 0,
  `refunded_at` datetime DEFAULT NULL,
  `appointment_id` bigint(20) unsigned DEFAULT NULL,
  `reschedules` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `cancelled_at` datetime DEFAULT NULL,
  `completed_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `telemed_bookings_reference_unique` (`reference`),
  UNIQUE KEY `telemed_bookings_slot_key_unique` (`slot_key`),
  UNIQUE KEY `telemed_bookings_gateway_ref_id_unique` (`gateway`,`ref_id`),
  KEY `telemed_bookings_user_id_starts_at_index` (`user_id`,`starts_at`),
  KEY `telemed_bookings_doctor_id_starts_at_index` (`doctor_id`,`starts_at`),
  KEY `telemed_bookings_status_hold_expires_at_index` (`status`,`hold_expires_at`),
  KEY `telemed_bookings_child_id_foreign` (`child_id`),
  KEY `telemed_bookings_appointment_id_foreign` (`appointment_id`),
  CONSTRAINT `telemed_bookings_appointment_id_foreign` FOREIGN KEY (`appointment_id`) REFERENCES `reminders` (`id`) ON DELETE SET NULL,
  CONSTRAINT `telemed_bookings_child_id_foreign` FOREIGN KEY (`child_id`) REFERENCES `children` (`id`) ON DELETE SET NULL,
  CONSTRAINT `telemed_bookings_doctor_id_foreign` FOREIGN KEY (`doctor_id`) REFERENCES `telemed_doctors` (`id`) ON DELETE CASCADE,
  CONSTRAINT `telemed_bookings_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `telemed_booking_consents` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `booking_id` bigint(20) unsigned NOT NULL,
  `scope` varchar(32) NOT NULL,
  `granted_at` datetime NOT NULL,
  `revoked_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `telemed_booking_consents_booking_id_scope_unique` (`booking_id`,`scope`),
  CONSTRAINT `telemed_booking_consents_booking_id_foreign` FOREIGN KEY (`booking_id`) REFERENCES `telemed_bookings` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('telemed_visit_reasons', 'ttc', 1, 1, NULL, '{"fa":"اقدام به بارداری","en":"Trying to conceive"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('telemed_visit_reasons', 'irregular_period', 2, 1, NULL, '{"fa":"پریود نامنظم","en":"Irregular periods"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('telemed_visit_reasons', 'pelvic_pain', 3, 1, NULL, '{"fa":"درد لگن","en":"Pelvic pain"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('telemed_visit_reasons', 'lab_followup', 4, 1, NULL, '{"fa":"پیگیری آزمایش","en":"Lab result follow-up"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('telemed_visit_reasons', 'other', 5, 1, NULL, '{"fa":"سایر","en":"Other"}', NULL, NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
-- Removes the seeded reasons only while nobody edited them (updated_at = created_at), like 00045's specialties.
DELETE FROM `catalog_items`
 WHERE `group` = 'telemed_visit_reasons'
   AND `code` IN ('ttc', 'irregular_period', 'pelvic_pain', 'lab_followup', 'other')
   AND `updated_at` <=> `created_at`;
DROP TABLE IF EXISTS `telemed_booking_consents`;
DROP TABLE IF EXISTS `telemed_bookings`;
