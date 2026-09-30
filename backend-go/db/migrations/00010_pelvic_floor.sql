-- 00010_pelvic_floor.sql — pelvic floor program (CB-PELV-01, roadmap/E07-pelv): the user's 8-week Kegel program,
-- one row per trained day and the bladder diary. Health data: user-scoped (FK cascade on account deletion), no
-- analytics. Levels (hold / rest / reps / sets per week range) are admin-editable catalog content (group
-- `pelvic_levels`, docs/canvas-build/catalog.md §4), seeded here together with `pelvic_alerts` (UTI warning and the
-- program's suitability note) — one migration for schema + seed (the task allows one goose number; the catalog
-- convention's separate `000NN_catalog_<group>.sql` is folded in, same INSERT IGNORE rows).
--
-- Twin of backend/database/migrations/2026_10_01_000001_create_pelvic_tables.php (Laravel owns the prod schema
-- until T-M2-27; it creates the same tables and insertOrIgnore's the same catalog rows), so `make schema-diff`
-- stays green. Spelled the way mariadb-dump prints the Laravel-built tables. IF NOT EXISTS: a database whose
-- Laravel half already created the tables (prod at cutover) must not fail here.
--
-- pelvic_programs     one per user; restarting moves `started_on` (week = days since start / 7 + 1, capped at 8)
-- pelvic_sessions     one row per (user, day): sessions_count / sets_completed / duration_sec accumulate;
--                     level_code = the pelvic_levels code in effect when the last session was saved
-- pelvic_bladder_logs one row per (user, day): leak none|cough|urgency|unexplained, night voids,
--                     uti_symptoms JSON list (burning|frequency|cloudy_odor); a row left empty is deleted

-- +goose Up
CREATE TABLE IF NOT EXISTS `pelvic_programs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `started_on` date NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pelvic_programs_user_id_unique` (`user_id`),
  CONSTRAINT `pelvic_programs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `pelvic_sessions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `session_date` date NOT NULL,
  `sessions_count` smallint(5) unsigned NOT NULL DEFAULT 0,
  `sets_completed` smallint(5) unsigned NOT NULL DEFAULT 0,
  `duration_sec` int(10) unsigned NOT NULL DEFAULT 0,
  `level_code` varchar(64) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pelvic_sessions_user_id_session_date_unique` (`user_id`,`session_date`),
  CONSTRAINT `pelvic_sessions_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `pelvic_bladder_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `leak` varchar(16) DEFAULT NULL,
  `night_voids` tinyint(3) unsigned DEFAULT NULL,
  `uti_symptoms` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pelvic_bladder_logs_user_id_log_date_unique` (`user_id`,`log_date`),
  CONSTRAINT `pelvic_bladder_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Catalog seed ([needs clinical review]: needs_review = 1). meta of pelvic_levels = the admin hint's shape
-- (admin-web/src/screens/catalog/lib/hints.ts): week_from = first program week of the level.
INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('pelvic_levels', 'level_1', 1, 1, NULL, '{"fa":"سطح ۱","en":"Level 1"}',
   '{"fa":"عضلاتی را منقبض کن که با آن جلوی ادرار را می‌گیری. شکم، باسن و ران‌ها شل بمانند و نفست را حبس نکن.","en":"Squeeze the muscles you use to stop the flow of urine. Keep your belly, buttocks and thighs relaxed and don''t hold your breath."}',
   '{"week_from":1,"hold_sec":3,"rest_sec":3,"reps":10,"sets":3}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pelvic_levels', 'level_2', 2, 1, NULL, '{"fa":"سطح ۲","en":"Level 2"}',
   '{"fa":"عضلاتی را منقبض کن که با آن جلوی ادرار را می‌گیری. شکم، باسن و ران‌ها شل بمانند و نفست را حبس نکن.","en":"Squeeze the muscles you use to stop the flow of urine. Keep your belly, buttocks and thighs relaxed and don''t hold your breath."}',
   '{"week_from":3,"hold_sec":5,"rest_sec":5,"reps":10,"sets":3}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pelvic_levels', 'level_3', 3, 1, NULL, '{"fa":"سطح ۳","en":"Level 3"}',
   '{"fa":"عضلاتی را منقبض کن که با آن جلوی ادرار را می‌گیری. شکم، باسن و ران‌ها شل بمانند و نفست را حبس نکن.","en":"Squeeze the muscles you use to stop the flow of urine. Keep your belly, buttocks and thighs relaxed and don''t hold your breath."}',
   '{"week_from":5,"hold_sec":8,"rest_sec":8,"reps":10,"sets":3}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pelvic_levels', 'level_4', 4, 1, NULL, '{"fa":"سطح ۴","en":"Level 4"}',
   '{"fa":"عضلاتی را منقبض کن که با آن جلوی ادرار را می‌گیری. شکم، باسن و ران‌ها شل بمانند و نفست را حبس نکن.","en":"Squeeze the muscles you use to stop the flow of urine. Keep your belly, buttocks and thighs relaxed and don''t hold your breath."}',
   '{"week_from":7,"hold_sec":10,"rest_sec":10,"reps":10,"sets":3}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pelvic_alerts', 'uti_warning', 1, 1, NULL,
   '{"fa":"اگر تب، لرز یا درد پهلو داری، زودتر به پزشک مراجعه کن.","en":"If you have a fever, chills or pain in your side, see a doctor soon."}',
   NULL, '{"severity":"urgent"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pelvic_alerts', 'program_suitability', 2, 1, NULL,
   '{"fa":"مناسب بعد از زایمان، یائسگی و هر وقت نشت ادرار داری. اگر درد لگن یا سنگینی داری، اول با پزشک یا فیزیوتراپ لگن مشورت کن.","en":"Suited to after childbirth, menopause and any time you leak urine. If you have pelvic pain or heaviness, talk to a doctor or pelvic physiotherapist first."}',
   NULL, '{"severity":"info"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'pelvic_levels' AND `code` IN ('level_1', 'level_2', 'level_3', 'level_4');
DELETE FROM `catalog_items` WHERE `group` = 'pelvic_alerts' AND `code` IN ('uti_warning', 'program_suitability');
DROP TABLE IF EXISTS `pelvic_bladder_logs`;
DROP TABLE IF EXISTS `pelvic_sessions`;
DROP TABLE IF EXISTS `pelvic_programs`;
