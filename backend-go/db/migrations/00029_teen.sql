-- 00029_teen.sql — teen mode (CB-TEEN-01, roadmap/E09-teen, boards nbl_Teen_Onb / nbl_Teen_Home / nbl_Teen_Parent): the
-- teen's onboarding answers, her school emergency-kit checklist and the admin-editable teen content. Minors' health
-- data: user-scoped (FK cascade on account deletion), no analytics, nothing shared unless the teen grants it through
-- bloom's companion system (type `parent`, teen-only sections — validated in Go, internal/companion; no schema change
-- there). The life-stage mode stays `user_life_profiles.life_mode = teen` (bloom B-N2-01/03).
--
-- Twin of backend/database/migrations/2026_10_02_000029_create_teen_tables.php (Laravel owns the prod schema until
-- T-M2-27; it creates the same tables and insertOrIgnore's the same catalog rows), so `make schema-diff` stays green.
-- Spelled the way mariadb-dump prints the Laravel-built tables. IF NOT EXISTS: a database whose Laravel half already
-- created the tables (prod at cutover) must not fail here. 00025, 00027 and 00028 belong to parallel tasks.
--
-- teen_profiles    one per user (Teen_Onb): age_band 10_12|13_15|16_17, menarche not_yet|under_1y|over_1y, and
--                  parent_note — the short note the teen writes for her parent («علائم و یادداشت‌ها»), shown on the
--                  parent's card only while the teen grants `teen_notes`.
-- teen_kit_checks  one row per (user, checked kit item): item_code = a `teen_kit_items` catalog code. No row = not
--                  checked; unchecking deletes the row.
--
-- Catalog seed ([needs clinical review]: needs_review = 1, audience teen; age-appropriate, no fertility or sexual-health
-- copy). meta of teen_signs: {kind: sign|estimate|talk, menarche: [answers it is for] (absent = all), age_bands: [...]
-- (absent = all), severity?: caution}; the API picks the first active `estimate` matching the teen's answers as the
-- readiness text. teen_faq and teen_kit_items carry no meta.

-- +goose Up
CREATE TABLE IF NOT EXISTS `teen_profiles` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `age_band` varchar(8) NOT NULL,
  `menarche` varchar(16) NOT NULL,
  `parent_note` varchar(280) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `teen_profiles_user_id_unique` (`user_id`),
  CONSTRAINT `teen_profiles_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `teen_kit_checks` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `item_code` varchar(64) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `teen_kit_checks_user_id_item_code_unique` (`user_id`,`item_code`),
  CONSTRAINT `teen_kit_checks_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('teen_signs', 'approaching_signs', 1, 1, '["teen"]',
   '{"fa":"نشانه‌های نزدیک شدن اولین پریود","en":"Signs your first period is getting close"}',
   '{"fa":"معمولاً حدود ۲ سال بعد از شروع رشد سینه‌ها. ترشح سفید یا شفاف از چند ماه قبل هم طبیعی است.","en":"It usually comes about 2 years after your breasts start to develop. White or clear discharge for a few months before is normal too."}',
   '{"kind":"sign","menarche":["not_yet"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_signs', 'growth_spurt', 2, 1, '["teen"]',
   '{"fa":"جهش قد","en":"A growth spurt"}',
   '{"fa":"رشد ناگهانی قد و رویش مو زیر بغل معمولاً قبل از اولین پریود شروع می‌شود.","en":"A sudden growth spurt and underarm hair usually start before the first period."}',
   '{"kind":"sign","menarche":["not_yet"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_signs', 'estimate_year_or_two', 3, 1, '["teen"]',
   '{"fa":"بر اساس جواب‌هایت: احتمالاً در یکی دو سال آینده","en":"Based on your answers: probably within the next year or two"}',
   '{"fa":"هر بدنی زمان خودش را دارد. وقتی کیف اضطراری آماده باشد، غافلگیر نمی‌شوی.","en":"Every body has its own timing. With your emergency kit ready, you won''t be caught off guard."}',
   '{"kind":"estimate","menarche":["not_yet"],"age_bands":["10_12"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_signs', 'estimate_coming_months', 4, 1, '["teen"]',
   '{"fa":"بر اساس جواب‌هایت: احتمالاً در ماه‌های آینده","en":"Based on your answers: probably in the coming months"}',
   '{"fa":"هر بدنی زمان خودش را دارد. وقتی کیف اضطراری آماده باشد، غافلگیر نمی‌شوی.","en":"Every body has its own timing. With your emergency kit ready, you won''t be caught off guard."}',
   '{"kind":"estimate","menarche":["not_yet"],"age_bands":["13_15"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_signs', 'estimate_talk', 5, 1, '["teen"]',
   '{"fa":"بهتر است با مادرت یا پزشک صحبت کنی","en":"It''s a good idea to talk to your mother or a doctor"}',
   '{"fa":"بیشتر دخترها تا ۱۵ سالگی پریود می‌شوند. صحبت با پزشک کمک می‌کند مطمئن شوی همه چیز روبه‌راه است.","en":"Most girls get their first period by 15. A doctor can help make sure everything is okay."}',
   '{"kind":"estimate","menarche":["not_yet"],"age_bands":["16_17"],"severity":"caution"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_signs', 'estimate_first_year', 6, 1, '["teen"]',
   '{"fa":"سال اول: نامنظم بودن طبیعی است","en":"The first year: irregular is normal"}',
   '{"fa":"در ۱ تا ۲ سال اول فاصله پریودها ممکن است خیلی فرق کند. ثبت کردن کمک می‌کند الگوی خودت را ببینی.","en":"In the first 1–2 years the gap between periods can vary a lot. Logging helps you see your own pattern."}',
   '{"kind":"estimate","menarche":["under_1y"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_signs', 'estimate_settling', 7, 1, '["teen"]',
   '{"fa":"چرخه‌ات در حال منظم شدن است","en":"Your cycle is settling"}',
   '{"fa":"ثبت کردن کمک می‌کند زمان پریود بعدی را زودتر بدانی و کیفت آماده باشد.","en":"Logging helps you know when your next period is coming so your kit is ready."}',
   '{"kind":"estimate","menarche":["over_1y"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_signs', 'when_to_talk', 8, 1, '["teen"]',
   '{"fa":"کی با مادرت یا پزشک صحبت کنی","en":"When to talk to your mother or a doctor"}',
   '{"fa":"اگر تا ۱۵ سالگی پریود نشدی، یا درد و خونریزی خیلی زیاد داری، با مادرت یا پزشک صحبت کن.","en":"If you haven''t had a period by 15, or you have a lot of pain or very heavy bleeding, talk to your mother or a doctor."}',
   '{"kind":"talk"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_faq', 'irregular', 1, 1, '["teen"]',
   '{"fa":"پریودم نامنظم است، مشکلی دارد؟","en":"My period is irregular — is something wrong?"}',
   '{"fa":"در ۱ تا ۲ سال اول نامنظم بودن خیلی شایع و طبیعی است.","en":"In the first 1–2 years irregular periods are very common and normal."}',
   NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_faq', 'how_much_bleeding', 2, 1, '["teen"]',
   '{"fa":"چقدر خونریزی طبیعی است؟","en":"How much bleeding is normal?"}',
   '{"fa":"معمولاً ۳ تا ۷ روز. اگر هر ساعت نوار پر می‌شود، به مادرت یا پزشک بگو.","en":"Usually 3 to 7 days. If you soak a pad every hour, tell your mother or a doctor."}',
   NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_faq', 'period_pain', 3, 1, '["teen"]',
   '{"fa":"درد پریود چه کنم؟","en":"What can I do about period pain?"}',
   '{"fa":"گرما روی شکم، کمی تحرک و مسکن ساده با اجازه بزرگ‌ترها کمک می‌کند.","en":"Warmth on your tummy, some gentle movement and a simple painkiller with an adult''s OK can help."}',
   NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_faq', 'pad_change', 4, 1, '["teen"]',
   '{"fa":"هر چند وقت نوار را عوض کنم؟","en":"How often should I change my pad?"}',
   '{"fa":"معمولاً هر ۴ تا ۶ ساعت، یا زودتر اگر پر شده است.","en":"Usually every 4 to 6 hours, or sooner if it is full."}',
   NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_faq', 'sports', 5, 1, '["teen"]',
   '{"fa":"در پریود می‌توانم ورزش کنم؟","en":"Can I do sports during my period?"}',
   '{"fa":"بله. ورزش سبک حتی ممکن است درد را کمتر کند.","en":"Yes. Gentle exercise may even ease cramps."}',
   NULL, 1, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_kit_items', 'pads', 1, 1, '["teen"]',
   '{"fa":"۲ نوار بهداشتی","en":"2 sanitary pads"}', NULL, NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_kit_items', 'underwear', 2, 1, '["teen"]',
   '{"fa":"یک لباس زیر اضافه","en":"A spare pair of underwear"}', NULL, NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_kit_items', 'wipes', 3, 1, '["teen"]',
   '{"fa":"دستمال مرطوب","en":"Wet wipes"}', NULL, NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('teen_kit_items', 'pouch', 4, 1, '["teen"]',
   '{"fa":"کیسه کوچک","en":"A small pouch"}', NULL, NULL, 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'teen_signs'
  AND `code` IN ('approaching_signs', 'growth_spurt', 'estimate_year_or_two', 'estimate_coming_months', 'estimate_talk',
                 'estimate_first_year', 'estimate_settling', 'when_to_talk');
DELETE FROM `catalog_items` WHERE `group` = 'teen_faq'
  AND `code` IN ('irregular', 'how_much_bleeding', 'period_pain', 'pad_change', 'sports');
DELETE FROM `catalog_items` WHERE `group` = 'teen_kit_items' AND `code` IN ('pads', 'underwear', 'wipes', 'pouch');
DROP TABLE IF EXISTS `teen_kit_checks`;
DROP TABLE IF EXISTS `teen_profiles`;
