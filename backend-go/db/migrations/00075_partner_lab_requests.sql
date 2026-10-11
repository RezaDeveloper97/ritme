-- 00075_partner_lab_requests.sql — home lab test and screening booking requests (bloom B-N10-04, D-94).
--
-- partner_lab_services   the admin catalog of orderable services (home blood draw, annual checkup / Pap): localized
--                        title + description (JSON {"fa":…,"en":…}), the partner's display name, the adapter code,
--                        active flag and order. The Today «پیشنهاد» cards exist only while a service is active.
-- partner_lab_requests   one request of one user. Everything personal the user typed (name, phone, city, address,
--                        note) lives in ONE sealed blob `payload_enc` (PRIVATE_NOTE_KEY AES-256-GCM, bound to the
--                        owner + reference) and is set to NULL by the retention sweep 30 days after the request is
--                        closed (`purged_at`). The row keeps only the non-personal fields (service, status, preferred
--                        day + window, consent flags, partner reference). Results are never stored: they are fetched
--                        from the partner at import time and copied into labs only with the user's consent.
--
-- Twin of backend/database/migrations/2026_10_11_000075_create_partner_lab_tables.php. IF NOT EXISTS so a database
-- whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `partner_lab_services` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(40) NOT NULL,
  `kind` varchar(24) NOT NULL,
  `title` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`title`)),
  `description` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`description`)),
  `partner_name` varchar(120) NOT NULL,
  `partner_code` varchar(40) NOT NULL DEFAULT 'fake',
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `partner_lab_services_code_unique` (`code`),
  KEY `partner_lab_services_is_active_sort_order_index` (`is_active`,`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `partner_lab_requests` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `reference` varchar(16) NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `service_id` bigint(20) unsigned NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'requested',
  `partner_ref` varchar(64) DEFAULT NULL,
  `partner_status` varchar(16) DEFAULT NULL,
  `preferred_date` date NOT NULL,
  `preferred_window` varchar(16) NOT NULL,
  `consent_share` tinyint(1) NOT NULL DEFAULT 0,
  `consent_import` tinyint(1) NOT NULL DEFAULT 0,
  `payload_enc` longtext DEFAULT NULL,
  `imported_lab_id` bigint(20) unsigned DEFAULT NULL,
  `closed_at` datetime DEFAULT NULL,
  `purged_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `partner_lab_requests_reference_unique` (`reference`),
  KEY `partner_lab_requests_service_id_foreign` (`service_id`),
  KEY `partner_lab_requests_user_id_created_at_index` (`user_id`,`created_at`),
  KEY `partner_lab_requests_status_created_at_index` (`status`,`created_at`),
  KEY `partner_lab_requests_closed_at_index` (`closed_at`),
  CONSTRAINT `partner_lab_requests_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `partner_lab_requests_service_id_foreign` FOREIGN KEY (`service_id`) REFERENCES `partner_lab_services` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `partner_lab_requests`;
DROP TABLE IF EXISTS `partner_lab_services`;
