-- 00062_challenges_v2.sql — Challenges v2: eligibility rules, daily tasks, rewards (bloom B-N9-07, D-87).
--
-- challenge_programs          1:1 extension of `challenges` (a row = a v2 challenge; legacy one-day challenges have none
--                             and keep the home daily-challenge pool): duration, start rule, reminder time, eligibility
--                             JSON {modes_include, modes_exclude, phases_include, phases_exclude}, reward (badge + Plus
--                             days) and the completion rule (completion_min_days NULL = every day).
-- challenge_tasks             per-day tasks: a task applies to days day_from..day_to of the run.
-- challenge_enrollments       one per user and challenge, ever (UNIQUE): started_on, status active | completed.
-- challenge_task_completions  a task done on a day of an enrollment (UNIQUE = idempotent).
-- challenge_reward_grants     the reward of a completed enrollment, UNIQUE (user_id, challenge_id) and enrollment_id:
--                             the exactly-once guard of the Plus-days grant (written in the completing transaction).
--
-- Twin of backend/database/migrations/2026_10_11_000062_create_challenges_v2_tables.php (Laravel owns the prod schema
-- until T-M2-27), so `make schema-diff` stays green. IF NOT EXISTS so a database whose Laravel half already ran
-- does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `challenge_programs` (
  `challenge_id` bigint(20) unsigned NOT NULL,
  `duration_days` smallint(5) unsigned NOT NULL,
  `start_rule` varchar(16) NOT NULL DEFAULT 'anytime',
  `starts_on` date DEFAULT NULL,
  `reminder_time` varchar(5) DEFAULT NULL,
  `eligibility` json DEFAULT NULL,
  `reward_badge_code` varchar(64) DEFAULT NULL,
  `reward_badge_title` json DEFAULT NULL,
  `reward_plus_days` smallint(5) unsigned NOT NULL DEFAULT 0,
  `completion_min_days` smallint(5) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`challenge_id`),
  CONSTRAINT `challenge_programs_challenge_id_foreign` FOREIGN KEY (`challenge_id`) REFERENCES `challenges` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `challenge_tasks` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `challenge_id` bigint(20) unsigned NOT NULL,
  `day_from` smallint(5) unsigned NOT NULL DEFAULT 1,
  `day_to` smallint(5) unsigned NOT NULL DEFAULT 1,
  `title` json NOT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `challenge_tasks_challenge_id_sort_order_index` (`challenge_id`,`sort_order`),
  CONSTRAINT `challenge_tasks_challenge_id_foreign` FOREIGN KEY (`challenge_id`) REFERENCES `challenges` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `challenge_enrollments` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `challenge_id` bigint(20) unsigned NOT NULL,
  `started_on` date NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'active',
  `completed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `challenge_enrollments_user_id_challenge_id_unique` (`user_id`,`challenge_id`),
  KEY `challenge_enrollments_challenge_id_status_index` (`challenge_id`,`status`),
  CONSTRAINT `challenge_enrollments_challenge_id_foreign` FOREIGN KEY (`challenge_id`) REFERENCES `challenges` (`id`) ON DELETE CASCADE,
  CONSTRAINT `challenge_enrollments_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `challenge_task_completions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `enrollment_id` bigint(20) unsigned NOT NULL,
  `task_id` bigint(20) unsigned NOT NULL,
  `day_number` smallint(5) unsigned NOT NULL,
  `completed_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `challenge_task_completions_unique` (`enrollment_id`,`task_id`,`day_number`),
  KEY `challenge_task_completions_task_id_foreign` (`task_id`),
  CONSTRAINT `challenge_task_completions_enrollment_id_foreign` FOREIGN KEY (`enrollment_id`) REFERENCES `challenge_enrollments` (`id`) ON DELETE CASCADE,
  CONSTRAINT `challenge_task_completions_task_id_foreign` FOREIGN KEY (`task_id`) REFERENCES `challenge_tasks` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `challenge_reward_grants` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `challenge_id` bigint(20) unsigned NOT NULL,
  `enrollment_id` bigint(20) unsigned NOT NULL,
  `badge_code` varchar(64) DEFAULT NULL,
  `badge_title` json DEFAULT NULL,
  `plus_days` smallint(5) unsigned NOT NULL DEFAULT 0,
  `subscription_id` bigint(20) unsigned DEFAULT NULL,
  `granted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `challenge_reward_grants_user_id_challenge_id_unique` (`user_id`,`challenge_id`),
  UNIQUE KEY `challenge_reward_grants_enrollment_id_unique` (`enrollment_id`),
  KEY `challenge_reward_grants_challenge_id_foreign` (`challenge_id`),
  CONSTRAINT `challenge_reward_grants_challenge_id_foreign` FOREIGN KEY (`challenge_id`) REFERENCES `challenges` (`id`) ON DELETE CASCADE,
  CONSTRAINT `challenge_reward_grants_enrollment_id_foreign` FOREIGN KEY (`enrollment_id`) REFERENCES `challenge_enrollments` (`id`) ON DELETE CASCADE,
  CONSTRAINT `challenge_reward_grants_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `challenge_reward_grants`;
DROP TABLE IF EXISTS `challenge_task_completions`;
DROP TABLE IF EXISTS `challenge_enrollments`;
DROP TABLE IF EXISTS `challenge_tasks`;
DROP TABLE IF EXISTS `challenge_programs`;
