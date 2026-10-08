-- 00055_learning_security.sql — courses security fixes (bloom B-N8-01b, audit of B-N8-01).
--
-- learning_grants.seen_at   first time the student opened a course of this instructor (course page or a lesson).
--                   Until then the instructor sees the grant as «pending» with no name / registered flag / dates /
--                   progress, so typing a number never tells whether it has a Ritme account (H1 phone enumeration).
-- learning_sms_suppressions numbers that answered STOP (M3): the course-unlocked SMS is never sent to them again
--                   (outbox rows end skipped/suppressed). There is no inbound SMS webhook yet: support records a STOP
--                   by inserting the number here (admin DB tool); a later inbound Kavenegar hook writes source=sms.
--
-- Twin of backend/database/migrations/2026_10_08_000055_add_learning_security_fields.php (Laravel owns the prod
-- schema until T-M2-27), so `make schema-diff` stays green. IF [NOT] EXISTS so a database whose Laravel half already
-- ran does not fail.

-- +goose Up
ALTER TABLE `learning_grants` ADD COLUMN IF NOT EXISTS `seen_at` datetime DEFAULT NULL AFTER `revoked_at`;
CREATE INDEX IF NOT EXISTS `learning_grants_instructor_id_created_at_index` ON `learning_grants` (`instructor_id`,`created_at`);

CREATE TABLE IF NOT EXISTS `learning_sms_suppressions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `phone` varchar(11) NOT NULL,
  `source` varchar(16) NOT NULL DEFAULT 'support',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `learning_sms_suppressions_phone_unique` (`phone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `learning_sms_suppressions`;
DROP INDEX IF EXISTS `learning_grants_instructor_id_created_at_index` ON `learning_grants`;
ALTER TABLE `learning_grants` DROP COLUMN IF EXISTS `seen_at`;
