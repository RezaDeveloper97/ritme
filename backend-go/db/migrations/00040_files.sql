-- 00040_files.sql — generic encrypted file storage (canvas-build CB-CORE-05, internal/files, docs/canvas-build/files.md):
-- one row per stored file of every document kind the canvas needs — record documents, insurance claim documents,
-- place photos / licences (directory), product images (shop). The bytes live encrypted (AES-256-GCM, FILE_KEY with
-- LAB_FILE_KEY as fallback) under STORAGE_PATH/app/private/files/<purpose>/<owner>/<random>.bin, never under
-- app/public. Lab sheets keep their own lab_files table (bloom B-N6-06) and only share the encryption code.
--
-- files   user_id = the owner (NULL = uploaded by the Ritme team for a public purpose, e.g. a product image).
--         purpose record_document | claim_document | place_photo | place_licence | product_image (validated in Go;
--         per-purpose size / count quotas live in internal/files). visibility private (owner + signed short-lived
--         URL only) | public (served without auth, public purposes only — images). mime / size_bytes / sha256 describe
--         the stored plaintext (photos are re-encoded to WebP first). path is relative to STORAGE_PATH.
--
-- Health data: rows are scoped to their owner, never logged, and go with the account (FK cascade; the blobs are
-- removed by internal/files.RemoveUser). Twin of backend/database/migrations/2026_10_06_000040_create_files_table.php
-- (Laravel owns the prod schema until T-M2-27), so `make schema-diff` stays green.

-- +goose Up
CREATE TABLE IF NOT EXISTS `files` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `purpose` varchar(32) NOT NULL,
  `visibility` varchar(8) NOT NULL,
  `mime` varchar(32) NOT NULL,
  `size_bytes` int(10) unsigned NOT NULL,
  `sha256` char(64) NOT NULL,
  `path` varchar(255) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `files_user_id_purpose_index` (`user_id`,`purpose`),
  CONSTRAINT `files_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `files`;
