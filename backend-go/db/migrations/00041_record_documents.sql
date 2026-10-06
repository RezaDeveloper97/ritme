-- 00041_record_documents.sql — record documents, extras and timeline (canvas-build CB-REC-01; boards nbl_Rec_Home,
-- nbl_Rec_Timeline, nbl_Rec_Doc), built on bloom's health record (B-N6-03, 00036) and the generic file storage
-- (CB-CORE-05, 00040). Domain logic: internal/healthrecord (documents*.go, extras.go, timeline.go).
--
-- health_records (extended, as 00036 planned — no second per-user row):
--   allergies_on_emergency_card  1 = the allergies are printed on the emergency card (CB-REC-03), 0 = hidden.
--   surgeries                    JSON list of {title, date (Y-m-d | null)} («بستری و جراحی»), NULL = never answered.
--   family_history               JSON list of {condition, relative (code | null)} («سابقه خانوادگی»).
--
-- record_documents       one document of the record: kind imaging | visit | prescription | hospital | other (lab sheets
--                        stay in lab_reports and only appear in the timeline), document_date (NULL until known — the
--                        timeline then files it under the upload day), ended_on (hospital discharge), centre, doctor,
--                        note (free text, owner only), extracted (the AI step's JSON, CB-REC-02; NULL until then),
--                        review_state manual | pending | needs_review | confirmed | failed.
-- record_document_files  the document's files (internal/files rows of purpose record_document, owned by the same
--                        user — checked in Go); a file belongs to at most one document (unique file_id); deleting the
--                        file drops the row (FK cascade), deleting the document deletes its files in Go.
-- record_document_links  «این سند کجا استفاده شده؟»: target_type claim (CB-INS) | pregnancy (pregnancy_profiles id,
--                        CB-REC-02 dating hook), target_id, state attached | waiting | applied. Written by the Go code
--                        of the owning task, never by a client.
--
-- Health data: owner-only, never logged, gone with the account (FK cascade; the blobs via files.RemoveUser). Codes are
-- varchar validated in Go. Twin of backend/database/migrations/2026_10_06_000041_create_record_documents_tables.php
-- (Laravel owns the prod schema until T-M2-27), so `make schema-diff` stays green.

-- +goose Up
ALTER TABLE `health_records` ADD COLUMN IF NOT EXISTS `allergies_on_emergency_card` tinyint(1) NOT NULL DEFAULT 1 AFTER `allergies`;
ALTER TABLE `health_records` ADD COLUMN IF NOT EXISTS `surgeries` json DEFAULT NULL AFTER `allergies_on_emergency_card`;
ALTER TABLE `health_records` ADD COLUMN IF NOT EXISTS `family_history` json DEFAULT NULL AFTER `surgeries`;

CREATE TABLE IF NOT EXISTS `record_documents` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(16) NOT NULL,
  `title` varchar(120) DEFAULT NULL,
  `document_date` date DEFAULT NULL,
  `ended_on` date DEFAULT NULL,
  `centre` varchar(120) DEFAULT NULL,
  `doctor` varchar(120) DEFAULT NULL,
  `note` text DEFAULT NULL,
  `extracted` json DEFAULT NULL,
  `review_state` varchar(16) NOT NULL DEFAULT 'manual',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `record_documents_user_id_document_date_index` (`user_id`,`document_date`),
  CONSTRAINT `record_documents_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `record_document_files` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `document_id` bigint(20) unsigned NOT NULL,
  `file_id` bigint(20) unsigned NOT NULL,
  `position` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `record_document_files_file_id_unique` (`file_id`),
  KEY `record_document_files_user_id_foreign` (`user_id`),
  KEY `record_document_files_document_id_foreign` (`document_id`),
  CONSTRAINT `record_document_files_document_id_foreign` FOREIGN KEY (`document_id`) REFERENCES `record_documents` (`id`) ON DELETE CASCADE,
  CONSTRAINT `record_document_files_file_id_foreign` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE CASCADE,
  CONSTRAINT `record_document_files_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `record_document_links` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `document_id` bigint(20) unsigned NOT NULL,
  `target_type` varchar(16) NOT NULL,
  `target_id` bigint(20) unsigned NOT NULL DEFAULT 0,
  `state` varchar(16) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `record_document_links_document_id_target_type_target_id_unique` (`document_id`,`target_type`,`target_id`),
  KEY `record_document_links_user_id_target_type_target_id_index` (`user_id`,`target_type`,`target_id`),
  CONSTRAINT `record_document_links_document_id_foreign` FOREIGN KEY (`document_id`) REFERENCES `record_documents` (`id`) ON DELETE CASCADE,
  CONSTRAINT `record_document_links_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `record_document_links`;
DROP TABLE IF EXISTS `record_document_files`;
DROP TABLE IF EXISTS `record_documents`;
ALTER TABLE `health_records` DROP COLUMN IF EXISTS `family_history`;
ALTER TABLE `health_records` DROP COLUMN IF EXISTS `surgeries`;
ALTER TABLE `health_records` DROP COLUMN IF EXISTS `allergies_on_emergency_card`;
