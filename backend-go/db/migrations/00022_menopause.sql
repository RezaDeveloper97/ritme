-- 00022_menopause.sql — menopause mode schema and content seeds (CB-MENO-01, roadmap/E02-meno, docs/canvas-build/menopause.md).
-- 00021 is reserved by bloom (intentionally not used here).
--
-- Built on bloom, nothing duplicated:
--   * the profile (stage peri|meno|post|unsure, approximate last period, surgical, HRT) stays in bloom's
--     `user_life_profiles.menopause_*` columns (B-N2-01, asked by Onb_Meno) — no menopause profile table;
--   * the daily symptoms, the bleeding choice and the triggers are log taxonomy v2 slots stored in
--     `health_log_entries` (B-N3-01; internal/healthlog/taxonomy menopause.go) — no day-log table;
--   * all clinical/list copy is admin-editable `catalog_items` (CB-CORE-03), fa + en, needs_review = 1.
--
-- New user-scoped health tables (FK cascade on account deletion, no analytics):
--   hot_flashes          one row per hot flash: started_at (Tehran wall clock), duration_s (NULL = timer still
--                        running), severity 1–4 (mild|moderate|severe|very severe, NULL until set), night, sweat,
--                        triggers = JSON list of trigger codes (menopause.triggers options + `unknown`)
--   menopause_scores     one per (user, month): month = first day of the Jalali month the questionnaire is for;
--                        answers = JSON {meno_score_items code: 0–4} (11 items), total /44 and the domain
--                        subtotals somatic /16, psychological /16, urogenital /12 (stored, computed by the API)
--   treatment_items      the user's HRT / supplement / lifestyle items: kind hrt|supplement|lifestyle, name and
--                        dose as typed, schedule morning|noon|evening|night|weekly, started_on, review_on (doctor
--                        review), weekly_goal + goal_unit sessions|minutes (lifestyle), stopped_on (NULL =
--                        active), reminder_id = the care `reminders` row the item created (NULL on delete)
--   treatment_intakes    one row per (item, day): taken, with amount = minutes/sessions for lifestyle items
--   side_effect_logs     one row per (user, day, code): breast_tenderness|spotting|headache|bloating|mood_change
--                        (codes are UI labels), optionally tied to the treatment item
--
-- checkup_types.audiences (JSON list of life modes, NULL = everyone) scopes M4 catalog rows to a mode. The nine
-- menopause checkups are seeded with audiences ["menopause"] and is_active = 0: the M4 engine does not read
-- audiences yet, so active rows would show to every user. They are activated once the checkups API filters by
-- audience (follow-up, see docs/canvas-build/menopause.md §6).
--
-- Twin of backend/database/migrations/2026_10_01_000022_create_menopause_tables.php (Laravel owns the prod schema
-- until T-M2-27; same tables, same column, same insertOrIgnore'd rows), so `make schema-diff` stays green. Spelled
-- the way mariadb-dump prints the Laravel-built tables. IF NOT EXISTS / INSERT IGNORE: a database whose Laravel half
-- already ran must not fail or get duplicates, and seeding never overwrites an admin edit. Seed rows are generated
-- from one source for both migrations.

-- +goose Up
ALTER TABLE `checkup_types` ADD COLUMN IF NOT EXISTS `audiences` json DEFAULT NULL AFTER `hide_in_pregnancy`;

CREATE TABLE IF NOT EXISTS `hot_flashes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `started_at` datetime NOT NULL,
  `duration_s` int(10) unsigned DEFAULT NULL,
  `severity` tinyint(3) unsigned DEFAULT NULL,
  `night` tinyint(1) NOT NULL DEFAULT 0,
  `sweat` tinyint(1) NOT NULL DEFAULT 0,
  `triggers` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `hot_flashes_user_id_started_at_index` (`user_id`,`started_at`),
  CONSTRAINT `hot_flashes_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `menopause_scores` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `month` date NOT NULL,
  `answers` json NOT NULL,
  `total` tinyint(3) unsigned NOT NULL,
  `somatic` tinyint(3) unsigned NOT NULL,
  `psychological` tinyint(3) unsigned NOT NULL,
  `urogenital` tinyint(3) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `menopause_scores_user_id_month_unique` (`user_id`,`month`),
  CONSTRAINT `menopause_scores_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `treatment_items` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `kind` varchar(16) NOT NULL,
  `name` varchar(120) NOT NULL,
  `dose` varchar(120) DEFAULT NULL,
  `schedule` varchar(16) DEFAULT NULL,
  `started_on` date DEFAULT NULL,
  `review_on` date DEFAULT NULL,
  `weekly_goal` smallint(5) unsigned DEFAULT NULL,
  `goal_unit` varchar(16) DEFAULT NULL,
  `stopped_on` date DEFAULT NULL,
  `reminder_id` bigint(20) unsigned DEFAULT NULL,
  `sort_order` int(11) NOT NULL DEFAULT 0,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `treatment_items_reminder_id_foreign` (`reminder_id`),
  KEY `treatment_items_user_id_kind_index` (`user_id`,`kind`),
  CONSTRAINT `treatment_items_reminder_id_foreign` FOREIGN KEY (`reminder_id`) REFERENCES `reminders` (`id`) ON DELETE SET NULL,
  CONSTRAINT `treatment_items_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `treatment_intakes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `treatment_item_id` bigint(20) unsigned NOT NULL,
  `intake_date` date NOT NULL,
  `amount` smallint(5) unsigned DEFAULT NULL,
  `taken_at` datetime NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `treatment_intakes_treatment_item_id_intake_date_unique` (`treatment_item_id`,`intake_date`),
  KEY `treatment_intakes_user_id_intake_date_index` (`user_id`,`intake_date`),
  CONSTRAINT `treatment_intakes_treatment_item_id_foreign` FOREIGN KEY (`treatment_item_id`) REFERENCES `treatment_items` (`id`) ON DELETE CASCADE,
  CONSTRAINT `treatment_intakes_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `side_effect_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `treatment_item_id` bigint(20) unsigned DEFAULT NULL,
  `log_date` date NOT NULL,
  `code` varchar(32) NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `side_effect_logs_user_id_log_date_code_unique` (`user_id`,`log_date`,`code`),
  KEY `side_effect_logs_treatment_item_id_foreign` (`treatment_item_id`),
  CONSTRAINT `side_effect_logs_treatment_item_id_foreign` FOREIGN KEY (`treatment_item_id`) REFERENCES `treatment_items` (`id`) ON DELETE SET NULL,
  CONSTRAINT `side_effect_logs_user_id_foreign` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Catalog seed ([needs clinical review]: needs_review = 1, audiences ["menopause"]); meta shapes per group in
-- docs/canvas-build/menopause.md §4: meno_score_items {domain somatic|psychological|urogenital, max 4, log: [taxonomy
-- slots]}, meno_score_bands {min, max}, meno_alerts {severity info|caution|urgent, primary?, emergency?, hotline?,
-- hrt?, stages?, cta?}, meno_tips {placement home|log|hot_flash|treatment|treatment_lifestyle|checkups|score,
-- stages?, weekly_goal?, goal_unit?, review_after_months?}, meno_checkup_groups {checkups: [checkup_types keys]}.
INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('meno_score_items', 'hot_flashes', 1, 1, '["menopause"]',
   '{"fa":"گرگرفتگی و تعریق","en":"Hot flashes and sweating"}',
   '{"fa":"موج‌های گرما و تعریق، در روز یا شب","en":"Waves of heat and sweating, by day or night"}',
   '{"domain":"somatic","max":4,"log":["symptoms.general.hot_flashes","symptoms.general.night_sweats"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'heart_discomfort', 2, 1, '["menopause"]',
   '{"fa":"ناراحتی قلبی","en":"Heart discomfort"}',
   '{"fa":"تپش قلب، تند یا نامنظم زدن قلب، احساس فشار در سینه","en":"Palpitations, a racing or skipping heartbeat, chest tightness"}',
   '{"domain":"somatic","max":4,"log":["symptoms.general.palpitations"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'sleep_problems', 3, 1, '["menopause"]',
   '{"fa":"مشکلات خواب","en":"Sleep problems"}',
   '{"fa":"سخت خوابیدن، بیدار شدن شبانه یا زود بیدار شدن","en":"Trouble falling asleep, waking at night or waking too early"}',
   '{"domain":"somatic","max":4,"log":["symptoms.general.insomnia"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'joint_muscle', 4, 1, '["menopause"]',
   '{"fa":"درد مفاصل و عضلات","en":"Joint and muscle pain"}',
   '{"fa":"درد یا خشکی مفاصل و عضلات","en":"Aching or stiff joints and muscles"}',
   '{"domain":"somatic","max":4,"log":["pain.location.joints"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'depressive_mood', 5, 1, '["menopause"]',
   '{"fa":"غمگینی یا بی‌حوصلگی","en":"Low mood"}',
   '{"fa":"احساس غم، بی‌حوصلگی، زود گریه کردن یا بی‌انگیزگی","en":"Feeling down, sad, tearful or unmotivated"}',
   '{"domain":"psychological","max":4,"log":["symptoms.general.low_mood"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'irritability', 6, 1, '["menopause"]',
   '{"fa":"تحریک‌پذیری","en":"Irritability"}',
   '{"fa":"عصبی بودن، زود از کوره در رفتن","en":"Feeling tense or quick to anger"}',
   '{"domain":"psychological","max":4,"log":["symptoms.general.irritability"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'anxiety', 7, 1, '["menopause"]',
   '{"fa":"اضطراب","en":"Anxiety"}',
   '{"fa":"بی‌قراری، دلشوره یا احساس ترس","en":"Restlessness, worry or feeling panicky"}',
   '{"domain":"psychological","max":4,"log":["symptoms.general.anxiety"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'exhaustion', 8, 1, '["menopause"]',
   '{"fa":"خستگی جسمی و ذهنی","en":"Physical and mental exhaustion"}',
   '{"fa":"کم شدن انرژی، تمرکز و حافظه","en":"Less energy, focus and memory"}',
   '{"domain":"psychological","max":4,"log":["symptoms.general.fatigue","symptoms.general.brain_fog"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'sexual_problems', 9, 1, '["menopause"]',
   '{"fa":"مشکلات جنسی","en":"Sexual problems"}',
   '{"fa":"تغییر در میل، رابطه یا رضایت جنسی","en":"Changes in sexual desire, activity or satisfaction"}',
   '{"domain":"urogenital","max":4,"log":["urogenital.symptoms.low_libido"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'bladder_problems', 10, 1, '["menopause"]',
   '{"fa":"مشکلات ادراری","en":"Bladder problems"}',
   '{"fa":"تکرر، فوریت یا نشت ادرار","en":"Needing to pass urine often or urgently, or leaking"}',
   '{"domain":"urogenital","max":4,"log":["urogenital.symptoms.bladder_symptoms"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_items', 'vaginal_dryness', 11, 1, '["menopause"]',
   '{"fa":"خشکی واژن","en":"Vaginal dryness"}',
   '{"fa":"احساس خشکی یا سوزش، یا درد هنگام رابطه","en":"Dryness or burning, or pain during sex"}',
   '{"domain":"urogenital","max":4,"log":["urogenital.symptoms.vaginal_dryness"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_bands', 'none', 1, 1, '["menopause"]',
   '{"fa":"بدون علامت","en":"No symptoms"}',
   NULL,
   '{"min":0,"max":4}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_bands', 'mild', 2, 1, '["menopause"]',
   '{"fa":"خفیف","en":"Mild"}',
   NULL,
   '{"min":5,"max":8}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_bands', 'moderate', 3, 1, '["menopause"]',
   '{"fa":"متوسط","en":"Moderate"}',
   NULL,
   '{"min":9,"max":16}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_score_bands', 'severe', 4, 1, '["menopause"]',
   '{"fa":"شدید","en":"Severe"}',
   NULL,
   '{"min":17,"max":44}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_alerts', 'postmenopausal_bleeding', 1, 1, '["menopause"]',
   '{"fa":"خونریزی بعد از یائسگی","en":"Bleeding after menopause"}',
   '{"fa":"هر خونریزی یا لکه‌بینی، حتی کم، بعد از ۱۲ ماه قطع پریود باید بررسی شود. بیشتر وقت‌ها علت ساده‌ای دارد، اما بررسی زودهنگام مهم است.","en":"Any bleeding or spotting, even a little, 12 months or more after your last period should be checked. Usually the cause is simple, but an early check matters."}',
   '{"severity":"urgent","primary":true,"stages":["meno","post"],"cta":{"fa":"این مورد را به پزشک بگو","en":"Tell your doctor about this"}}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_alerts', 'heavy_perimenopause_bleeding', 2, 1, '["menopause"]',
   '{"fa":"خونریزی خیلی زیاد یا طولانی در پیش‌یائسگی","en":"Very heavy or long bleeding in perimenopause"}',
   '{"fa":"بیش از ۷ روز، یا پر شدن نوار در کمتر از ۲ ساعت","en":"More than 7 days, or soaking a pad in under 2 hours"}',
   '{"severity":"caution","stages":["peri","unsure"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_alerts', 'chest_pain_palpitations', 3, 1, '["menopause"]',
   '{"fa":"درد قفسه سینه یا تپش قلب شدید","en":"Chest pain or a severe racing heart"}',
   '{"fa":"اگر ناگهانی و شدید است، با اورژانس (۱۱۵) تماس بگیر","en":"If it is sudden and severe, call emergency services (115)"}',
   '{"severity":"urgent","emergency":true,"hotline":"115"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_alerts', 'one_sided_leg_swelling', 4, 1, '["menopause"]',
   '{"fa":"درد و ورم یک‌طرفه ساق پا","en":"Pain and swelling in one calf"}',
   '{"fa":"به‌خصوص اگر هورمون‌درمانی می‌کنی","en":"Especially if you take hormone therapy"}',
   '{"severity":"urgent","hrt":true}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_alerts', 'breast_change', 5, 1, '["menopause"]',
   '{"fa":"توده یا تغییر در پستان","en":"A lump or change in the breast"}',
   '{"fa":"هر تغییر تازه را به پزشک نشان بده","en":"Have any new change checked by a doctor"}',
   '{"severity":"caution"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_alerts', 'persistent_low_mood', 6, 1, '["menopause"]',
   '{"fa":"غم عمیق یا ناامیدی چند هفته‌ای","en":"Deep sadness or hopelessness for weeks"}',
   '{"fa":"حرف زدن با پزشک یا مشاور کمک می‌کند","en":"Talking to a doctor or counsellor helps"}',
   '{"severity":"caution"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_alerts', 'fragility_fracture', 7, 1, '["menopause"]',
   '{"fa":"شکستگی با ضربه یا زمین خوردن ساده","en":"A fracture from a minor knock or fall"}',
   '{"fa":"ممکن است نشانه پوکی استخوان باشد","en":"It may be a sign of osteoporosis"}',
   '{"severity":"caution"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'stage_peri', 1, 1, '["menopause"]',
   '{"fa":"پیش‌یائسگی","en":"Perimenopause"}',
   '{"fa":"پریودها ممکن است نامنظم، کم یا زیاد شوند و علائم تازه بیایند. ثبت پریود و علائم کمک می‌کند الگوها را ببینی و با پزشک دقیق‌تر حرف بزنی.","en":"Periods may become irregular, lighter or heavier, and new symptoms may start. Logging periods and symptoms helps you see patterns and talk to your doctor in detail."}',
   '{"placement":"home","stages":["peri"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'stage_meno', 2, 1, '["menopause"]',
   '{"fa":"یائسگی","en":"Menopause"}',
   '{"fa":"از ۱۲ ماه گذشته، یعنی وارد یائسگی شده‌ای. از این به بعد هر خونریزی یا لکه‌بینی را ثبت کن و به پزشک خبر بده.","en":"It has been more than 12 months, which means you have reached menopause. From now on, log any bleeding or spotting and tell your doctor."}',
   '{"placement":"home","stages":["meno"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'stage_post', 3, 1, '["menopause"]',
   '{"fa":"پس از یائسگی","en":"Postmenopause"}',
   '{"fa":"در این سال‌ها مراقبت از استخوان و قلب مهم‌تر می‌شود. چکاپ‌های منظم را دنبال کن و هر خونریزی یا لکه‌بینی را به پزشک بگو.","en":"In these years caring for your bones and heart matters more. Keep up with regular checkups and tell your doctor about any bleeding or spotting."}',
   '{"placement":"home","stages":["post"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'stage_unsure', 4, 1, '["menopause"]',
   '{"fa":"مطمئن نیستم","en":"Not sure"}',
   '{"fa":"اگر مطمئن نیستی کجای مسیر هستی، تاریخ تقریبی آخرین پریود کمک می‌کند. ۱۲ ماه بدون پریود یعنی یائسگی؛ پزشک می‌تواند دقیق‌تر بگوید.","en":"If you are not sure where you are, the approximate date of your last period helps. 12 months without a period means menopause; your doctor can tell you more."}',
   '{"placement":"home","stages":["unsure"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'log_bleeding_note', 5, 1, '["menopause"]',
   '{"fa":"خونریزی بعد از یائسگی","en":"Bleeding after menopause"}',
   '{"fa":"بعد از یائسگی هر خونریزی باید به پزشک گفته شود؛ اگر ثبت کنی راهنمایی‌ات می‌کنیم.","en":"After menopause any bleeding should be reported to a doctor; if you log it, we will guide you."}',
   '{"placement":"log","stages":["meno","post"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'hot_flash_breathing', 6, 1, '["menopause"]',
   '{"fa":"در لحظه گرگرفتگی","en":"During a hot flash"}',
   '{"fa":"نفس عمیق و آرام (۶ بار در دقیقه) و خنک کردن مچ و گردن در لحظه کمک می‌کند.","en":"Slow, deep breathing (6 breaths a minute) and cooling your wrists and neck help in the moment."}',
   '{"placement":"hot_flash"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'hrt_review', 7, 1, '["menopause"]',
   '{"fa":"بازبینی با پزشک","en":"Review with your doctor"}',
   '{"fa":"معمولاً ۳ ماه بعد از شروع، اثر و عوارض بررسی می‌شود.","en":"Effects and side effects are usually reviewed about 3 months after starting."}',
   '{"placement":"treatment","review_after_months":3}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'hrt_spotting', 8, 1, '["menopause"]',
   '{"fa":"لکه‌بینی در ماه‌های اول","en":"Spotting in the first months"}',
   '{"fa":"لکه‌بینی در ماه‌های اول شایع است، اما اگر ادامه داشت یا زیاد بود به پزشک بگو.","en":"Spotting is common in the first months, but tell your doctor if it continues or is heavy."}',
   '{"placement":"treatment"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'doctor_only', 9, 1, '["menopause"]',
   '{"fa":"فقط با نظر پزشک","en":"Only with your doctor"}',
   '{"fa":"شروع، قطع یا تغییر دوز هر دارو فقط با نظر پزشک.","en":"Start, stop or change the dose of any medicine only with your doctor."}',
   '{"placement":"treatment"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'lifestyle_resistance', 10, 1, '["menopause"]',
   '{"fa":"ورزش مقاومتی","en":"Resistance exercise"}',
   '{"fa":"برای استخوان و عضله","en":"For bones and muscles"}',
   '{"placement":"treatment_lifestyle","weekly_goal":2,"goal_unit":"sessions"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'lifestyle_brisk_walk', 11, 1, '["menopause"]',
   '{"fa":"پیاده‌روی تند","en":"Brisk walking"}',
   '{"fa":"۱۵۰ دقیقه در هفته","en":"150 minutes a week"}',
   '{"placement":"treatment_lifestyle","weekly_goal":150,"goal_unit":"minutes"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'lifestyle_relaxation', 12, 1, '["menopause"]',
   '{"fa":"تمرین آرام‌سازی","en":"Relaxation practice"}',
   '{"fa":"۱۰ دقیقه، برای گرگرفتگی و خواب","en":"10 minutes, for hot flashes and sleep"}',
   '{"placement":"treatment_lifestyle","weekly_goal":70,"goal_unit":"minutes"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'checkups_intro', 13, 1, '["menopause"]',
   '{"fa":"چکاپ‌ها و آزمایش‌ها","en":"Checkups and tests"}',
   '{"fa":"بعد از یائسگی خطر پوکی استخوان و بیماری قلبی بیشتر می‌شود. این فهرست یادآور است؛ زمان‌بندی دقیق را پزشکت تعیین می‌کند.","en":"After menopause the risk of osteoporosis and heart disease rises. This list is a reminder; your doctor sets the exact timing."}',
   '{"placement":"checkups"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_tips', 'patterns_disclaimer', 14, 1, '["menopause"]',
   '{"fa":"الگوهایی که دیدیم","en":"Patterns we noticed"}',
   '{"fa":"بر اساس ثبت‌های خودت · تشخیص پزشکی نیست","en":"Based on your own logs · not a medical diagnosis"}',
   '{"placement":"score"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_checkup_groups', 'heart_metabolic', 1, 1, '["menopause"]',
   '{"fa":"قلب و متابولیسم","en":"Heart and metabolism"}',
   NULL,
   '{"checkups":["meno_blood_pressure","meno_blood_sugar","meno_lipids","meno_weight_waist"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_checkup_groups', 'bone', 2, 1, '["menopause"]',
   '{"fa":"استخوان","en":"Bones"}',
   NULL,
   '{"checkups":["meno_bone_density","meno_vitamin_d_calcium"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_checkup_groups', 'cancer_screening', 3, 1, '["menopause"]',
   '{"fa":"غربالگری سرطان","en":"Cancer screening"}',
   NULL,
   '{"checkups":["mammography","pap_smear","meno_colon_screening"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_checkup_groups', 'other', 4, 1, '["menopause"]',
   '{"fa":"سایر","en":"Other"}',
   NULL,
   '{"checkups":["meno_thyroid","meno_eye_exam","dentist"]}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

INSERT IGNORE INTO `checkup_types` (`key`, `category`, `title`, `subtitle`, `why`, `performed_by`, `icon`, `tone`, `interval_months`, `interval_months_max`, `age_min`, `remind_lead_days`, `prep_steps`, `hide_in_pregnancy`, `audiences`, `is_active`, `sort_order`, `source_note`, `created_at`, `updated_at`) VALUES
  ('meno_blood_pressure', 'monthly', '{"fa":"فشار خون","en":"Blood pressure"}',
   '{"fa":"ماهانه در خانه، سالانه نزد پزشک","en":"Monthly at home, yearly with your doctor"}',
   '{"fa":"بعد از یائسگی خطر فشار خون بالا بیشتر می‌شود و اغلب بی‌علامت است. اندازه‌گیری منظم در خانه تغییرها را زود نشان می‌دهد.","en":"After menopause the risk of high blood pressure rises and it often has no symptoms. Regular home readings show changes early."}',
   'self', 'heart', 'rose', 1, NULL, NULL, 3,
   '[{"fa":"۵ دقیقه آرام بنشین و بعد اندازه بگیر","en":"Sit quietly for 5 minutes before measuring"}]', 1, '["menopause"]', 0, 101, 'Menopause catalog (CB-MENO-01). [needs clinical review]', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_blood_sugar', 'multi_year', '{"fa":"قند خون ناشتا یا HbA1c","en":"Fasting blood sugar or HbA1c"}',
   '{"fa":"معمولاً هر ۱ تا ۳ سال","en":"Usually every 1 to 3 years"}',
   '{"fa":"با بالا رفتن سن و تغییرات هورمونی، خطر دیابت بیشتر می‌شود. آزمایش منظم آن را زود نشان می‌دهد.","en":"With age and hormonal change the risk of diabetes rises. A regular test shows it early."}',
   'lab', 'blood', 'amber', 12, 36, NULL, 14,
   '[{"fa":"برای قند ناشتا ۸ تا ۱۰ ساعت چیزی جز آب نخور","en":"For fasting sugar, have nothing but water for 8 to 10 hours"}]', 1, '["menopause"]', 0, 102, 'Menopause catalog (CB-MENO-01). [needs clinical review]', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_lipids', 'multi_year', '{"fa":"چربی خون (کلسترول و تری‌گلیسرید)","en":"Blood lipids (cholesterol and triglycerides)"}',
   '{"fa":"معمولاً هر ۱ تا ۵ سال","en":"Usually every 1 to 5 years"}',
   '{"fa":"بعد از یائسگی کلسترول معمولاً بالا می‌رود و خطر بیماری قلبی بیشتر می‌شود.","en":"After menopause cholesterol usually rises and so does the risk of heart disease."}',
   'lab', 'flask', 'amber', 12, 60, NULL, 14,
   '[{"fa":"اگر پزشک گفته، ۹ تا ۱۲ ساعت ناشتا باش","en":"If your doctor asked, fast for 9 to 12 hours"}]', 1, '["menopause"]', 0, 103, 'Menopause catalog (CB-MENO-01). [needs clinical review]', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_weight_waist', 'monthly', '{"fa":"وزن و دور کمر","en":"Weight and waist"}',
   '{"fa":"ماهانه","en":"Monthly"}',
   '{"fa":"در یائسگی چربی بیشتر دور شکم جمع می‌شود که با خطر قلبی همراه است. اندازه‌گیری ماهانه تغییرها را نشان می‌دهد.","en":"In menopause more fat gathers around the waist, which is linked to heart risk. A monthly measurement shows changes."}',
   'self', 'note', 'neutral', 1, NULL, NULL, 3,
   NULL, 1, '["menopause"]', 0, 104, 'Menopause catalog (CB-MENO-01). [needs clinical review]', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_bone_density', 'age_based', '{"fa":"سنجش تراکم استخوان (DEXA)","en":"Bone density scan (DEXA)"}',
   '{"fa":"معمولاً از ۶۵ سالگی، یا زودتر با عامل خطر","en":"Usually from 65, or earlier with a risk factor"}',
   '{"fa":"بعد از یائسگی استخوان‌ها سریع‌تر تحلیل می‌روند. این سنجش پوکی استخوان را پیش از شکستگی نشان می‌دهد. اگر عامل خطر داری، زودتر با پزشک هماهنگ کن.","en":"After menopause bone is lost faster. This scan shows osteoporosis before a fracture. If you have a risk factor, ask your doctor about testing sooner."}',
   'lab', 'shield', 'violet', 24, NULL, 65, 30,
   NULL, 1, '["menopause"]', 0, 105, 'Menopause catalog (CB-MENO-01). [needs clinical review]', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_vitamin_d_calcium', 'annual', '{"fa":"ویتامین D و کلسیم","en":"Vitamin D and calcium"}',
   '{"fa":"طبق نظر پزشک","en":"As your doctor advises"}',
   '{"fa":"ویتامین D و کلسیم کافی برای استخوان مهم است؛ پزشک تصمیم می‌گیرد آزمایش یا مکمل لازم است یا نه.","en":"Enough vitamin D and calcium matters for bone; your doctor decides whether a test or supplement is needed."}',
   'lab', 'pill', 'amber', 12, NULL, NULL, 14,
   NULL, 1, '["menopause"]', 0, 106, 'Menopause catalog (CB-MENO-01). [needs clinical review]', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_colon_screening', 'age_based', '{"fa":"غربالگری سرطان روده","en":"Bowel cancer screening"}',
   '{"fa":"از ۴۵ سالگی، طبق روش انتخابی","en":"From 45, depending on the method"}',
   '{"fa":"غربالگری روده می‌تواند پولیپ یا سرطان را زود پیدا کند. فاصله تکرار به روش (آزمایش مدفوع یا کولونوسکوپی) بستگی دارد.","en":"Bowel screening can find polyps or cancer early. How often depends on the method (stool test or colonoscopy)."}',
   'doctor', 'shieldCheck', 'violet', 12, 120, 45, 30,
   NULL, 1, '["menopause"]', 0, 107, 'Menopause catalog (CB-MENO-01). [needs clinical review]', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_thyroid', 'annual', '{"fa":"تیروئید (TSH)","en":"Thyroid (TSH)"}',
   '{"fa":"اگر خستگی یا تغییر وزن داری","en":"If you have tiredness or weight change"}',
   '{"fa":"مشکلات تیروئید در این سن شایع‌اند و علائمشان می‌تواند شبیه یائسگی باشد.","en":"Thyroid problems are common at this age and can look like menopause symptoms."}',
   'lab', 'flask', 'amber', 12, NULL, NULL, 14,
   NULL, 1, '["menopause"]', 0, 108, 'Menopause catalog (CB-MENO-01). [needs clinical review]', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('meno_eye_exam', 'annual', '{"fa":"معاینه چشم","en":"Eye exam"}',
   '{"fa":"سالانه","en":"Yearly"}',
   '{"fa":"معاینه منظم چشم تغییرات بینایی و بیماری‌هایی مثل آب سیاه را زود نشان می‌دهد.","en":"A regular eye exam shows vision changes and conditions such as glaucoma early."}',
   'doctor', 'stetho', 'green', 12, NULL, NULL, 30,
   NULL, 1, '["menopause"]', 0, 109, 'Menopause catalog (CB-MENO-01). [needs clinical review]', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'meno_score_items'
  AND `code` IN ('hot_flashes', 'heart_discomfort', 'sleep_problems', 'joint_muscle', 'depressive_mood', 'irritability', 'anxiety', 'exhaustion', 'sexual_problems', 'bladder_problems', 'vaginal_dryness');
DELETE FROM `catalog_items` WHERE `group` = 'meno_score_bands'
  AND `code` IN ('none', 'mild', 'moderate', 'severe');
DELETE FROM `catalog_items` WHERE `group` = 'meno_alerts'
  AND `code` IN ('postmenopausal_bleeding', 'heavy_perimenopause_bleeding', 'chest_pain_palpitations', 'one_sided_leg_swelling', 'breast_change', 'persistent_low_mood', 'fragility_fracture');
DELETE FROM `catalog_items` WHERE `group` = 'meno_tips'
  AND `code` IN ('stage_peri', 'stage_meno', 'stage_post', 'stage_unsure', 'log_bleeding_note', 'hot_flash_breathing', 'hrt_review', 'hrt_spotting', 'doctor_only', 'lifestyle_resistance', 'lifestyle_brisk_walk', 'lifestyle_relaxation', 'checkups_intro', 'patterns_disclaimer');
DELETE FROM `catalog_items` WHERE `group` = 'meno_checkup_groups'
  AND `code` IN ('heart_metabolic', 'bone', 'cancer_screening', 'other');
DELETE FROM `checkup_types` WHERE `user_id` IS NULL
  AND `key` IN ('meno_blood_pressure', 'meno_blood_sugar', 'meno_lipids', 'meno_weight_waist', 'meno_bone_density', 'meno_vitamin_d_calcium', 'meno_colon_screening', 'meno_thyroid', 'meno_eye_exam');
DROP TABLE IF EXISTS `side_effect_logs`;
DROP TABLE IF EXISTS `treatment_intakes`;
DROP TABLE IF EXISTS `treatment_items`;
DROP TABLE IF EXISTS `menopause_scores`;
DROP TABLE IF EXISTS `hot_flashes`;
ALTER TABLE `checkup_types` DROP COLUMN IF EXISTS `audiences`;
