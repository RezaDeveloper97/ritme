-- 00033_baby_logs.sql — baby logs and pregnancy tools (bloom B-N5-03; artboards nbl_Log_Feed, nbl_Log_Kick,
-- nbl_Log_Contraction, the «امروز» card of nbl_v16_ChildHome, nbl_An_Hub_Post): feeding / baby sleep sessions and
-- diapers per child, kick-count sessions and contraction sessions per pregnant user. Domain logic: internal/babylog
-- and internal/pregnancy/tools (5-1-1 maths: internal/pregnancy/labor). 00032 belongs to B-N6-05; the canvas queue
-- numbers from 00034.
--
-- baby_feeds                      one feed of a child (children.id): type breast|bottle|pump, started_at, ended_at
--                                 (NULL = running). Breast feeds time each side: left_seconds / right_seconds hold
--                                 the closed segments, active_side + side_started_at the running one (NULL = paused),
--                                 last_side the side fed last («دفعه قبل · راست»). duration_seconds is set when the
--                                 feed ends (breast = left + right, bottle / pump = ended − started). amount_ml for
--                                 bottle / pump. active_lock = 1 while running, NULL once ended: UNIQUE(child_id,
--                                 active_lock) makes «one running feed per child» a database rule (a concurrent second
--                                 start fails with a duplicate key).
-- baby_sleeps                     one sleep of a child; same running-session rule.
-- baby_diapers                    one diaper change: kind wet|dirty|both, changed_at, optional note.
-- pregnancy_kick_sessions         a kick count («شمارش حرکات جنین»): kicks, the 10th kick's time (time to 10),
--                                 the last kick, the pregnancy week at the start; one running session per user. When a
--                                 session ends, the day's pregnancy_fetal_movements row (the existing
--                                 /pregnancy/fetal-movement log) gets the day's kick total, so the log sheet tile,
--                                 the analysis and the alert engine keep reading one table.
-- pregnancy_contraction_sessions  a contraction timing run («زمان‌سنج انقباض»): alert_at = the first time the run met
--                                 the 5-1-1 rule; one running session per user.
-- pregnancy_contractions          one contraction of a session: started_at, ended_at (NULL = in progress, at most one
--                                 per session through UNIQUE(session_id, active_lock)). Interval = start-to-start.
--
-- message_contents pregnancy_alert / contractions_511 (fa, en): the 5-1-1 rule of the pregnancy alert engine
-- (internal/messages/pregnancyalerts) — params, call / hospital copy and actions are admin-editable through
-- admin-web pregnancy-alert-rules. [needs clinical review]
--
-- Health data: every row is scoped to its child (owner, or the owner's spouse read-only via children.Service.Access)
-- or to its user; never logged. Codes are varchar validated in Go. Twin of
-- backend/database/migrations/2026_10_04_000033_create_baby_logs_tables.php (Laravel owns the prod schema until
-- T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables; IF NOT
-- EXISTS / INSERT IGNORE so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `baby_feeds` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `child_id` bigint(20) unsigned NOT NULL,
  `type` varchar(8) NOT NULL,
  `started_at` datetime NOT NULL,
  `ended_at` datetime DEFAULT NULL,
  `active_side` varchar(8) DEFAULT NULL,
  `side_started_at` datetime DEFAULT NULL,
  `last_side` varchar(8) DEFAULT NULL,
  `left_seconds` int(10) unsigned NOT NULL DEFAULT 0,
  `right_seconds` int(10) unsigned NOT NULL DEFAULT 0,
  `duration_seconds` int(10) unsigned DEFAULT NULL,
  `amount_ml` smallint(5) unsigned DEFAULT NULL,
  `note` varchar(500) DEFAULT NULL,
  `active_lock` tinyint(3) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `baby_feeds_child_id_active_lock_unique` (`child_id`,`active_lock`),
  KEY `baby_feeds_child_id_started_at_index` (`child_id`,`started_at`),
  CONSTRAINT `baby_feeds_child_id_foreign` FOREIGN KEY (`child_id`) REFERENCES `children` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `baby_sleeps` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `child_id` bigint(20) unsigned NOT NULL,
  `started_at` datetime NOT NULL,
  `ended_at` datetime DEFAULT NULL,
  `note` varchar(500) DEFAULT NULL,
  `active_lock` tinyint(3) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `baby_sleeps_child_id_active_lock_unique` (`child_id`,`active_lock`),
  KEY `baby_sleeps_child_id_started_at_index` (`child_id`,`started_at`),
  CONSTRAINT `baby_sleeps_child_id_foreign` FOREIGN KEY (`child_id`) REFERENCES `children` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `baby_diapers` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `child_id` bigint(20) unsigned NOT NULL,
  `changed_at` datetime NOT NULL,
  `kind` varchar(8) NOT NULL,
  `note` varchar(500) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `baby_diapers_child_id_changed_at_index` (`child_id`,`changed_at`),
  CONSTRAINT `baby_diapers_child_id_foreign` FOREIGN KEY (`child_id`) REFERENCES `children` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `pregnancy_kick_sessions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `started_at` datetime NOT NULL,
  `ended_at` datetime DEFAULT NULL,
  `kicks` smallint(5) unsigned NOT NULL DEFAULT 0,
  `tenth_kick_at` datetime DEFAULT NULL,
  `last_kick_at` datetime DEFAULT NULL,
  `pregnancy_week` tinyint(3) unsigned DEFAULT NULL,
  `active_lock` tinyint(3) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pregnancy_kick_sessions_user_id_active_lock_unique` (`user_id`,`active_lock`),
  KEY `pregnancy_kick_sessions_user_id_started_at_index` (`user_id`,`started_at`),
  CONSTRAINT `pregnancy_kick_sessions_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `pregnancy_contraction_sessions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `started_at` datetime NOT NULL,
  `ended_at` datetime DEFAULT NULL,
  `alert_at` datetime DEFAULT NULL,
  `active_lock` tinyint(3) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pregnancy_contraction_sessions_user_id_active_lock_unique` (`user_id`,`active_lock`),
  KEY `pregnancy_contraction_sessions_user_id_started_at_index` (`user_id`,`started_at`),
  CONSTRAINT `pregnancy_contraction_sessions_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `pregnancy_contractions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `session_id` bigint(20) unsigned NOT NULL,
  `user_id` bigint(20) unsigned NOT NULL,
  `started_at` datetime NOT NULL,
  `ended_at` datetime DEFAULT NULL,
  `active_lock` tinyint(3) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pregnancy_contractions_session_id_active_lock_unique` (`session_id`,`active_lock`),
  KEY `pregnancy_contractions_user_id_started_at_index` (`user_id`,`started_at`),
  CONSTRAINT `pregnancy_contractions_session_id_foreign` FOREIGN KEY (`session_id`) REFERENCES `pregnancy_contraction_sessions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `pregnancy_contractions_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `message_contents` (`group`, `item_key`, `locale`, `label`, `payload`, `is_active`, `is_approved`, `sort_order`, `created_at`, `updated_at`) VALUES
  ('pregnancy_alert', 'contractions_511', 'fa', 'pregnancy_alert / contractions_511', '{"enabled":true,"level":"urgent","window_days":1,"params":{"interval_max_minutes":5,"duration_min_seconds":45,"run_minutes":60},"title":"انقباض‌ها به الگوی ۵-۱-۱ رسیده","what_we_saw":"در {minutes} دقیقهٔ گذشته {count} انقباض ثبت کردی؛ فاصلهٔ میانگین {interval} و مدت میانگین {duration}","how_sure":"این محاسبه از زمان‌هایی است که خودت ثبت کردی؛ فقط معاینه می‌تواند شروع زایمان را تأیید کند","advice":"وقت تماس با زایشگاه یا مامای خودت است؛ وسایل بیمارستان را آماده کن. اگر کیسهٔ آب پاره شد، خون‌ریزی داری یا حرکات جنین کم شده، منتظر نمان.","actions":[{"key":"call","label":"تماس با زایشگاه"},{"key":"ack","label":"دیدم، ممنون"}],"contact":"اگر خون‌ریزی شدید، درد مداوم یا کم‌شدن حرکات جنین داری، همین حالا با اورژانس (۱۱۵) تماس بگیر."}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pregnancy_alert', 'contractions_511', 'en', 'pregnancy_alert / contractions_511', '{"enabled":true,"level":"urgent","window_days":1,"params":{"interval_max_minutes":5,"duration_min_seconds":45,"run_minutes":60},"title":"Your contractions reached the 5-1-1 pattern","what_we_saw":"You timed {count} contractions in the last {minutes} minutes, on average {interval} apart and {duration} long","how_sure":"This is worked out from the times you logged; only an exam can confirm that labour has started","advice":"It is time to call your maternity unit or midwife and get your hospital bag ready. If your waters break, you are bleeding or the baby moves less, do not wait.","actions":[{"key":"call","label":"Call the maternity unit"},{"key":"ack","label":"Got it, thanks"}],"contact":"If you have heavy bleeding, constant pain or the baby moves less, call emergency services (115) now."}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
-- Removes the seeded rule rows only while nobody edited them (label untouched, updated_at = created_at), like 00005.
DELETE FROM `message_contents`
 WHERE `group` = 'pregnancy_alert' AND `item_key` = 'contractions_511'
   AND `locale` IN ('fa', 'en')
   AND `label` = CONCAT(`group`, ' / ', `item_key`)
   AND `updated_at` <=> `created_at`;
DROP TABLE IF EXISTS `pregnancy_contractions`;
DROP TABLE IF EXISTS `pregnancy_contraction_sessions`;
DROP TABLE IF EXISTS `pregnancy_kick_sessions`;
DROP TABLE IF EXISTS `baby_diapers`;
DROP TABLE IF EXISTS `baby_sleeps`;
DROP TABLE IF EXISTS `baby_feeds`;
