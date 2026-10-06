-- 00043_todo.sql — to-do list «کارهای من» (bloom B-N6-08, D-66; artboards nbl_Todo_* in f-tools): tasks with a
-- category, an optional due date / time and reminder, shopping lists (items under a task) and the cycle suggestion
-- («پریودت ۵ روز دیگر است؛ «خرید نوار بهداشتی» را اضافه کنم؟»). Domain logic: internal/todo.
--
-- todo_tasks              one task of a user. category shopping | work | personal | health (validated in Go);
--                         due_date / due_time (HH:MM) = Tehran wall-clock, both optional; remind = push at due_date
--                         due_time (needs both; planned through notifications.Decide); done_at = when it was ticked
--                         (NULL = open); suggestion_key = the cycle suggestion it came from (NULL = typed by hand).
-- todo_items              the items of a list («لیست خرید»): any task may carry items; due_date optional («قبل از
--                         ۱۷ مهر»); done_at NULL = open. user_id is denormalised so every query is scoped by user.
-- todo_suggestion_events  the user's answer to one cycle suggestion for one predicted period (ref_date = the
--                         predicted period start): accepted | dismissed, so it is not offered again for that period.
--
-- message_contents todo_suggestion / period_supplies (fa, en): the admin-editable copy of the suggestion (prompt with
-- {days}, the tomorrow variant, the action label, the task / item titles it adds). The timing window stays in code.
--
-- Health data (the suggestion reveals the cycle): rows are scoped to their user and never logged. Twin of
-- backend/database/migrations/2026_10_06_000043_create_todo_tables.php (Laravel owns the prod schema until T-M2-27),
-- so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables; IF NOT EXISTS /
-- INSERT IGNORE so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `todo_tasks` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `title` varchar(120) NOT NULL,
  `note` varchar(500) DEFAULT NULL,
  `category` varchar(12) NOT NULL,
  `due_date` date DEFAULT NULL,
  `due_time` varchar(5) DEFAULT NULL,
  `remind` tinyint(1) NOT NULL DEFAULT 0,
  `done_at` datetime DEFAULT NULL,
  `suggestion_key` varchar(40) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `todo_tasks_user_id_done_at_index` (`user_id`,`done_at`),
  KEY `todo_tasks_user_id_due_date_index` (`user_id`,`due_date`),
  CONSTRAINT `todo_tasks_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `todo_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `task_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `title` varchar(120) NOT NULL,
  `due_date` date DEFAULT NULL,
  `done_at` datetime DEFAULT NULL,
  `sort_order` smallint(5) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `todo_items_task_id_sort_order_index` (`task_id`,`sort_order`),
  KEY `todo_items_user_id_index` (`user_id`),
  CONSTRAINT `todo_items_task_id_foreign` FOREIGN KEY (`task_id`) REFERENCES `todo_tasks` (`id`) ON DELETE CASCADE,
  CONSTRAINT `todo_items_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `todo_suggestion_events` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `suggestion_key` varchar(40) NOT NULL,
  `ref_date` date NOT NULL,
  `action` varchar(10) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `todo_suggestion_events_user_id_suggestion_key_ref_date_unique` (`user_id`,`suggestion_key`,`ref_date`),
  CONSTRAINT `todo_suggestion_events_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `message_contents` (`group`, `item_key`, `locale`, `label`, `payload`, `is_active`, `is_approved`, `sort_order`, `created_at`, `updated_at`) VALUES
  ('todo_suggestion', 'period_supplies', 'fa', 'todo_suggestion / period_supplies', '{"prompt":"پریودت {days} روز دیگر است؛ «{title}» را اضافه کنم؟","prompt_tomorrow":"پریودت احتمالاً از فردا شروع می‌شود؛ «{title}» را اضافه کنم؟","action":"افزودن","task_title":"خرید نوار بهداشتی","item_title":"نوار بهداشتی"}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('todo_suggestion', 'period_supplies', 'en', 'todo_suggestion / period_supplies', '{"prompt":"Your period is {days} days away. Add “{title}”?","prompt_tomorrow":"Your period may start tomorrow. Add “{title}”?","action":"Add","task_title":"Buy pads","item_title":"Pads"}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
-- Removes the seeded copy rows only while nobody edited them (label untouched, updated_at = created_at), like 00035.
DELETE FROM `message_contents`
 WHERE `group` = 'todo_suggestion' AND `item_key` = 'period_supplies'
   AND `locale` IN ('fa', 'en')
   AND `label` = CONCAT(`group`, ' / ', `item_key`)
   AND `updated_at` <=> `created_at`;
DROP TABLE IF EXISTS `todo_suggestion_events`;
DROP TABLE IF EXISTS `todo_items`;
DROP TABLE IF EXISTS `todo_tasks`;
