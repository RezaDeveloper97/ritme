-- 00121_push_subscriptions.sql — Web Push for medication reminders: the browsers a user allowed notifications on
-- (push_subscriptions) and the doses already pushed (reminder_pushes: one row per reminder, day and slot, claimed with
-- INSERT IGNORE so two processes never push the same dose twice).
--
-- Laravel twin: 2026_11_10_000121_create_push_subscriptions_tables.php (schema only, no model).

-- +goose Up
CREATE TABLE IF NOT EXISTS `push_subscriptions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `endpoint` text NOT NULL,
  `endpoint_hash` char(64) NOT NULL,
  `p256dh` varchar(255) NOT NULL,
  `auth` varchar(255) NOT NULL,
  `locale` varchar(5) NOT NULL DEFAULT 'fa',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `push_subscriptions_endpoint_hash_unique` (`endpoint_hash`),
  KEY `push_subscriptions_user_id_foreign` (`user_id`),
  CONSTRAINT `push_subscriptions_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `reminder_pushes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `reminder_id` bigint(20) unsigned NOT NULL,
  `intake_date` date NOT NULL,
  `slot` char(5) NOT NULL,
  `sent_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `reminder_pushes_reminder_id_intake_date_slot_unique` (`reminder_id`,`intake_date`,`slot`),
  KEY `reminder_pushes_user_id_foreign` (`user_id`),
  CONSTRAINT `reminder_pushes_reminder_id_foreign` FOREIGN KEY (`reminder_id`) REFERENCES `reminders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `reminder_pushes_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS `reminder_pushes`;
DROP TABLE IF EXISTS `push_subscriptions`;
