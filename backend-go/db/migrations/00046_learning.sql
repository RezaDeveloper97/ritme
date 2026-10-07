-- 00046_learning.sql — courses «دوره‌های من» / instructor panel (bloom B-N8-01, D-68; artboards nbl_Learn_* / nbl_Ins_*
-- in e-learning-instructor). Instructors (midwives, doctors) publish courses and standalone content and open them to
-- their students by mobile number. Domain logic: internal/learning. The media pipeline (B-N8-02), the learning screens
-- (B-N8-03), instructor-web (B-N8-05..07) and the admin moderation (B-N8-08) build on these tables.
--
-- learning_instructors   the instructor role of a user (one row per user): display_name / title («ماما») / bio as the
--                        students see them; status pending (applied) | approved (by an admin, approved_by = admins.id)
--                        | revoked. Only approved instructors reach /api/instructor/v1 and only their published
--                        courses are shown to students.
-- learning_courses       kind course (chapters → lessons) | standalone (one lesson, no chapter: «محتوای مستقل»);
--                        status draft | published. Title / description are the instructor's own text (one language).
--                        cover_media_id = a media id of B-N8-02 (no FK yet).
-- learning_chapters      ordered chapters; unlock_at (Tehran wall-clock) NULL = open, else locked until then
--                        («فصل ۳ از ۱۵ مهر توسط مدرس باز می‌شود»).
-- learning_lessons       kind video | audio | pdf; chapter_id NULL for a standalone item (or an unplaced lesson); status
--                        draft | published; duration_seconds (video/audio), page_count (pdf), size_bytes; media_id +
--                        media_status (none | uploading | processing | ready | failed) are placeholders B-N8-02 fills.
-- learning_groups        an instructor's student group («گروه مهر ۱۴۰۵»); learning_group_courses = the courses it opens.
-- learning_grants        access by mobile number: phone = the normalised 09xxxxxxxxx form; scope = exactly one of
--                        group_id / course_id; duration unlimited | days (duration_days 30 / 90) | until (until_date,
--                        inclusive). status pending (the phone has no account yet: user_id NULL) | active | revoked.
--                        A pending grant becomes active when that phone signs up (OTP signup hook) — expires_at is
--                        computed at activation (days count from then; until = the day after until_date, 00:00).
-- learning_progress      per user and lesson: position_seconds, percent (0–100), completed_at, last_seen_at
--                        («ادامه از جایی که ماندی»). course_id is denormalised for the per-course aggregates.
-- learning_sms_outbox    the «دوره برایت باز شد» SMS of a grant, sent by the in-process dispatcher (fake provider by
--                        default) after the notification policy (category learning, quiet hours) and the per-phone /
--                        per-instructor daily caps: status pending | sent | skipped | failed, due_at (deferred past
--                        quiet hours), lease_until (claimed by one worker).
--
-- Phone numbers are personal data: never logged in full (sms.MaskMobile), only shown to the instructor who typed them.
-- Twin of backend/database/migrations/2026_10_07_000046_create_learning_tables.php (Laravel owns the prod schema until
-- T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables; IF NOT
-- EXISTS so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `learning_instructors` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `display_name` varchar(80) NOT NULL,
  `title` varchar(60) DEFAULT NULL,
  `bio` varchar(500) DEFAULT NULL,
  `status` varchar(10) NOT NULL DEFAULT 'pending',
  `approved_at` datetime DEFAULT NULL,
  `approved_by` bigint(20) unsigned DEFAULT NULL,
  `revoked_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `learning_instructors_user_id_unique` (`user_id`),
  KEY `learning_instructors_status_index` (`status`),
  CONSTRAINT `learning_instructors_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `learning_courses` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `instructor_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(12) NOT NULL DEFAULT 'course',
  `title` varchar(150) NOT NULL,
  `description` text DEFAULT NULL,
  `cover_media_id` bigint(20) unsigned DEFAULT NULL,
  `status` varchar(10) NOT NULL DEFAULT 'draft',
  `published_at` datetime DEFAULT NULL,
  `sort_order` smallint(5) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `learning_courses_instructor_id_status_index` (`instructor_id`,`status`),
  CONSTRAINT `learning_courses_instructor_id_foreign` FOREIGN KEY (`instructor_id`) REFERENCES `learning_instructors` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `learning_chapters` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `course_id` bigint(20) unsigned NOT NULL,
  `title` varchar(150) NOT NULL,
  `sort_order` smallint(5) unsigned NOT NULL DEFAULT 0,
  `unlock_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `learning_chapters_course_id_sort_order_index` (`course_id`,`sort_order`),
  CONSTRAINT `learning_chapters_course_id_foreign` FOREIGN KEY (`course_id`) REFERENCES `learning_courses` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `learning_lessons` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `course_id` bigint(20) unsigned NOT NULL,
  `chapter_id` bigint(20) unsigned DEFAULT NULL,
  `kind` varchar(8) NOT NULL,
  `title` varchar(150) NOT NULL,
  `description` text DEFAULT NULL,
  `duration_seconds` int(10) unsigned DEFAULT NULL,
  `page_count` smallint(5) unsigned DEFAULT NULL,
  `size_bytes` bigint(20) unsigned DEFAULT NULL,
  `media_id` bigint(20) unsigned DEFAULT NULL,
  `media_status` varchar(12) NOT NULL DEFAULT 'none',
  `status` varchar(10) NOT NULL DEFAULT 'draft',
  `published_at` datetime DEFAULT NULL,
  `sort_order` smallint(5) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `learning_lessons_course_id_sort_order_index` (`course_id`,`sort_order`),
  KEY `learning_lessons_chapter_id_index` (`chapter_id`),
  CONSTRAINT `learning_lessons_chapter_id_foreign` FOREIGN KEY (`chapter_id`) REFERENCES `learning_chapters` (`id`) ON DELETE SET NULL,
  CONSTRAINT `learning_lessons_course_id_foreign` FOREIGN KEY (`course_id`) REFERENCES `learning_courses` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `learning_groups` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `instructor_id` bigint(20) unsigned NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `learning_groups_instructor_id_index` (`instructor_id`),
  CONSTRAINT `learning_groups_instructor_id_foreign` FOREIGN KEY (`instructor_id`) REFERENCES `learning_instructors` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `learning_group_courses` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `group_id` bigint(20) unsigned NOT NULL,
  `course_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `learning_group_courses_group_id_course_id_unique` (`group_id`,`course_id`),
  KEY `learning_group_courses_course_id_index` (`course_id`),
  CONSTRAINT `learning_group_courses_course_id_foreign` FOREIGN KEY (`course_id`) REFERENCES `learning_courses` (`id`) ON DELETE CASCADE,
  CONSTRAINT `learning_group_courses_group_id_foreign` FOREIGN KEY (`group_id`) REFERENCES `learning_groups` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `learning_grants` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `instructor_id` bigint(20) unsigned NOT NULL,
  `phone` varchar(11) NOT NULL,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `group_id` bigint(20) unsigned DEFAULT NULL,
  `course_id` bigint(20) unsigned DEFAULT NULL,
  `duration` varchar(10) NOT NULL,
  `duration_days` smallint(5) unsigned DEFAULT NULL,
  `until_date` date DEFAULT NULL,
  `status` varchar(10) NOT NULL DEFAULT 'pending',
  `activated_at` datetime DEFAULT NULL,
  `expires_at` datetime DEFAULT NULL,
  `revoked_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `learning_grants_phone_status_index` (`phone`,`status`),
  KEY `learning_grants_user_id_status_index` (`user_id`,`status`),
  KEY `learning_grants_instructor_id_status_index` (`instructor_id`,`status`),
  KEY `learning_grants_group_id_index` (`group_id`),
  KEY `learning_grants_course_id_index` (`course_id`),
  CONSTRAINT `learning_grants_course_id_foreign` FOREIGN KEY (`course_id`) REFERENCES `learning_courses` (`id`) ON DELETE CASCADE,
  CONSTRAINT `learning_grants_group_id_foreign` FOREIGN KEY (`group_id`) REFERENCES `learning_groups` (`id`) ON DELETE CASCADE,
  CONSTRAINT `learning_grants_instructor_id_foreign` FOREIGN KEY (`instructor_id`) REFERENCES `learning_instructors` (`id`) ON DELETE CASCADE,
  CONSTRAINT `learning_grants_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `learning_progress` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `lesson_id` bigint(20) unsigned NOT NULL,
  `course_id` bigint(20) unsigned NOT NULL,
  `position_seconds` int(10) unsigned NOT NULL DEFAULT 0,
  `percent` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `completed_at` datetime DEFAULT NULL,
  `last_seen_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `learning_progress_user_id_lesson_id_unique` (`user_id`,`lesson_id`),
  KEY `learning_progress_user_id_last_seen_at_index` (`user_id`,`last_seen_at`),
  KEY `learning_progress_lesson_id_index` (`lesson_id`),
  KEY `learning_progress_course_id_index` (`course_id`),
  CONSTRAINT `learning_progress_course_id_foreign` FOREIGN KEY (`course_id`) REFERENCES `learning_courses` (`id`) ON DELETE CASCADE,
  CONSTRAINT `learning_progress_lesson_id_foreign` FOREIGN KEY (`lesson_id`) REFERENCES `learning_lessons` (`id`) ON DELETE CASCADE,
  CONSTRAINT `learning_progress_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `learning_sms_outbox` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `grant_id` bigint(20) unsigned NOT NULL,
  `status` varchar(10) NOT NULL DEFAULT 'pending',
  `reason` varchar(20) DEFAULT NULL,
  `due_at` datetime NOT NULL,
  `attempts` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `lease_until` datetime DEFAULT NULL,
  `sent_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `learning_sms_outbox_status_due_at_index` (`status`,`due_at`),
  KEY `learning_sms_outbox_grant_id_index` (`grant_id`),
  CONSTRAINT `learning_sms_outbox_grant_id_foreign` FOREIGN KEY (`grant_id`) REFERENCES `learning_grants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `learning_sms_outbox`;
DROP TABLE IF EXISTS `learning_progress`;
DROP TABLE IF EXISTS `learning_grants`;
DROP TABLE IF EXISTS `learning_group_courses`;
DROP TABLE IF EXISTS `learning_groups`;
DROP TABLE IF EXISTS `learning_lessons`;
DROP TABLE IF EXISTS `learning_chapters`;
DROP TABLE IF EXISTS `learning_courses`;
DROP TABLE IF EXISTS `learning_instructors`;
