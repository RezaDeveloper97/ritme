-- 00035_vitals.sql — vitals «علائم حیاتی» (bloom B-N6-01; artboards nbl_Vitals_* in c-health-record): timed blood
-- pressure, blood glucose and heart-rate readings, the weekly measurement plan, and the copy of the two urgent safety
-- messages. Domain logic: internal/vitals (classification ACC/AHA 2017 + ADA, unit conversion, reports, plan,
-- reminders through notifications.Decide). Bloom owns 00034–00039 (00034 = labs, B-N6-06).
--
-- vital_readings      one reading of a user. type bp | glucose | hr; measured_at = Tehran wall-clock.
--                     bp: systolic / diastolic (mmHg), optional pulse (bpm), arm left|right, position
--                     sitting|standing|lying. glucose: glucose_mg_dl (canonical, 1 decimal; mmol/L × 18 on input),
--                     glucose_unit = the unit the user typed (mg_dl|mmol_l, echoed back), context
--                     fasting|before_meal|after_meal|bedtime|random, method glucometer|lab|sensor. hr: pulse (bpm),
--                     context resting|after_exercise|after_waking|stress. Classification is computed on read from the
--                     thresholds in internal/vitals/thresholds.go (never stored, so a threshold change re-classifies
--                     history). Codes are varchar validated in Go.
-- vital_plan_items    the weekly plan «برنامه اندازه‌گیری این هفته»: one row per (user, type, slot); slot
--                     morning|evening for bp / hr, fasting|before_meal|after_meal|bedtime for glucose. days = weekday
--                     bitmask (bit 0 = Saturday … bit 6 = Friday); remind_at HH:MM (NULL = no reminder; the push goes
--                     through notifications.Decide under the `vitals` category).
--
-- message_contents vitals_alert / bp_crisis and vitals_alert / glucose_low (fa, en): the copy of the urgent modal
-- (title, what we saw, advice, call / ack actions with the emergency number) returned when a saved reading crosses the
-- safety thresholds (BP > 180 and/or > 120, glucose < 54 mg/dL). The thresholds stay in code. [needs clinical review]
--
-- Health data: every row is scoped to its user and never logged. Twin of
-- backend/database/migrations/2026_10_06_000035_create_vitals_tables.php (Laravel owns the prod schema until
-- T-M2-27), so `make schema-diff` stays green. Spelled the way mariadb-dump prints the Laravel-built tables; IF NOT
-- EXISTS / INSERT IGNORE so a database whose Laravel half already ran does not fail.

-- +goose Up
CREATE TABLE IF NOT EXISTS `vital_readings` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `type` varchar(8) NOT NULL,
  `measured_at` datetime NOT NULL,
  `systolic` smallint(5) unsigned DEFAULT NULL,
  `diastolic` smallint(5) unsigned DEFAULT NULL,
  `pulse` smallint(5) unsigned DEFAULT NULL,
  `arm` varchar(8) DEFAULT NULL,
  `position` varchar(12) DEFAULT NULL,
  `glucose_mg_dl` decimal(5,1) DEFAULT NULL,
  `glucose_unit` varchar(8) DEFAULT NULL,
  `context` varchar(16) DEFAULT NULL,
  `method` varchar(12) DEFAULT NULL,
  `note` varchar(500) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `vital_readings_user_id_type_measured_at_index` (`user_id`,`type`,`measured_at`),
  KEY `vital_readings_user_id_measured_at_index` (`user_id`,`measured_at`),
  CONSTRAINT `vital_readings_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `vital_plan_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `type` varchar(8) NOT NULL,
  `slot` varchar(16) NOT NULL,
  `days` tinyint(3) unsigned NOT NULL DEFAULT 127,
  `remind_at` varchar(5) DEFAULT NULL,
  `sort_order` tinyint(3) unsigned NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `vital_plan_items_user_id_type_slot_unique` (`user_id`,`type`,`slot`),
  CONSTRAINT `vital_plan_items_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `message_contents` (`group`, `item_key`, `locale`, `label`, `payload`, `is_active`, `is_approved`, `sort_order`, `created_at`, `updated_at`) VALUES
  ('vitals_alert', 'bp_crisis', 'fa', 'vitals_alert / bp_crisis', '{"level":"urgent","title":"فشار خونت در محدودهٔ بحرانی است","what_we_saw":"فشار {value} ثبت کردی؛ عدد بالاتر از ۱۸۰/۱۲۰ اورژانسی حساب می‌شود.","advice":"۵ دقیقه آرام بنشین و دوباره اندازه بگیر. اگر باز هم بالاتر از ۱۸۰/۱۲۰ بود، یا سردرد شدید، درد قفسهٔ سینه، تنگی نفس، تاری دید، ضعف یا بی‌حسی یک طرف بدن یا اشکال در حرف زدن داری، منتظر نمان.","actions":[{"key":"call","label":"تماس با اورژانس ۱۱۵","phone":"115"},{"key":"ack","label":"دوباره اندازه می‌گیرم"}],"contact":"ریتمی تشخیص پزشکی نمی‌دهد؛ در شرایط اورژانسی همین حالا با ۱۱۵ تماس بگیر."}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('vitals_alert', 'bp_crisis', 'en', 'vitals_alert / bp_crisis', '{"level":"urgent","title":"Your blood pressure is in the crisis range","what_we_saw":"You logged {value}; a reading above 180/120 counts as an emergency.","advice":"Sit quietly for 5 minutes and measure again. If it is still above 180/120, or you have a severe headache, chest pain, shortness of breath, blurred vision, weakness or numbness on one side or trouble speaking, do not wait.","actions":[{"key":"call","label":"Call emergency services (115)","phone":"115"},{"key":"ack","label":"I will measure again"}],"contact":"Ritme does not give a medical diagnosis; in an emergency call 115 now."}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('vitals_alert', 'glucose_low', 'fa', 'vitals_alert / glucose_low', '{"level":"urgent","title":"قند خونت خیلی پایین است","what_we_saw":"قند {value} ثبت کردی؛ کمتر از ۵۴ mg/dL (۳٫۰ mmol/L) افت قند شدید حساب می‌شود.","advice":"همین حالا ۱۵ گرم قند زودجذب بخور (مثلاً نصف لیوان آب‌میوه یا ۳ تا ۴ حبه قند) و ۱۵ دقیقه بعد دوباره اندازه بگیر. اگر گیج یا خواب‌آلودی، نمی‌توانی چیزی بخوری یا قند بالا نمی‌رود، از اطرافیان کمک بخواه.","actions":[{"key":"call","label":"تماس با اورژانس ۱۱۵","phone":"115"},{"key":"ack","label":"قند خوردم، دوباره اندازه می‌گیرم"}],"contact":"ریتمی تشخیص پزشکی نمی‌دهد؛ در شرایط اورژانسی همین حالا با ۱۱۵ تماس بگیر."}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('vitals_alert', 'glucose_low', 'en', 'vitals_alert / glucose_low', '{"level":"urgent","title":"Your blood sugar is very low","what_we_saw":"You logged {value}; below 54 mg/dL (3.0 mmol/L) counts as severe low blood sugar.","advice":"Take 15 g of fast-acting sugar now (for example half a glass of juice or 3 to 4 sugar cubes) and measure again after 15 minutes. If you feel confused or drowsy, cannot eat or it does not go up, ask someone near you for help.","actions":[{"key":"call","label":"Call emergency services (115)","phone":"115"},{"key":"ack","label":"I had sugar, measuring again"}],"contact":"Ritme does not give a medical diagnosis; in an emergency call 115 now."}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
-- Removes the seeded copy rows only while nobody edited them (label untouched, updated_at = created_at), like 00033.
DELETE FROM `message_contents`
 WHERE `group` = 'vitals_alert' AND `item_key` IN ('bp_crisis', 'glucose_low')
   AND `locale` IN ('fa', 'en')
   AND `label` = CONCAT(`group`, ' / ', `item_key`)
   AND `updated_at` <=> `created_at`;
DROP TABLE IF EXISTS `vital_plan_items`;
DROP TABLE IF EXISTS `vital_readings`;
