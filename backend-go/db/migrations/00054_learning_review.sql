-- 00054_learning_review.sql — admin moderation of courses «مدرسین و دوره‌ها» (bloom B-N8-08, admin-api.md §20).
--
-- learning_lessons (00046) gets a content-review state the admin review queue (and later the B-N9-10 safety queue)
-- reads:
--   review_status   pending (never reviewed — every existing and new lesson) | approved | flagged
--   reviewed_at     when an admin last approved / flagged it; a lesson whose updated_at is later than reviewed_at
--                   (the instructor edited it, or B-N8-02 replaced its media) is «changed since review» and shows up in
--                   the queue again without the instructor code having to reset anything.
--   reviewed_by     admins.id of that admin (no FK: the trail outlives a deleted admin account).
--   review_note     the admin's reason for a flag (≤ 300 chars, shown to admins only).
--
-- learning_moderation_log  one row per admin moderation action (the durable audit trail; the slog "admin audit" line is
--                   written too): instructor.approve | instructor.revoke | lesson.approve | lesson.flag |
--                   lesson.unpublish. target_type instructor | lesson + target_id; instructor_id = the instructor the
--                   action concerns (no FK: kept after the instructor row is deleted); note = the admin's reason. Ids
--                   only — never phone numbers or lesson text.
--
-- Twin of backend/database/migrations/2026_10_07_000054_add_review_to_learning_lessons.php (Laravel owns the prod
-- schema until T-M2-27), so `make schema-diff` stays green. IF [NOT] EXISTS so a database whose Laravel half already
-- ran does not fail.

-- +goose Up
ALTER TABLE `learning_lessons` ADD COLUMN IF NOT EXISTS `review_status` varchar(10) NOT NULL DEFAULT 'pending' AFTER `published_at`;
ALTER TABLE `learning_lessons` ADD COLUMN IF NOT EXISTS `reviewed_at` datetime DEFAULT NULL AFTER `review_status`;
ALTER TABLE `learning_lessons` ADD COLUMN IF NOT EXISTS `reviewed_by` bigint(20) unsigned DEFAULT NULL AFTER `reviewed_at`;
ALTER TABLE `learning_lessons` ADD COLUMN IF NOT EXISTS `review_note` varchar(300) DEFAULT NULL AFTER `reviewed_by`;
CREATE INDEX IF NOT EXISTS `learning_lessons_review_status_index` ON `learning_lessons` (`review_status`);

CREATE TABLE IF NOT EXISTS `learning_moderation_log` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `admin_id` bigint(20) unsigned DEFAULT NULL,
  `action` varchar(32) NOT NULL,
  `target_type` varchar(16) NOT NULL,
  `target_id` bigint(20) unsigned NOT NULL,
  `instructor_id` bigint(20) unsigned DEFAULT NULL,
  `note` varchar(300) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `learning_moderation_log_target_type_target_id_index` (`target_type`,`target_id`),
  KEY `learning_moderation_log_instructor_id_index` (`instructor_id`),
  KEY `learning_moderation_log_created_at_index` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `learning_moderation_log`;
DROP INDEX IF EXISTS `learning_lessons_review_status_index` ON `learning_lessons`;
ALTER TABLE `learning_lessons` DROP COLUMN IF EXISTS `review_note`;
ALTER TABLE `learning_lessons` DROP COLUMN IF EXISTS `reviewed_by`;
ALTER TABLE `learning_lessons` DROP COLUMN IF EXISTS `reviewed_at`;
ALTER TABLE `learning_lessons` DROP COLUMN IF EXISTS `review_status`;
