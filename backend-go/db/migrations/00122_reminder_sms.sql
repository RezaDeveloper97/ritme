-- 00122_reminder_sms.sql — reminders by SMS for Ritme Plus (internal/remindersms): one row per reminder occurrence
-- (a medication dose "2026-11-11 08:00", an appointment's remind-at), claimed with INSERT IGNORE so two processes
-- never text the same occurrence twice. `sent` = 1 once the gateway accepted it (the per-day cap counts those).
--
-- Laravel twin: 2026_11_11_000122_create_reminder_sms_table.php (schema only, no model).

-- +goose Up
CREATE TABLE IF NOT EXISTS `reminder_sms` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `reminder_id` bigint(20) unsigned NOT NULL,
  `occurrence` varchar(32) NOT NULL,
  `sent` tinyint(1) NOT NULL DEFAULT 0,
  `sent_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `reminder_sms_reminder_id_occurrence_unique` (`reminder_id`,`occurrence`),
  KEY `reminder_sms_user_id_sent_at_index` (`user_id`,`sent_at`),
  CONSTRAINT `reminder_sms_reminder_id_foreign` FOREIGN KEY (`reminder_id`) REFERENCES `reminders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `reminder_sms_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `reminder_sms`;
