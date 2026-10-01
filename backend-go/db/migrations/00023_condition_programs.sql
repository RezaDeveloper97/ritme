-- 00023_condition_programs.sql — condition programs (CB-COND-01, roadmap/E05-cond): enrolment in the endometriosis /
-- PMDD / heavy-bleeding / PCOS programs, the pain-diary fields the log taxonomy has no slot for, the PMDD daily
-- questionnaire and the PBAC pad chart. Health data: user-scoped (FK cascade on account deletion), no analytics.
-- No parallel log: the pain locations, the 0–10 score and the relief methods of the pain diary are bloom's log
-- taxonomy (`health_log_entries`, category `pain`, B-N3-01); associated symptoms with a taxonomy slot (catalog
-- `pain_associated` meta.log) and the PBAC clot chips (`bleeding.clots` / `bleeding.clot_size`) are stored there too.
-- PCOS is enrolment only (DECISIONS #8): it reads the existing logs.
--
-- Twin of backend/database/migrations/2026_10_01_000023_create_condition_program_tables.php (Laravel owns the prod
-- schema until T-M2-27; it creates the same tables and insertOrIgnore's the same catalog rows), so `make schema-diff`
-- stays green. Spelled the way mariadb-dump prints the Laravel-built tables. IF NOT EXISTS: a database whose Laravel
-- half already created the tables (prod at cutover) must not fail here. 00021 / 00022 belong to parallel tasks.
--
-- condition_enrolments    one row per (user, program): endo | pmdd | heavy_bleeding | pcos; leaving deletes the row,
--                         the diary data stays
-- condition_pain_entries  one row per (user, day): pain types and associated symptoms without a taxonomy slot (catalog
--                         codes, JSON lists), missed work/school, analgesic name + time (HH:MM) + effect
--                         (no|a_little|helped)
-- pmdd_entries            one row per (user, day): scores JSON {pmdd_items code: 1–6}
-- pbac_entries            one row per (user, day): pads per soak level (light ×1, medium ×5, heavy ×20 points),
--                         flooding (leak through to clothes, 5 points)

-- +goose Up
CREATE TABLE IF NOT EXISTS `condition_enrolments` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `program` varchar(32) NOT NULL,
  `enrolled_on` date NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `condition_enrolments_user_id_program_unique` (`user_id`,`program`),
  CONSTRAINT `condition_enrolments_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `condition_pain_entries` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `entry_date` date NOT NULL,
  `pain_types` json DEFAULT NULL,
  `associated` json DEFAULT NULL,
  `missed_activity` tinyint(1) DEFAULT NULL,
  `analgesic` varchar(100) DEFAULT NULL,
  `analgesic_time` varchar(5) DEFAULT NULL,
  `analgesic_effect` varchar(16) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `condition_pain_entries_user_id_entry_date_unique` (`user_id`,`entry_date`),
  CONSTRAINT `condition_pain_entries_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `pmdd_entries` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `entry_date` date NOT NULL,
  `scores` json NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pmdd_entries_user_id_entry_date_unique` (`user_id`,`entry_date`),
  CONSTRAINT `pmdd_entries_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `pbac_entries` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `entry_date` date NOT NULL,
  `light_count` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `medium_count` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `heavy_count` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `flooding` tinyint(1) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pbac_entries_user_id_entry_date_unique` (`user_id`,`entry_date`),
  CONSTRAINT `pbac_entries_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Catalog seed ([needs clinical review]: needs_review = 1; docs/canvas-build/catalog.md §5). Groups:
--   condition_programs  hub cards endo|pmdd|heavy_bleeding|pcos; meta {logs: {fa, en}} = the «ثبت می‌کنی» line
--   pain_types          pain-diary «چه جور دردی؟» chips (codes stored in condition_pain_entries.pain_types)
--   pain_associated     «همراه با» chips; meta {log: "category.param.item"} = the log-taxonomy slot the choice is
--                       stored in (a multi option or a yes level of an items param); without it the code is stored
--                       in condition_pain_entries.associated
--   pmdd_items          the daily questionnaire items, each scored 1 (not at all) … 6 (extreme)
--   condition_alerts    notes and alerts, meta {severity: info|caution|urgent[, hotlines: [{number, label}]]}
INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('condition_programs', 'endo', 1, 1, NULL,
   '{"fa":"اندومتریوز","en":"Endometriosis"}',
   '{"fa":"دفترچه درد با محل و شدت","en":"A pain diary with location and intensity"}',
   '{"logs":{"fa":"محل درد، شدت، نوع درد، اثر دارو","en":"pain location, intensity, type of pain, painkiller effect"}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_programs', 'pmdd', 2, 1, NULL,
   '{"fa":"اختلال شدید پیش از قاعدگی (PMDD)","en":"Premenstrual dysphoric disorder (PMDD)"}',
   '{"fa":"پرسشنامه روزانه خلق برای ۲ سیکل","en":"A daily mood questionnaire for 2 cycles"}',
   '{"logs":{"fa":"خلق، اضطراب، تحریک‌پذیری، تمرکز","en":"mood, anxiety, irritability, concentration"}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_programs', 'heavy_bleeding', 3, 1, NULL,
   '{"fa":"خونریزی شدید","en":"Heavy bleeding"}',
   '{"fa":"جدول امتیاز خونریزی و خطر کم‌خونی","en":"A bleeding score chart and anaemia risk"}',
   '{"logs":{"fa":"تعداد و میزان خیس شدن نوار، لخته","en":"number of pads and how soaked they are, clots"}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_programs', 'pcos', 4, 1, NULL,
   '{"fa":"سندرم تخمدان پلی‌کیستیک","en":"Polycystic ovary syndrome (PCOS)"}',
   '{"fa":"نظم سیکل، پوست و مو، وزن و قند","en":"Cycle regularity, skin and hair, weight and blood sugar"}',
   '{"logs":{"fa":"فاصله پریودها، آکنه، موهای زائد، وزن","en":"time between periods, acne, unwanted hair, weight"}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pain_types', 'cramping', 1, 1, NULL,
   '{"fa":"گرفتگی","en":"Cramping"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pain_types', 'stabbing', 2, 1, NULL,
   '{"fa":"تیر کشنده","en":"Stabbing"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pain_types', 'burning', 3, 1, NULL,
   '{"fa":"سوزشی","en":"Burning"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pain_types', 'dull_heavy', 4, 1, NULL,
   '{"fa":"مبهم و سنگین","en":"Dull and heavy"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pain_associated', 'dyspareunia', 1, 1, NULL,
   '{"fa":"درد در رابطه","en":"Pain during sex"}',
   NULL,
   '{"log":"sex.symptoms.pain_during_intercourse"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pain_associated', 'dyschezia', 2, 1, NULL,
   '{"fa":"درد هنگام اجابت مزاج","en":"Pain with bowel movements"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pain_associated', 'dysuria', 3, 1, NULL,
   '{"fa":"درد هنگام ادرار","en":"Pain when urinating"}',
   NULL,
   '{"log":"urogenital.symptoms.urination_burning"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pain_associated', 'bloating', 4, 1, NULL,
   '{"fa":"نفخ","en":"Bloating"}',
   NULL,
   '{"log":"symptoms.digestive.bloating"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pain_associated', 'nausea', 5, 1, NULL,
   '{"fa":"تهوع","en":"Nausea"}',
   NULL,
   '{"log":"symptoms.digestive.nausea"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pmdd_items', 'sadness', 1, 1, NULL,
   '{"fa":"غمگینی یا ناامیدی","en":"Sadness or hopelessness"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pmdd_items', 'anxiety', 2, 1, NULL,
   '{"fa":"اضطراب یا تنش","en":"Anxiety or tension"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pmdd_items', 'mood_swings', 3, 1, NULL,
   '{"fa":"نوسان خلق و زودرنجی","en":"Mood swings or feeling easily hurt"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pmdd_items', 'anger', 4, 1, NULL,
   '{"fa":"عصبانیت یا درگیری با دیگران","en":"Anger or conflicts with others"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pmdd_items', 'loss_of_interest', 5, 1, NULL,
   '{"fa":"بی‌علاقگی به کارهای معمول","en":"Less interest in usual activities"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pmdd_items', 'concentration', 6, 1, NULL,
   '{"fa":"سختی در تمرکز","en":"Difficulty concentrating"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_alerts', 'not_a_diagnosis', 1, 1, NULL,
   '{"fa":"این برنامه‌ها تشخیص نمی‌دهند","en":"These programs do not diagnose"}',
   '{"fa":"کمک می‌کنند با داده دقیق نزد پزشک بروی.","en":"They help you see your doctor with accurate data."}',
   '{"severity":"info"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_alerts', 'pain_scale', 2, 1, NULL,
   '{"fa":"شدت درد از ۰ تا ۱۰","en":"Pain from 0 to 10"}',
   '{"fa":"۰ یعنی بدون درد و ۱۰ بدترین درد ممکن. درد شدید یا ناگهانی، تب یا غش را همان روز به پزشک بگو.","en":"0 means no pain and 10 the worst pain possible. Tell a doctor the same day about severe or sudden pain, fever or fainting."}',
   '{"severity":"info"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_alerts', 'pbac_scoring', 3, 1, NULL,
   '{"fa":"امتیاز خونریزی چطور حساب می‌شود؟","en":"How is the bleeding score counted?"}',
   '{"fa":"هر نوار کمی خیس ۱ امتیاز، نیمه خیس ۵ و کاملاً خیس ۲۰ امتیاز دارد؛ لخته کوچک ۱، لخته بزرگ ۵ و نشت از نوار به لباس ۵ امتیاز. جمع یک پریود بالای ۱۰۰ معمولاً یعنی خونریزی شدید.","en":"Each lightly soaked pad scores 1, half soaked 5 and fully soaked 20; a small clot 1, a large clot 5 and leaking through to clothes 5. A period total over 100 usually means heavy bleeding."}',
   '{"severity":"info"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_alerts', 'pbac_over_100', 4, 1, NULL,
   '{"fa":"امتیاز این پریود از ۱۰۰ بیشتر شده","en":"This period\'s score is over 100"}',
   '{"fa":"معمولاً یعنی خونریزی شدید است. به پزشک بگو و درباره آزمایش کم‌خونی (هموگلوبین و فریتین) بپرس.","en":"This usually means heavy bleeding. Tell your doctor and ask about an anaemia test (haemoglobin and ferritin)."}',
   '{"severity":"caution"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_alerts', 'pmdd_needs_two_cycles', 5, 1, NULL,
   '{"fa":"برای جمع‌بندی، ۲ سیکل ثبت کامل لازم است","en":"2 fully logged cycles are needed for a summary"}',
   '{"fa":"یک سیکل کامل یعنی دست‌کم ۷ روز از ۱۰ روز آخر سیکل و ۴ روز از روزهای ۴ تا ۱۰ سیکل را ثبت کرده باشی.","en":"A cycle counts as fully logged when you rated at least 7 of its last 10 days and 4 of its days 4 to 10."}',
   '{"severity":"info"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_alerts', 'pmdd_pattern_luteal', 6, 1, NULL,
   '{"fa":"الگوی تو","en":"Your pattern"}',
   '{"fa":"علائم در ۱۰ روز آخر سیکل بالا می‌رود و با شروع پریود کم می‌شود.","en":"Symptoms rise in the last 10 days of the cycle and ease when your period starts."}',
   '{"severity":"info"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_alerts', 'pmdd_pattern_unclear', 7, 1, NULL,
   '{"fa":"الگوی تو","en":"Your pattern"}',
   '{"fa":"هنوز الگوی روشنی بین ۱۰ روز آخر سیکل و هفته بعد از پریود دیده نمی‌شود. ثبت روزانه را ادامه بده.","en":"There is no clear difference yet between the last 10 days of the cycle and the week after your period. Keep rating every day."}',
   '{"severity":"info"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_alerts', 'pmdd_not_enough_data', 8, 1, NULL,
   '{"fa":"هنوز داده کافی نیست","en":"Not enough data yet"}',
   '{"fa":"برای دیدن الگو، هم در هفته بعد از پریود و هم در ۱۰ روز آخر سیکل هر روز ثبت کن.","en":"To see a pattern, rate every day in the week after your period and in the last 10 days of the cycle."}',
   '{"severity":"info"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('condition_alerts', 'pmdd_crisis', 9, 1, NULL,
   '{"fa":"اگر به آسیب زدن به خودت فکر می‌کنی","en":"If you are thinking about hurting yourself"}',
   '{"fa":"همین حالا با صدای مشاور ۱۴۸۰ یا اورژانس اجتماعی ۱۲۳ تماس بگیر.","en":"Call the 1480 counselling line or the 123 social emergency line right now."}',
   '{"severity":"urgent","hotlines":[{"number":"1480","label":{"fa":"صدای مشاور","en":"Counselling line"}},{"number":"123","label":{"fa":"اورژانس اجتماعی","en":"Social emergency"}}]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'condition_programs' AND `code` IN ('endo', 'pmdd', 'heavy_bleeding', 'pcos');
DELETE FROM `catalog_items` WHERE `group` = 'pain_types' AND `code` IN ('cramping', 'stabbing', 'burning', 'dull_heavy');
DELETE FROM `catalog_items` WHERE `group` = 'pain_associated'
  AND `code` IN ('dyspareunia', 'dyschezia', 'dysuria', 'bloating', 'nausea');
DELETE FROM `catalog_items` WHERE `group` = 'pmdd_items'
  AND `code` IN ('sadness', 'anxiety', 'mood_swings', 'anger', 'loss_of_interest', 'concentration');
DELETE FROM `catalog_items` WHERE `group` = 'condition_alerts'
  AND `code` IN ('not_a_diagnosis', 'pain_scale', 'pbac_scoring', 'pbac_over_100', 'pmdd_needs_two_cycles',
                 'pmdd_pattern_luteal', 'pmdd_pattern_unclear', 'pmdd_not_enough_data', 'pmdd_crisis');
DROP TABLE IF EXISTS `pbac_entries`;
DROP TABLE IF EXISTS `pmdd_entries`;
DROP TABLE IF EXISTS `condition_pain_entries`;
DROP TABLE IF EXISTS `condition_enrolments`;
