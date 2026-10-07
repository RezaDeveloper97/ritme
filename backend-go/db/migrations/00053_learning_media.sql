-- 00053_learning_media.sql — the media pipeline of the courses domain (bloom B-N8-02, D-81): resumable (tus-style)
-- uploads of lesson video / audio / pdf by approved instructors, and the files the signed, range-capable playback URLs
-- serve. Domain logic: internal/media (docs: its package comment). learning_lessons.media_id points at a ready row here
-- once its upload completes; learning_lessons.media_status mirrors the latest upload (none | uploading | processing |
-- ready | failed).
--
-- learning_media   one upload / stored file: instructor_id (owner), lesson_id (NULL once the lesson is deleted → the
--                  sweep removes the row and the file), kind video | audio | pdf (= the lesson kind), mime (sniffed
--                  from the bytes on the first chunk, never from the client), size_bytes (declared Upload-Length,
--                  bounded per kind: 2 GiB video), offset_bytes (bytes received so far — the resume point),
--                  path (relative to the media root: <instructor_id>/<random>.part|.bin), status uploading |
--                  processing | ready | failed, upload_expires_at (an unfinished upload is swept after it).
--
-- Course media is the instructor's published teaching material, not a user's health data: stored unencrypted (range
-- reads of a 2 GB file) on its own volume, directories 0700 / files 0600, served only through short-lived HMAC URLs.
-- Twin of backend/database/migrations/2026_10_07_000053_create_learning_media_table.php (Laravel owns the prod schema
-- until T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built table; IF
-- NOT EXISTS so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `learning_media` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `instructor_id` bigint(20) unsigned NOT NULL,
  `lesson_id` bigint(20) unsigned DEFAULT NULL,
  `kind` varchar(8) NOT NULL,
  `mime` varchar(40) DEFAULT NULL,
  `size_bytes` bigint(20) unsigned NOT NULL,
  `offset_bytes` bigint(20) unsigned NOT NULL DEFAULT 0,
  `path` varchar(120) NOT NULL,
  `status` varchar(12) NOT NULL DEFAULT 'uploading',
  `upload_expires_at` datetime DEFAULT NULL,
  `completed_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `learning_media_instructor_id_status_index` (`instructor_id`,`status`),
  KEY `learning_media_lesson_id_index` (`lesson_id`),
  KEY `learning_media_status_upload_expires_at_index` (`status`,`upload_expires_at`),
  CONSTRAINT `learning_media_instructor_id_foreign` FOREIGN KEY (`instructor_id`) REFERENCES `learning_instructors` (`id`) ON DELETE CASCADE,
  CONSTRAINT `learning_media_lesson_id_foreign` FOREIGN KEY (`lesson_id`) REFERENCES `learning_lessons` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `learning_media`;
