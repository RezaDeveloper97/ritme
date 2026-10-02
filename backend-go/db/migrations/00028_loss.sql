-- 00028_loss.sql — pregnancy loss path (CB-LOSS-01, roadmap/E04-loss; boards nbl_Loss_Start / _Care / _Next): the loss
-- event that durably stops pregnancy content, the follow-up state (bleeding until it stops, beta hCG until negative,
-- a visit about 2 weeks later — the dated ones are M3 care appointments in `reminders`), daily mood check-ins, an
-- encrypted private note and the chosen next step. Health data of the most sensitive kind: user-scoped (FK cascade on
-- account deletion), never shared with a companion (only the optional one-line notice), never logged or counted.
--
-- Twin of backend/database/migrations/2026_10_02_000028_create_pregnancy_loss_tables.php (Laravel owns the prod schema
-- until T-M2-27; it creates the same tables and insertOrIgnore's the same catalog rows), so `make schema-diff` stays
-- green. Spelled the way mariadb-dump prints the Laravel-built tables. IF NOT EXISTS: a database whose Laravel half
-- already created the tables (prod at cutover) must not fail here. 00027 belongs to a parallel task.
--
-- pregnancy_losses       one row per loss event: loss_type early_miscarriage|late_miscarriage|ectopic|chemical|
--                        unspecified (validated in Go), approximate occurred_on, the companion-notice choice and when
--                        it was sent, when pregnancy content was stopped and which reminders that paused (JSON ids),
--                        follow-up state (bleeding_stopped_on, beta_next_on / beta_negative_on, the care appointments
--                        created for the beta test and the visit — SET NULL when the user deletes them), next_step
--                        cycle|ttc|nothing, private_note = AES-256-GCM ciphertext (PRIVATE_NOTE_KEY, bound to the
--                        user and row; returned to nobody but the owner).
-- pregnancy_loss_moods   one mood check-in per (loss, day): sad|numb|angry|a_bit_better.

-- +goose Up
CREATE TABLE IF NOT EXISTS `pregnancy_losses` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `loss_type` varchar(32) NOT NULL DEFAULT 'unspecified',
  `occurred_on` date DEFAULT NULL,
  `notify_companion` tinyint(1) NOT NULL DEFAULT 0,
  `companion_notified_at` timestamp NULL DEFAULT NULL,
  `content_stopped_at` timestamp NULL DEFAULT NULL,
  `paused_reminders` json DEFAULT NULL,
  `bleeding_stopped_on` date DEFAULT NULL,
  `beta_next_on` date DEFAULT NULL,
  `beta_negative_on` date DEFAULT NULL,
  `beta_reminder_id` bigint(20) unsigned DEFAULT NULL,
  `visit_reminder_id` bigint(20) unsigned DEFAULT NULL,
  `next_step` varchar(16) DEFAULT NULL,
  `next_step_at` timestamp NULL DEFAULT NULL,
  `private_note` text DEFAULT NULL,
  `note_updated_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `pregnancy_losses_beta_reminder_id_foreign` (`beta_reminder_id`),
  KEY `pregnancy_losses_visit_reminder_id_foreign` (`visit_reminder_id`),
  KEY `pregnancy_losses_user_id_created_at_index` (`user_id`,`created_at`),
  CONSTRAINT `pregnancy_losses_beta_reminder_id_foreign` FOREIGN KEY (`beta_reminder_id`) REFERENCES `reminders` (`id`) ON DELETE SET NULL,
  CONSTRAINT `pregnancy_losses_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `pregnancy_losses_visit_reminder_id_foreign` FOREIGN KEY (`visit_reminder_id`) REFERENCES `reminders` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `pregnancy_loss_moods` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `loss_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `mood` varchar(16) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pregnancy_loss_moods_loss_id_log_date_unique` (`loss_id`,`log_date`),
  KEY `pregnancy_loss_moods_user_id_log_date_index` (`user_id`,`log_date`),
  CONSTRAINT `pregnancy_loss_moods_loss_id_foreign` FOREIGN KEY (`loss_id`) REFERENCES `pregnancy_losses` (`id`) ON DELETE CASCADE,
  CONSTRAINT `pregnancy_loss_moods_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Catalog seed ([needs clinical review]: needs_review = 1; docs/canvas-build/catalog.md §5). No audience filter (a
-- loss can be recorded from any mode). Groups:
--   loss_types          «چه اتفاقی افتاد؟» options (codes = pregnancy_losses.loss_type)
--   loss_warning_signs  «این علائم را فوراً پیگیری کن»; meta {severity: urgent, hotline: "115"}
--   loss_hotlines       emergency 115, counselling 1480, social emergency 123; meta {number, kind}
--   loss_followups      the «پیگیری جسمی» rows bleeding | beta | visit
--   loss_moods          «امروز چطوری؟» chips (codes = pregnancy_loss_moods.mood)
--   loss_support        mood_support, crisis (meta hotlines), recurrent_hint (≥ 2 losses), companion_notice (the one
--                       line a companion with a pregnancy grant gets; {name} = the owner, meta.someone without a name)
--   loss_next_steps     cycle | ttc | nothing; meta {life_mode} = the mode the choice switches to
INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('loss_types', 'early_miscarriage', 1, 1, NULL,
   '{"fa":"سقط در ۳ ماه اول","en":"Miscarriage in the first 3 months"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_types', 'late_miscarriage', 2, 1, NULL,
   '{"fa":"سقط بعد از ۳ ماه اول","en":"Miscarriage after the first 3 months"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_types', 'ectopic', 3, 1, NULL,
   '{"fa":"حاملگی خارج از رحم","en":"Ectopic pregnancy"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_types', 'chemical', 4, 1, NULL,
   '{"fa":"بارداری شیمیایی یا تست مثبت کوتاه","en":"Chemical pregnancy or a brief positive test"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_types', 'unspecified', 5, 1, NULL,
   '{"fa":"ترجیح می‌دهم نگویم","en":"I''d rather not say"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_warning_signs', 'heavy_bleeding', 1, 1, NULL,
   '{"fa":"خونریزی خیلی زیاد","en":"Very heavy bleeding"}',
   '{"fa":"پر شدن ۲ نوار در ساعت، ۲ ساعت پشت سر هم","en":"Soaking 2 pads an hour, 2 hours in a row"}',
   '{"severity":"urgent","hotline":"115"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_warning_signs', 'fever', 2, 1, NULL,
   '{"fa":"تب ۳۸ درجه یا بیشتر","en":"A fever of 38 °C or higher"}',
   NULL,
   '{"severity":"urgent","hotline":"115"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_warning_signs', 'severe_pain', 3, 1, NULL,
   '{"fa":"درد شدید شکم","en":"Severe pain in the abdomen"}',
   NULL,
   '{"severity":"urgent","hotline":"115"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_warning_signs', 'dizziness_fainting', 4, 1, NULL,
   '{"fa":"سرگیجه یا غش","en":"Dizziness or fainting"}',
   NULL,
   '{"severity":"urgent","hotline":"115"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_warning_signs', 'foul_discharge', 5, 1, NULL,
   '{"fa":"ترشح بدبو","en":"Foul-smelling discharge"}',
   NULL,
   '{"severity":"urgent","hotline":"115"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_hotlines', 'emergency', 1, 1, NULL,
   '{"fa":"اورژانس ۱۱۵","en":"Emergency 115"}',
   '{"fa":"برای علائم خطر جسمی، همین حالا تماس بگیر","en":"For any of the warning signs, call now"}',
   '{"number":"115","kind":"medical"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_hotlines', 'counselling', 2, 1, NULL,
   '{"fa":"صدای مشاور ۱۴۸۰","en":"Counselling line 1480"}',
   '{"fa":"مشاوره تلفنی برای وقتی که حالت سنگین است","en":"Phone counselling for when things feel heavy"}',
   '{"number":"1480","kind":"mental_health"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_hotlines', 'social_emergency', 3, 1, NULL,
   '{"fa":"اورژانس اجتماعی ۱۲۳","en":"Social emergency 123"}',
   '{"fa":"اگر به آسیب زدن به خودت فکر می‌کنی","en":"If you are thinking about hurting yourself"}',
   '{"number":"123","kind":"crisis"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_followups', 'bleeding', 1, 1, NULL,
   '{"fa":"خونریزی","en":"Bleeding"}',
   '{"fa":"ثبت روزانه تا قطع شدن","en":"Log it every day until it stops"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_followups', 'beta', 2, 1, NULL,
   '{"fa":"آزمایش بتا تا منفی شدن","en":"Beta hCG test until negative"}',
   '{"fa":"تکرار آزمایش را با پزشکت هماهنگ کن","en":"Plan the repeat tests with your doctor"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_followups', 'visit', 3, 1, NULL,
   '{"fa":"ویزیت پیگیری","en":"Follow-up visit"}',
   '{"fa":"حدود ۲ هفته بعد نوبت بگیر","en":"Book one for about 2 weeks later"}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_moods', 'sad', 1, 1, NULL,
   '{"fa":"غمگین","en":"Sad"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_moods', 'numb', 2, 1, NULL,
   '{"fa":"بی‌حس","en":"Numb"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_moods', 'angry', 3, 1, NULL,
   '{"fa":"عصبانی","en":"Angry"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_moods', 'a_bit_better', 4, 1, NULL,
   '{"fa":"کمی بهتر","en":"A bit better"}',
   NULL,
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_support', 'mood_support', 1, 1, NULL,
   '{"fa":"حال دلت","en":"How you feel"}',
   '{"fa":"غم، احساس گناه یا بی‌حسی بعد از سقط طبیعی است و تقصیر تو نیست.","en":"Sadness, guilt or numbness after a pregnancy loss is natural, and it is not your fault."}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_support', 'crisis', 2, 1, NULL,
   '{"fa":"اگر غم خیلی سنگین است","en":"If the sadness feels too heavy"}',
   '{"fa":"اگر غم خیلی سنگین است یا به آسیب زدن به خودت فکر می‌کنی، همین حالا با صدای مشاور ۱۴۸۰ یا اورژانس اجتماعی ۱۲۳ تماس بگیر.","en":"If the sadness feels too heavy or you are thinking about hurting yourself, call the 1480 counselling line or the 123 social emergency line right now."}',
   '{"severity":"urgent","hotlines":[{"number":"1480","label":{"fa":"صدای مشاور","en":"Counselling line"}},{"number":"123","label":{"fa":"اورژانس اجتماعی","en":"Social emergency"}}]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_support', 'recurrent_hint', 3, 1, NULL,
   '{"fa":"اگر سقط دوم یا سوم بود","en":"If this was a second or third loss"}',
   '{"fa":"اگر سقط دوم یا سوم بود، درباره آزمایش‌های بررسی علت با پزشکت صحبت کن.","en":"If this was a second or third loss, talk with your doctor about tests to look for a cause."}',
   NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_support', 'companion_notice', 4, 1, NULL,
   '{"fa":"{name} خبر داد که بارداری ادامه ندارد.","en":"{name} let you know that the pregnancy is not continuing."}',
   NULL,
   '{"someone":{"fa":"همراهت","en":"Your partner"}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_next_steps', 'cycle', 1, 1, NULL,
   '{"fa":"فعلاً فقط پیگیری سیکل","en":"Just track my cycle for now"}',
   '{"fa":"اولین پریود معمولاً ۴ تا ۶ هفته بعد می‌آید","en":"The first period usually comes 4 to 6 weeks later"}',
   '{"life_mode":"cycle"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_next_steps', 'ttc', 2, 1, NULL,
   '{"fa":"دوباره اقدام به بارداری","en":"Try to conceive again"}',
   '{"fa":"زمان مناسب را با پزشکت هماهنگ کن","en":"Agree on the right time with your doctor"}',
   '{"life_mode":"ttc"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('loss_next_steps', 'nothing', 3, 1, NULL,
   '{"fa":"فعلاً هیچ‌چیز","en":"Nothing for now"}',
   '{"fa":"فقط یادآور پیگیری‌های پزشکی می‌ماند","en":"Only the medical follow-up reminders stay"}',
   '{"life_mode":"cycle"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'loss_types' AND `code` IN ('early_miscarriage', 'late_miscarriage', 'ectopic', 'chemical', 'unspecified');
DELETE FROM `catalog_items` WHERE `group` = 'loss_warning_signs' AND `code` IN ('heavy_bleeding', 'fever', 'severe_pain', 'dizziness_fainting', 'foul_discharge');
DELETE FROM `catalog_items` WHERE `group` = 'loss_hotlines' AND `code` IN ('emergency', 'counselling', 'social_emergency');
DELETE FROM `catalog_items` WHERE `group` = 'loss_followups' AND `code` IN ('bleeding', 'beta', 'visit');
DELETE FROM `catalog_items` WHERE `group` = 'loss_moods' AND `code` IN ('sad', 'numb', 'angry', 'a_bit_better');
DELETE FROM `catalog_items` WHERE `group` = 'loss_support' AND `code` IN ('mood_support', 'crisis', 'recurrent_hint', 'companion_notice');
DELETE FROM `catalog_items` WHERE `group` = 'loss_next_steps' AND `code` IN ('cycle', 'ttc', 'nothing');
DROP TABLE IF EXISTS `pregnancy_loss_moods`;
DROP TABLE IF EXISTS `pregnancy_losses`;
