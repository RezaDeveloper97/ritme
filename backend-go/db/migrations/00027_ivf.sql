-- 00027_ivf.sql — IVF treatment tracking (CB-IVF-01, roadmap/E03-ivf, deviation D-51). Gives bloom's «IVF/IUI» switch
-- (`user_life_profiles.ivf_iui`, B-N2-03) its meaning: starting a cycle switches it on. Health data: every row is
-- user-scoped (FK cascade on account deletion), no analytics. Built on bloom/M3, never parallel to it:
--   * each IVF medicine IS a care medication reminder (`reminders` type medication, its name/dose/times in the care
--     meta) — the injection reminder, GET /care/today and the companion `meds` section see it as any other medicine;
--   * a dose taken IS a care intake (`reminder_intakes`); `ivf_dose_logs` only adds where it was injected;
--   * scan / retrieval / transfer / beta dates create ordinary care appointments (`reminders` type appointment),
--     linked through `ivf_reminders` — the companion `appointments` section sees them through the B-N4-02 grants.
-- Clinical copy (stages, protocols, the 8 injection sites, medicine presets, screen notes, danger signs) is
-- admin-editable catalog content seeded below (needs_review = 1).
--
-- Twin of backend/database/migrations/2026_10_02_000027_create_ivf_tables.php (Laravel owns the prod schema until
-- T-M2-27; it creates the same tables and insertOrIgnore's the same catalog rows), so `make schema-diff` stays green.
-- Spelled the way mariadb-dump prints the Laravel-built tables. IF NOT EXISTS: a database whose Laravel half already
-- created the tables (prod at cutover) must not fail here.
--
-- ivf_cycles     one row per treatment cycle. active_user_id = user_id while the cycle is open, NULL once closed
--                (the unique key allows one open cycle per user). number = «سیکل اول»; protocol = catalog
--                `ivf_protocols` code; stage prep|stim|retrieval|transfer|tww|test; dates of the stages; next_scan_at;
--                notify_companion = «همدمت هم در جریان باشد»; outcome positive|negative|cancelled when closed.
-- ivf_meds       an IVF medicine: the care reminder it is (reminder_id, unique), its role stimulation|suppression|
--                trigger|luteal_support|other, route subcutaneous|intramuscular|oral|vaginal|other, the trigger's exact
--                time, and the inventory (stock_units of stock_unit as counted at stock_counted_at, doses_per_unit;
--                doses taken since the count come from reminder_intakes).
-- ivf_dose_logs  the injection site of a taken dose (catalog `ivf_injection_sites` code), one per (med, day, slot).
-- ivf_scans      one ultrasound per (cycle, day): follicle counts per ovary per size bin (<10, 10–14, 15–17, ≥18 mm),
--                endometrium thickness (mm) and estradiol (E2) with its unit.
-- ivf_tww_logs   the two-week-wait mood check-in, one per (cycle, day): calm|hopeful|worried|tired.
-- ivf_reminders  the care appointments the cycle dates created, one per (cycle, kind scan|retrieval|transfer|beta).

-- +goose Up
CREATE TABLE IF NOT EXISTS `ivf_cycles` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `active_user_id` bigint(20) unsigned DEFAULT NULL,
  `number` tinyint(3) unsigned NOT NULL DEFAULT 1,
  `protocol` varchar(32) DEFAULT NULL,
  `stage` varchar(16) NOT NULL DEFAULT 'prep',
  `started_on` date NOT NULL,
  `stim_started_on` date DEFAULT NULL,
  `retrieval_at` datetime DEFAULT NULL,
  `transfer_at` datetime DEFAULT NULL,
  `beta_on` date DEFAULT NULL,
  `next_scan_at` datetime DEFAULT NULL,
  `notify_companion` tinyint(1) NOT NULL DEFAULT 0,
  `outcome` varchar(16) DEFAULT NULL,
  `outcome_on` date DEFAULT NULL,
  `closed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ivf_cycles_active_user_id_unique` (`active_user_id`),
  KEY `ivf_cycles_user_id_foreign` (`user_id`),
  CONSTRAINT `ivf_cycles_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ivf_meds` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `cycle_id` bigint(20) unsigned NOT NULL,
  `reminder_id` bigint(20) unsigned NOT NULL,
  `role` varchar(16) NOT NULL,
  `route` varchar(16) NOT NULL,
  `trigger_at` datetime DEFAULT NULL,
  `stock_units` smallint(5) unsigned DEFAULT NULL,
  `stock_unit` varchar(24) DEFAULT NULL,
  `doses_per_unit` smallint(5) unsigned NOT NULL DEFAULT 1,
  `stock_counted_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ivf_meds_reminder_id_unique` (`reminder_id`),
  KEY `ivf_meds_user_id_foreign` (`user_id`),
  KEY `ivf_meds_cycle_id_foreign` (`cycle_id`),
  CONSTRAINT `ivf_meds_cycle_id_foreign` FOREIGN KEY (`cycle_id`) REFERENCES `ivf_cycles` (`id`) ON DELETE CASCADE,
  CONSTRAINT `ivf_meds_reminder_id_foreign` FOREIGN KEY (`reminder_id`) REFERENCES `reminders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `ivf_meds_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ivf_dose_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `ivf_med_id` bigint(20) unsigned NOT NULL,
  `dose_date` date NOT NULL,
  `slot` char(5) NOT NULL,
  `site` varchar(32) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ivf_dose_logs_ivf_med_id_dose_date_slot_unique` (`ivf_med_id`,`dose_date`,`slot`),
  KEY `ivf_dose_logs_user_id_foreign` (`user_id`),
  CONSTRAINT `ivf_dose_logs_ivf_med_id_foreign` FOREIGN KEY (`ivf_med_id`) REFERENCES `ivf_meds` (`id`) ON DELETE CASCADE,
  CONSTRAINT `ivf_dose_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ivf_scans` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `cycle_id` bigint(20) unsigned NOT NULL,
  `scan_date` date NOT NULL,
  `right_lt_10` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `right_10_14` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `right_15_17` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `right_18_plus` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `left_lt_10` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `left_10_14` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `left_15_17` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `left_18_plus` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `endometrium_mm` decimal(4,1) DEFAULT NULL,
  `e2` decimal(8,2) DEFAULT NULL,
  `e2_unit` varchar(8) DEFAULT NULL,
  `notes` varchar(500) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ivf_scans_cycle_id_scan_date_unique` (`cycle_id`,`scan_date`),
  KEY `ivf_scans_user_id_foreign` (`user_id`),
  CONSTRAINT `ivf_scans_cycle_id_foreign` FOREIGN KEY (`cycle_id`) REFERENCES `ivf_cycles` (`id`) ON DELETE CASCADE,
  CONSTRAINT `ivf_scans_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ivf_tww_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `cycle_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `mood` varchar(16) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ivf_tww_logs_cycle_id_log_date_unique` (`cycle_id`,`log_date`),
  KEY `ivf_tww_logs_user_id_foreign` (`user_id`),
  CONSTRAINT `ivf_tww_logs_cycle_id_foreign` FOREIGN KEY (`cycle_id`) REFERENCES `ivf_cycles` (`id`) ON DELETE CASCADE,
  CONSTRAINT `ivf_tww_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ivf_reminders` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `cycle_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(16) NOT NULL,
  `reminder_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ivf_reminders_cycle_id_kind_unique` (`cycle_id`,`kind`),
  KEY `ivf_reminders_user_id_foreign` (`user_id`),
  KEY `ivf_reminders_reminder_id_foreign` (`reminder_id`),
  CONSTRAINT `ivf_reminders_cycle_id_foreign` FOREIGN KEY (`cycle_id`) REFERENCES `ivf_cycles` (`id`) ON DELETE CASCADE,
  CONSTRAINT `ivf_reminders_reminder_id_foreign` FOREIGN KEY (`reminder_id`) REFERENCES `reminders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `ivf_reminders_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Catalog seed ([needs clinical review]: needs_review = 1, audience ttc). meta shapes (read by internal/ivf and the IVF screens):
--   ivf_stages           — (code = ivf_cycles.stage; body = the timeline hint)
--   ivf_protocols        — (code = ivf_cycles.protocol)
--   ivf_injection_sites  {region: abdomen|thigh|arm, side: right|left} (sort_order = rotation order)
--   ivf_med_presets      {role, route, unit, times?: ["HH:MM"], stock_unit} — POST /ivf/meds pre-fill
--   ivf_guidance         {placement: meds|scan|tww}
--   ivf_danger_signs     {severity: urgent, placement: [home|tww], hotlines: ["115"]}
INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('ivf_stages', 'prep', 1, 1, '["ttc"]',
   '{"fa":"آماده‌سازی","en":"Preparation"}',
   '{"fa":"آزمایش‌ها و سونوی پایه","en":"Tests and a baseline scan"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_stages', 'stim', 2, 1, '["ttc"]',
   '{"fa":"تحریک تخمک‌گذاری","en":"Ovarian stimulation"}',
   '{"fa":"تزریق روزانه · حدود ۱۰ تا ۱۲ روز","en":"Daily injections · about 10 to 12 days"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_stages', 'retrieval', 3, 1, '["ttc"]',
   '{"fa":"تخمک‌کشی","en":"Egg retrieval"}',
   '{"fa":"حدود ۳۶ ساعت بعد از تزریق تریگر، طبق برنامه کلینیک","en":"About 36 hours after the trigger shot, as your clinic schedules it"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_stages', 'transfer', 4, 1, '["ttc"]',
   '{"fa":"انتقال جنین","en":"Embryo transfer"}',
   '{"fa":"تاریخ را کلینیک بر اساس رشد جنین تعیین می‌کند","en":"Your clinic sets the date from how the embryos develop"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_stages', 'tww', 5, 1, '["ttc"]',
   '{"fa":"انتظار دو هفته‌ای","en":"Two-week wait"}',
   '{"fa":"داروها را طبق نسخه ادامه بده","en":"Keep taking your medicines as prescribed"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_stages', 'test', 6, 1, '["ttc"]',
   '{"fa":"تست بارداری","en":"Pregnancy test"}',
   '{"fa":"آزمایش خون بتا در تاریخی که کلینیک گفته","en":"A beta blood test on the day your clinic gave you"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_protocols', 'antagonist', 1, 1, '["ttc"]',
   '{"fa":"آنتاگونیست","en":"Antagonist"}',
   '{"fa":"رایج‌ترین پروتکل؛ تحریک از روزهای اول سیکل و داروی جلوگیری از تخمک‌گذاری زودرس از حدود روز پنجم تحریک","en":"The most common protocol: stimulation from early in the cycle, with a medicine that prevents early ovulation from about day 5 of stimulation"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_protocols', 'long_agonist', 2, 1, '["ttc"]',
   '{"fa":"آگونیست طولانی","en":"Long agonist"}',
   '{"fa":"سرکوب هورمونی از سیکل قبل، بعد تحریک","en":"Hormones are switched off from the cycle before, then stimulation starts"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_protocols', 'short_agonist', 3, 1, '["ttc"]',
   '{"fa":"آگونیست کوتاه","en":"Short agonist"}',
   '{"fa":"آگونیست و تحریک تقریباً هم‌زمان شروع می‌شوند","en":"The agonist and stimulation start at about the same time"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_protocols', 'mild', 4, 1, '["ttc"]',
   '{"fa":"تحریک ملایم","en":"Mild stimulation"}',
   '{"fa":"دوز کمتر دارو و معمولاً تخمک کمتر","en":"Lower doses and usually fewer eggs"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_protocols', 'natural', 5, 1, '["ttc"]',
   '{"fa":"سیکل طبیعی","en":"Natural cycle"}',
   '{"fa":"بدون تحریک یا با داروی خیلی کم","en":"No or very little stimulation"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_protocols', 'frozen_transfer', 6, 1, '["ttc"]',
   '{"fa":"انتقال جنین منجمد","en":"Frozen embryo transfer"}',
   '{"fa":"آماده‌سازی رحم برای انتقال جنین فریزشده","en":"Preparing the womb lining to transfer a frozen embryo"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_injection_sites', 'abdomen_upper_right', 1, 1, '["ttc"]',
   '{"fa":"شکم · راست بالا","en":"Abdomen · upper right"}',
   '{"fa":"حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن","en":"At least 5 cm away from the navel; avoid bruises and the last spot"}',
   '{"region":"abdomen","side":"right"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_injection_sites', 'abdomen_upper_left', 2, 1, '["ttc"]',
   '{"fa":"شکم · چپ بالا","en":"Abdomen · upper left"}',
   '{"fa":"حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن","en":"At least 5 cm away from the navel; avoid bruises and the last spot"}',
   '{"region":"abdomen","side":"left"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_injection_sites', 'thigh_right', 3, 1, '["ttc"]',
   '{"fa":"ران · راست","en":"Thigh · right"}',
   '{"fa":"جلو و بیرون ران، میانه فاصله زانو تا لگن","en":"Front and outer thigh, halfway between knee and hip"}',
   '{"region":"thigh","side":"right"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_injection_sites', 'thigh_left', 4, 1, '["ttc"]',
   '{"fa":"ران · چپ","en":"Thigh · left"}',
   '{"fa":"جلو و بیرون ران، میانه فاصله زانو تا لگن","en":"Front and outer thigh, halfway between knee and hip"}',
   '{"region":"thigh","side":"left"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_injection_sites', 'abdomen_lower_right', 5, 1, '["ttc"]',
   '{"fa":"شکم · راست پایین","en":"Abdomen · lower right"}',
   '{"fa":"حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن","en":"At least 5 cm away from the navel; avoid bruises and the last spot"}',
   '{"region":"abdomen","side":"right"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_injection_sites', 'abdomen_lower_left', 6, 1, '["ttc"]',
   '{"fa":"شکم · چپ پایین","en":"Abdomen · lower left"}',
   '{"fa":"حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن","en":"At least 5 cm away from the navel; avoid bruises and the last spot"}',
   '{"region":"abdomen","side":"left"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_injection_sites', 'arm_right', 7, 1, '["ttc"]',
   '{"fa":"بازو · راست","en":"Arm · right"}',
   '{"fa":"پشت بازو؛ معمولاً وقتی کسی کمکت می‌کند","en":"Back of the upper arm; usually when someone helps you"}',
   '{"region":"arm","side":"right"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_injection_sites', 'arm_left', 8, 1, '["ttc"]',
   '{"fa":"بازو · چپ","en":"Arm · left"}',
   '{"fa":"پشت بازو؛ معمولاً وقتی کسی کمکت می‌کند","en":"Back of the upper arm; usually when someone helps you"}',
   '{"region":"arm","side":"left"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_med_presets', 'fsh', 1, 1, '["ttc"]',
   '{"fa":"هورمون تحریک (FSH)","en":"Stimulation hormone (FSH)"}',
   NULL,
   '{"role":"stimulation","route":"subcutaneous","unit":"iu","times":["20:00"],"stock_unit":"pen"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_med_presets', 'hmg', 2, 1, '["ttc"]',
   '{"fa":"hMG (FSH + LH)","en":"hMG (FSH + LH)"}',
   NULL,
   '{"role":"stimulation","route":"subcutaneous","unit":"iu","times":["20:00"],"stock_unit":"vial"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_med_presets', 'gnrh_antagonist', 3, 1, '["ttc"]',
   '{"fa":"داروی جلوگیری از تخمک‌گذاری زودرس (آنتاگونیست)","en":"Medicine to prevent early ovulation (antagonist)"}',
   NULL,
   '{"role":"suppression","route":"subcutaneous","unit":"mg","times":["08:00"],"stock_unit":"prefilled_syringe"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_med_presets', 'gnrh_agonist', 4, 1, '["ttc"]',
   '{"fa":"آگونیست GnRH","en":"GnRH agonist"}',
   NULL,
   '{"role":"suppression","route":"subcutaneous","unit":"mg","times":["08:00"],"stock_unit":"vial"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_med_presets', 'hcg_trigger', 5, 1, '["ttc"]',
   '{"fa":"تزریق تریگر (hCG)","en":"Trigger shot (hCG)"}',
   NULL,
   '{"role":"trigger","route":"subcutaneous","unit":"iu","stock_unit":"prefilled_syringe"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_med_presets', 'agonist_trigger', 6, 1, '["ttc"]',
   '{"fa":"تزریق تریگر (آگونیست)","en":"Trigger shot (agonist)"}',
   NULL,
   '{"role":"trigger","route":"subcutaneous","unit":"mg","stock_unit":"prefilled_syringe"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_med_presets', 'progesterone_vaginal', 7, 1, '["ttc"]',
   '{"fa":"پروژسترون واژینال","en":"Vaginal progesterone"}',
   NULL,
   '{"role":"luteal_support","route":"vaginal","unit":"mg","times":["08:00","20:00"],"stock_unit":"box"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_med_presets', 'progesterone_im', 8, 1, '["ttc"]',
   '{"fa":"پروژسترون تزریقی","en":"Progesterone injection"}',
   NULL,
   '{"role":"luteal_support","route":"intramuscular","unit":"mg","times":["20:00"],"stock_unit":"ampoule"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_med_presets', 'estradiol', 9, 1, '["ttc"]',
   '{"fa":"استرادیول","en":"Estradiol"}',
   NULL,
   '{"role":"luteal_support","route":"oral","unit":"mg","times":["08:00","20:00"],"stock_unit":"box"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_guidance', 'trigger_timing', 1, 1, '["ttc"]',
   '{"fa":"تزریق تریگر (آخرین تزریق)","en":"Trigger shot (the last injection)"}',
   '{"fa":"وقتی پزشک اعلام کرد، دقیقاً در همان ساعت تزریق کن؛ زمان آن برای تخمک‌کشی مهم است.","en":"Inject at exactly the time your doctor gives you; its timing matters for the egg retrieval."}',
   '{"placement":"meds"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_guidance', 'site_rotation', 2, 1, '["ttc"]',
   '{"fa":"محل تزریق را عوض کن","en":"Change the injection spot"}',
   '{"fa":"هر بار جای دیگری تزریق کن تا پوست تحریک و کبود نشود. در شکم حداقل ۵ سانت از ناف فاصله بگیر.","en":"Use a different spot each time so the skin doesn''t get sore or bruised. On the abdomen keep at least 5 cm from the navel."}',
   '{"placement":"meds"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_guidance', 'scan_interpretation', 3, 1, '["ttc"]',
   '{"fa":"تفسیر با پزشک است","en":"Your doctor interprets it"}',
   '{"fa":"تفسیر و تصمیم درباره زمان تخمک‌کشی با پزشکت است. ما فقط کمک می‌کنیم همه‌چیز یک‌جا ثبت شود.","en":"Interpreting the scan and deciding when to retrieve is your doctor''s call. We only help you keep everything in one place."}',
   '{"placement":"scan"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_guidance', 'tww_feelings', 4, 1, '["ttc"]',
   '{"fa":"هر حسی داری طبیعی است","en":"Whatever you feel is normal"}',
   '{"fa":"هر حسی داری طبیعی است. تست خانگی زودتر از موعد می‌تواند گمراه‌کننده باشد.","en":"Whatever you feel is normal. A home test taken too early can be misleading."}',
   '{"placement":"tww"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_guidance', 'early_test', 5, 1, '["ttc"]',
   '{"fa":"تست خانگی زودهنگام","en":"Testing early at home"}',
   '{"fa":"داروهای تریگر و پروژسترون می‌توانند جواب تست خانگی را در روزهای اول اشتباه نشان دهند؛ منتظر آزمایش خون بتا بمان.","en":"Trigger and progesterone medicines can make an early home test misleading; wait for the beta blood test."}',
   '{"placement":"tww"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_danger_signs', 'ohss', 1, 1, '["ttc"]',
   '{"fa":"اگر این‌ها را داشتی فوراً به پزشک خبر بده","en":"Tell your doctor straight away if you have"}',
   '{"fa":"نفخ شدید و سریع شکم، تنگی نفس، کم شدن ادرار، درد شدید شکم، یا خونریزی زیاد.","en":"Severe or fast-growing bloating, shortness of breath, passing much less urine, severe tummy pain, or heavy bleeding."}',
   '{"severity":"urgent","placement":["home","tww"],"hotlines":["115"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('ivf_danger_signs', 'fever_after_procedure', 2, 1, '["ttc"]',
   '{"fa":"تب بعد از تخمک‌کشی یا انتقال","en":"Fever after retrieval or transfer"}',
   '{"fa":"تب، لرز یا ترشح بدبو بعد از تخمک‌کشی یا انتقال جنین را همان روز به کلینیک بگو.","en":"Report a fever, chills or smelly discharge after the retrieval or transfer to your clinic the same day."}',
   '{"severity":"urgent","placement":["home","tww"],"hotlines":["115"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'ivf_stages'
  AND `code` IN ('prep', 'stim', 'retrieval', 'transfer', 'tww', 'test');
DELETE FROM `catalog_items` WHERE `group` = 'ivf_protocols'
  AND `code` IN ('antagonist', 'long_agonist', 'short_agonist', 'mild', 'natural', 'frozen_transfer');
DELETE FROM `catalog_items` WHERE `group` = 'ivf_injection_sites'
  AND `code` IN ('abdomen_upper_right', 'abdomen_upper_left', 'thigh_right', 'thigh_left', 'abdomen_lower_right', 'abdomen_lower_left', 'arm_right', 'arm_left');
DELETE FROM `catalog_items` WHERE `group` = 'ivf_med_presets'
  AND `code` IN ('fsh', 'hmg', 'gnrh_antagonist', 'gnrh_agonist', 'hcg_trigger', 'agonist_trigger', 'progesterone_vaginal', 'progesterone_im', 'estradiol');
DELETE FROM `catalog_items` WHERE `group` = 'ivf_guidance'
  AND `code` IN ('trigger_timing', 'site_rotation', 'scan_interpretation', 'tww_feelings', 'early_test');
DELETE FROM `catalog_items` WHERE `group` = 'ivf_danger_signs'
  AND `code` IN ('ohss', 'fever_after_procedure');
DROP TABLE IF EXISTS `ivf_reminders`;
DROP TABLE IF EXISTS `ivf_tww_logs`;
DROP TABLE IF EXISTS `ivf_scans`;
DROP TABLE IF EXISTS `ivf_dose_logs`;
DROP TABLE IF EXISTS `ivf_meds`;
DROP TABLE IF EXISTS `ivf_cycles`;
