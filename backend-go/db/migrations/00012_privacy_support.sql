-- 00012_privacy_support.sql — privacy & support (B-N1-12, bloom/N1).
--
--  1. `user_consents` — one row per (user, consent code) once the user has answered: AI lab analysis, the health
--     assistant using the profile, anonymous statistics. No row = not granted (every consent is opt-in).
--     `granted_at` / `revoked_at` keep the timestamp of the last grant / withdrawal (Tehran wall-clock).
--  2. `support_reports` — «گزارش مشکل»: the user's text, an optional screenshot stored on the private storage
--     volume (`screenshot_path`, relative to STORAGE_PATH, never on the public disk), app version and user agent.
--  3. Seed `info_sections` rows the new screens read by key (INSERT IGNORE on the (group, key) unique index, so an
--     existing / admin-edited row is kept): `privacy/summary` and `terms/summary` (the Legal screen's «خلاصه در
--     ۳ خط», one line per `\n`), `about/disclaimer` (the About footer) and `support/email` (the support channel the
--     Support screen links to). The
--     `support` group itself is new (internal/content InfoGroups); admins edit all of them like any info box.
--
-- Twin of backend/database/migrations/2026_10_01_000003_create_privacy_support_tables.php (Laravel owns the prod
-- schema until T-M2-27; schema + the same seed rows, no models/routes), so `make schema-diff` stays green. Spelled
-- the way mariadb-dump prints the Laravel-built tables. IF NOT EXISTS: a database whose Laravel half already created
-- the tables (prod at cutover) must not fail here.

-- +goose Up
CREATE TABLE IF NOT EXISTS `user_consents` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `consent` varchar(64) NOT NULL,
  `granted` tinyint(1) NOT NULL DEFAULT 0,
  `granted_at` timestamp NULL DEFAULT NULL,
  `revoked_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_consents_user_id_consent_unique` (`user_id`,`consent`),
  CONSTRAINT `user_consents_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `support_reports` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `message` text NOT NULL,
  `screenshot_path` varchar(255) DEFAULT NULL,
  `app_version` varchar(32) DEFAULT NULL,
  `user_agent` varchar(255) DEFAULT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'open',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `support_reports_user_id_foreign` (`user_id`),
  KEY `support_reports_status_created_at_index` (`status`,`created_at`),
  CONSTRAINT `support_reports_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `info_sections` (`group`, `key`, `heading`, `body`, `link_label`, `link_url`, `is_active`, `sort_order`, `created_at`, `updated_at`) VALUES
  ('privacy', 'summary', '{"fa":"خلاصه در ۳ خط","en":"In 3 lines"}', '{"fa":"داده‌های سلامتت را نمی‌فروشیم و برای تبلیغات استفاده نمی‌کنیم\\nاشتراک با پزشک، همراه یا هوش مصنوعی فقط با رضایت صریح توست\\nهر زمان می‌توانی داده‌هایت را دریافت یا حذف کنی","en":"We never sell your health data or use it for advertising\\nSharing with a doctor, a companion or AI happens only with your explicit consent\\nYou can download or delete your data at any time"}', NULL, NULL, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('terms', 'summary', '{"fa":"خلاصه در ۳ خط","en":"In 3 lines"}', '{"fa":"ریتمی ابزار آگاهی سلامت است و جایگزین تشخیص یا درمان پزشکی نیست\\nپیش‌بینی‌های ریتمی روش پیشگیری از بارداری نیستند\\nکد ورود را به کسی نده؛ امنیت حساب و گوشی با خودت است","en":"Ritme is a health-awareness tool, not a substitute for medical diagnosis or treatment\\nRitme\'s predictions are not a method of contraception\\nNever share your sign-in code; keeping your account and phone secure is up to you"}', NULL, NULL, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('about', 'disclaimer', '{"fa":"یادآوری مهم","en":"Please note"}', '{"fa":"ریتمی ابزار آگاهی سلامت است و جایگزین تشخیص یا درمان پزشکی نیست. پیش‌بینی‌ها روش پیشگیری از بارداری نیستند.","en":"Ritme is a health-awareness tool, not a substitute for medical diagnosis or treatment. Predictions are not a method of contraception."}', NULL, NULL, 1, 1000, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('support', 'email', '{"fa":"ایمیل پشتیبانی","en":"Support email"}', '{"fa":"برایمان بنویس؛ هرچه زودتر جواب می‌دهیم","en":"Write to us and we\'ll reply as soon as we can"}', '{"fa":"support@ritmesalamat.com","en":"support@ritmesalamat.com"}', 'mailto:support@ritmesalamat.com', 1, 10, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `info_sections`
 WHERE ((`group` = 'privacy' AND `key` = 'summary') OR (`group` = 'terms' AND `key` = 'summary')
        OR (`group` = 'about' AND `key` = 'disclaimer') OR (`group` = 'support' AND `key` = 'email'))
   AND `updated_at` <=> `created_at`;
DROP TABLE IF EXISTS `support_reports`;
DROP TABLE IF EXISTS `user_consents`;
