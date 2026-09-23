-- 00003_checkups.sql — periodic checkups: admin catalog, user records, per-type settings (T-M4-01,
-- docs/checkups/README.md).
--
-- Twin of backend/database/migrations/2026_09_24_000001_create_checkup_tables.php
-- (Laravel owns the prod schema until T-M2-27); `make schema-diff` proves both build the same tables
-- and seed the same number of catalog rows. Spelled the way mariadb-dump prints the Laravel-built
-- tables (JSON columns written as `json`, see migrations.md). The tables use IF NOT EXISTS and the
-- seed is idempotent on the unique `key`: a database whose Laravel half already created and seeded
-- them (prod at cutover, T-M2-27: baseline stamped, then migrations after 00001 applied) must not
-- fail or duplicate rows here.
--
-- The default catalog copy needs a medical review before production.

-- +goose Up
CREATE TABLE IF NOT EXISTS `checkup_types` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(64) DEFAULT NULL,
  `user_id` bigint(20) unsigned DEFAULT NULL,
  `category` varchar(20) NOT NULL,
  `title` json NOT NULL,
  `subtitle` json DEFAULT NULL,
  `why` json DEFAULT NULL,
  `performed_by` varchar(10) NOT NULL,
  `icon` varchar(40) DEFAULT NULL,
  `tone` varchar(10) NOT NULL DEFAULT 'neutral',
  `interval_months` smallint(5) unsigned NOT NULL,
  `interval_months_max` smallint(5) unsigned DEFAULT NULL,
  `age_min` tinyint(3) unsigned DEFAULT NULL,
  `age_max` tinyint(3) unsigned DEFAULT NULL,
  `cycle_day_from` tinyint(3) unsigned DEFAULT NULL,
  `cycle_day_to` tinyint(3) unsigned DEFAULT NULL,
  `remind_lead_days` smallint(5) unsigned NOT NULL DEFAULT 7,
  `prep_steps` json DEFAULT NULL,
  `guide_steps` json DEFAULT NULL,
  `finding_options` json DEFAULT NULL,
  `hide_in_pregnancy` tinyint(1) NOT NULL DEFAULT 0,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `source_note` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `checkup_types_key_unique` (`key`),
  KEY `checkup_types_user_id_foreign` (`user_id`),
  KEY `checkup_types_is_active_sort_order_index` (`is_active`,`sort_order`),
  CONSTRAINT `checkup_types_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `checkup_records` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `checkup_type_id` bigint(20) unsigned NOT NULL,
  `done_on` date NOT NULL,
  `result` varchar(10) NOT NULL,
  `findings` json DEFAULT NULL,
  `note` text DEFAULT NULL,
  `has_attachment` tinyint(1) NOT NULL DEFAULT 0,
  `next_due_on` date DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `checkup_records_checkup_type_id_foreign` (`checkup_type_id`),
  KEY `checkup_records_user_id_checkup_type_id_done_on_index` (`user_id`,`checkup_type_id`,`done_on`),
  CONSTRAINT `checkup_records_checkup_type_id_foreign` FOREIGN KEY (`checkup_type_id`) REFERENCES `checkup_types` (`id`) ON DELETE CASCADE,
  CONSTRAINT `checkup_records_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `user_checkup_settings` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `checkup_type_id` bigint(20) unsigned NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT 1,
  `remind` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_checkup_settings_user_id_checkup_type_id_unique` (`user_id`,`checkup_type_id`),
  KEY `user_checkup_settings_checkup_type_id_foreign` (`checkup_type_id`),
  CONSTRAINT `user_checkup_settings_checkup_type_id_foreign` FOREIGN KEY (`checkup_type_id`) REFERENCES `checkup_types` (`id`) ON DELETE CASCADE,
  CONSTRAINT `user_checkup_settings_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Default catalog: the same values as the Laravel twin (generated from one source), Tehran wall-clock
-- timestamps like the baseline's `languages` seed.
INSERT IGNORE INTO `checkup_types` (`key`, `category`, `title`, `subtitle`, `why`, `performed_by`, `icon`, `tone`, `interval_months`, `interval_months_max`, `age_min`, `age_max`, `cycle_day_from`, `cycle_day_to`, `remind_lead_days`, `prep_steps`, `guide_steps`, `finding_options`, `hide_in_pregnancy`, `is_active`, `sort_order`, `source_note`, `created_at`, `updated_at`) VALUES
  ('breast_self_exam', 'monthly', '{"fa":"خودآزمایی سینه","en":"Breast self-exam"}', '{"fa":"چند دقیقه در خانه، بعد از پریود","en":"A few minutes at home, after your period"}', '{"fa":"وقتی حالت طبیعی سینه‌هایت را بشناسی، هر تغییری را زودتر می‌بینی. بهترین زمان چند روز بعد از پریود است؛ سینه‌ها کمتر حساس و متورم‌اند.","en":"Knowing how your breasts normally feel helps you notice any change early. The best time is a few days after your period, when breasts are less tender and swollen."}', 'self', 'ribbon', 'rose', 1, NULL, NULL, NULL, 7, 10, 3, '[{"fa":"روز ۷ تا ۱۰ سیکل، چند روز بعد از پریود را انتخاب کن","en":"Pick cycle day 7 to 10, a few days after your period"},{"fa":"حدود ۵ دقیقه وقت آرام و یک آینه کافی است","en":"About 5 quiet minutes and a mirror are enough"}]', '[{"title":{"fa":"جلوی آینه","en":"In front of a mirror"},"body":{"fa":"با دست‌ها پایین و بعد بالا، به تغییر شکل، اندازه، فرورفتگی یا تغییر پوست نگاه کن.","en":"With your arms down and then raised, look for changes in shape, size, dimpling or skin."}},{"title":{"fa":"ایستاده یا زیر دوش","en":"Standing or in the shower"},"body":{"fa":"با سه انگشت میانی و فشار ملایم تا محکم، به‌صورت دایره‌ای کل سینه و زیر بغل را لمس کن.","en":"Using your three middle fingers and light to firm pressure, feel the whole breast and armpit in small circles."}},{"title":{"fa":"دراز کشیده","en":"Lying down"},"body":{"fa":"بالشی زیر شانه بگذار و همان حرکت را تکرار کن؛ نوک سینه را هم به‌آرامی فشار بده.","en":"Put a pillow under your shoulder and repeat the same motion; gently squeeze the nipple too."}}]', '[{"key":"none","exclusive":true,"label":{"fa":"چیزی متفاوت نبود","en":"Nothing different"}},{"key":"lump","label":{"fa":"توده یا سفتی","en":"Lump or thickening"}},{"key":"skin_change","label":{"fa":"تغییر پوست","en":"Skin change"}},{"key":"discharge","label":{"fa":"ترشح","en":"Discharge"}},{"key":"pain","label":{"fa":"درد","en":"Pain"}}]', 1, 1, 1, 'Default catalog (T-M4-01). Needs medical review before production.', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('clinical_breast_exam', 'annual', '{"fa":"معاینه بالینی سینه","en":"Clinical breast exam"}', '{"fa":"توسط پزشک","en":"By a doctor"}', '{"fa":"پزشک یا ماما در معاینه بالینی تغییراتی را بررسی می‌کند که ممکن است در خودآزمایی دیده نشوند. معمولاً سالی یک‌بار، همراه با چکاپ عمومی انجام می‌شود.","en":"In a clinical exam a doctor or midwife checks for changes you may not notice yourself. It is usually done once a year, together with a general checkup."}', 'doctor', 'breast', 'violet', 12, NULL, NULL, NULL, NULL, NULL, 30, '[{"fa":"بهترین زمان حدود یک هفته بعد از پریود است","en":"The best time is about a week after your period"},{"fa":"هر تغییری که در خودآزمایی دیدی را یادداشت کن و همراه ببر","en":"Note any change you found in your self-exams and bring it along"}]', NULL, NULL, 0, 1, 2, 'Default catalog (T-M4-01). Needs medical review before production.', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pap_smear', 'multi_year', '{"fa":"پاپ‌اسمیر / HPV","en":"Pap smear / HPV"}', '{"fa":"غربالگری دهانه رحم","en":"Cervical screening"}', '{"fa":"پاپ‌اسمیر تغییرات سلول‌های دهانه رحم را پیش از آنکه به مشکل جدی تبدیل شوند نشان می‌دهد. برای بیشتر زنان ۲۱ تا ۶۵ ساله هر ۳ سال یک‌بار توصیه می‌شود؛ همراه با تست HPV می‌تواند هر ۵ سال شود.","en":"A Pap smear shows changes in cervical cells before they become a serious problem. For most women aged 21 to 65 it is recommended every 3 years; together with an HPV test it can be every 5 years."}', 'doctor', 'flask', 'violet', 36, NULL, 21, 65, 10, 20, 30, '[{"fa":"بهترین زمان: وسط سیکل، نه در روزهای پریود","en":"Best time: mid-cycle, not during your period"},{"fa":"۴۸ ساعت قبل از رابطه، دوش واژینال و کرم‌های واژینال پرهیز کن","en":"Avoid sex, douching and vaginal creams for 48 hours before"},{"fa":"جواب معمولاً ۱ تا ۳ هفته بعد آماده می‌شود","en":"Results are usually ready 1 to 3 weeks later"}]', NULL, NULL, 0, 1, 3, 'Default catalog (T-M4-01). Needs medical review before production.', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('blood_test', 'annual', '{"fa":"آزمایش خون کامل","en":"Full blood test"}', '{"fa":"CBC، تیروئید، ویتامین D، آهن","en":"CBC, thyroid, vitamin D, iron"}', '{"fa":"آزمایش خون سالانه کم‌خونی، کمبود آهن و ویتامین D و مشکلات تیروئید را نشان می‌دهد که در زنان شایع‌اند و اغلب بی‌علامت شروع می‌شوند.","en":"A yearly blood test shows anaemia, low iron and vitamin D, and thyroid problems, which are common in women and often start without symptoms."}', 'lab', 'blood', 'amber', 12, NULL, NULL, NULL, NULL, NULL, 14, '[{"fa":"اگر پزشک گفته، ۱۰ تا ۱۲ ساعت ناشتا باش","en":"If your doctor asked, fast for 10 to 12 hours"},{"fa":"فهرست داروها و مکمل‌هایت را همراه داشته باش","en":"Bring a list of your medicines and supplements"},{"fa":"آب کافی بنوش تا خون‌گیری راحت‌تر شود","en":"Drink enough water so the blood draw is easier"}]', NULL, NULL, 0, 1, 4, 'Default catalog (T-M4-01). Needs medical review before production.', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('dentist', 'six_monthly', '{"fa":"دندان‌پزشکی","en":"Dentist"}', '{"fa":"معاینه و جرم‌گیری","en":"Check-up and cleaning"}', '{"fa":"معاینه و جرم‌گیری منظم از پوسیدگی و بیماری لثه پیشگیری می‌کند. تغییرات هورمونی سیکل و بارداری هم می‌تواند لثه‌ها را حساس‌تر کند.","en":"Regular check-ups and cleaning prevent decay and gum disease. Hormonal changes during the cycle and pregnancy can also make gums more sensitive."}', 'dentist', 'tooth', 'teal', 6, NULL, NULL, NULL, NULL, NULL, 14, '[{"fa":"اگر جایی از دندان یا لثه‌ات درد یا حساسیت دارد یادداشت کن","en":"Note any tooth or gum that hurts or feels sensitive"},{"fa":"اگر باردار هستی یا احتمالش را می‌دهی به دندان‌پزشک بگو","en":"Tell your dentist if you are or might be pregnant"}]', NULL, NULL, 0, 1, 5, 'Default catalog (T-M4-01). Needs medical review before production.', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('mammography', 'age_based', '{"fa":"ماموگرافی","en":"Mammography"}', '{"fa":"غربالگری سرطان سینه","en":"Breast cancer screening"}', '{"fa":"ماموگرافی می‌تواند توده‌هایی را سال‌ها پیش از آنکه لمس شوند نشان دهد. برای بیشتر زنان از ۴۰ سالگی هر ۱ تا ۲ سال توصیه می‌شود؛ با سابقه خانوادگی ممکن است پزشک زودتر شروع کند.","en":"A mammogram can show lumps years before they can be felt. For most women it is recommended every 1 to 2 years from age 40; with a family history your doctor may start earlier."}', 'lab', 'shieldCheck', 'rose', 12, 24, 40, NULL, NULL, NULL, 30, '[{"fa":"روز معاینه دئودورانت، پودر یا لوسیون روی سینه و زیر بغل نزن","en":"On the day, skip deodorant, powder or lotion on your breasts and underarms"},{"fa":"حدود یک هفته بعد از پریود که سینه‌ها کمتر حساس‌اند بهترین زمان است","en":"About a week after your period, when breasts are less tender, is the best time"},{"fa":"اگر ماموگرافی قبلی داری همراه ببر","en":"Bring any previous mammograms"}]', NULL, NULL, 1, 1, 6, 'Default catalog (T-M4-01). Needs medical review before production.', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DROP TABLE IF EXISTS `user_checkup_settings`;
DROP TABLE IF EXISTS `checkup_records`;
DROP TABLE IF EXISTS `checkup_types`;
