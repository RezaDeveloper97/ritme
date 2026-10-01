-- 00017_contraception.sql — contraception tracking (CB-CONTRA-01, roadmap/E06-contra): the user's current method,
-- daily pill logs and the links to the care reminders the method creates. Health data: user-scoped (FK cascade on
-- account deletion), no analytics. One switch and one pill reminder (docs/canvas-build/README.md C5): the «track
-- contraception» flag stays `user_life_profiles.track_contraception` (B-N2-01) and the daily pill reminder stays the
-- `pill` category + time of `notification_preferences` (B-N1-09) — nothing here duplicates them. Missed-pill guidance
-- is admin-editable catalog content (group `missed_pill_rules`, seeded below, needs_review = 1).
--
-- Twin of backend/database/migrations/2026_10_01_000008_create_contraception_tables.php (Laravel owns the prod schema
-- until T-M2-27; it creates the same tables and insertOrIgnore's the same catalog rows), so `make schema-diff` stays
-- green. Spelled the way mariadb-dump prints the Laravel-built tables. IF NOT EXISTS: a database whose Laravel half
-- already created the tables (prod at cutover) must not fail here.
--
-- contraception_methods    one per user: method combined_pill|progestin_pill|copper_iud|hormonal_iud|injection|
--                          implant|condom|other; pill pack (pack_type 21_7|28|24_4, pack_started_on = first day of a
--                          pack, packs repeat every 28 days; packs_left counted on packs_counted_on = the start of
--                          the pack in use then); IUD inserted_on + lifetime + 6-week visit done; last injection;
--                          implant replace_on
-- contraception_pill_logs  one row per (user, day): taken | missed
-- contraception_reminders  one row per (user, kind): the `reminders` row (care reminders, type appointment|custom)
--                          the method created — iud_string_check, iud_followup, iud_replacement, injection_next,
--                          implant_replacement, pill_refill. A reminder the user deletes takes its link with it.

-- +goose Up
CREATE TABLE IF NOT EXISTS `contraception_methods` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `method` varchar(32) NOT NULL,
  `pack_type` varchar(8) DEFAULT NULL,
  `pack_started_on` date DEFAULT NULL,
  `packs_left` tinyint(3) unsigned DEFAULT NULL,
  `packs_counted_on` date DEFAULT NULL,
  `inserted_on` date DEFAULT NULL,
  `iud_lifetime_years` tinyint(3) unsigned DEFAULT NULL,
  `followup_done` tinyint(1) NOT NULL DEFAULT 0,
  `injected_on` date DEFAULT NULL,
  `replace_on` date DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `contraception_methods_user_id_unique` (`user_id`),
  CONSTRAINT `contraception_methods_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `contraception_pill_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `log_date` date NOT NULL,
  `status` varchar(8) NOT NULL,
  `logged_at` datetime NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `contraception_pill_logs_user_id_log_date_unique` (`user_id`,`log_date`),
  CONSTRAINT `contraception_pill_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `contraception_reminders` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(32) NOT NULL,
  `reminder_id` bigint(20) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `contraception_reminders_user_id_kind_unique` (`user_id`,`kind`),
  KEY `contraception_reminders_reminder_id_foreign` (`reminder_id`),
  CONSTRAINT `contraception_reminders_reminder_id_foreign` FOREIGN KEY (`reminder_id`) REFERENCES `reminders` (`id`) ON DELETE CASCADE,
  CONSTRAINT `contraception_reminders_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Catalog seed ([needs clinical review]: needs_review = 1). meta of missed_pill_rules (admin hint, hints.ts):
-- {methods: [method codes the rule is for], missed: 1 = one pill (< 48 h late) | 2 = two or more,
--  severity: info|caution|urgent, steps: [{fa, en}, …]} — the client shows the rule matching the user's method and
-- the count she picks; `week1_unprotected` / `progestin_note` carry no `missed` and show with every count.
INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('missed_pill_rules', 'combined_one', 1, 1, NULL,
   '{"fa":"۱ قرص (کمتر از ۴۸ ساعت دیر)","en":"1 pill (less than 48 hours late)"}',
   '{"fa":"این راهنمای عمومی قرص ترکیبی است. در صورت شک با پزشک یا داروساز صحبت کن.","en":"This is general guidance for the combined pill. If in doubt, talk to a doctor or pharmacist."}',
   '{"methods":["combined_pill"],"missed":1,"severity":"caution","steps":[{"fa":"قرص جاافتاده را همین حالا بخور، حتی اگر یعنی امروز دو قرص بخوری.","en":"Take the missed pill now, even if it means taking two pills today."},{"fa":"بقیه قرص‌ها را طبق معمول ادامه بده.","en":"Carry on with the rest of the pack as usual."}]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('missed_pill_rules', 'combined_two_plus', 2, 1, NULL,
   '{"fa":"۲ قرص یا بیشتر","en":"2 or more pills"}',
   '{"fa":"این راهنمای عمومی قرص ترکیبی است. در صورت شک با پزشک یا داروساز صحبت کن.","en":"This is general guidance for the combined pill. If in doubt, talk to a doctor or pharmacist."}',
   '{"methods":["combined_pill"],"missed":2,"severity":"caution","steps":[{"fa":"آخرین قرص جاافتاده را همین حالا بخور، حتی اگر یعنی امروز دو قرص بخوری.","en":"Take the last missed pill now, even if it means taking two pills today."},{"fa":"بقیه قرص‌ها را طبق معمول ادامه بده.","en":"Carry on with the rest of the pack as usual."},{"fa":"تا ۷ روز پشت سر هم قرص نخورده‌ای، از کاندوم استفاده کن.","en":"Use condoms until you have taken 7 pills in a row."},{"fa":"اگر در ۷ روز آخر قرص‌های فعال هستی، بعد از تمام شدنشان روزهای استراحت را حذف کن و بسته بعد را مستقیم شروع کن.","en":"If you are in the last 7 active pills, skip the break after them and start the next pack straight away."}]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('missed_pill_rules', 'week1_unprotected', 3, 1, NULL,
   '{"fa":"اگر در هفته اول بسته رابطه محافظت‌نشده داشتی","en":"If you had unprotected sex in the first week of the pack"}',
   '{"fa":"ممکن است به پیشگیری اضطراری نیاز داشته باشی. هر چه زودتر با پزشک یا داروساز مشورت کن.","en":"You may need emergency contraception. Talk to a doctor or pharmacist as soon as possible."}',
   '{"methods":["combined_pill"],"severity":"urgent","pack_week":1}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('missed_pill_rules', 'progestin_note', 4, 1, NULL,
   '{"fa":"قرص تک‌هورمونی","en":"Progestogen-only pill"}',
   '{"fa":"برای قرص تک‌هورمونی قواعد فرق دارد و به نوع قرص بستگی دارد؛ در صورت شک با پزشک یا داروساز صحبت کن.","en":"The rules differ for the progestogen-only pill and depend on the type of pill; if in doubt, talk to a doctor or pharmacist."}',
   '{"methods":["progestin_pill"],"severity":"caution","steps":[{"fa":"قرص جاافتاده را به محض یادآوری بخور و قرص بعدی را سر ساعت همیشگی بخور.","en":"Take the missed pill as soon as you remember and the next one at the usual time."},{"fa":"اگر بیشتر از زمان مجاز نوع قرصت دیر کردی، تا ۲ روز از کاندوم استفاده کن.","en":"If you are later than your pill type allows, use condoms for the next 2 days."}]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'missed_pill_rules'
  AND `code` IN ('combined_one', 'combined_two_plus', 'week1_unprotected', 'progestin_note');
DROP TABLE IF EXISTS `contraception_reminders`;
DROP TABLE IF EXISTS `contraception_pill_logs`;
DROP TABLE IF EXISTS `contraception_methods`;
