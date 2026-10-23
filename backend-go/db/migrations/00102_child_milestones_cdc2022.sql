-- 00102_child_milestones_cdc2022.sql — children: CDC 2022 milestone bands (T-M10-22, D-173; DECISIONS.md H-15).
-- Data only (catalog_items, groups of 00031):
--
--   15 and 30 months  new bands from CDC «Learn the Signs. Act Early.» 2022 (Zubler JM et al., Pediatrics
--                     2022;149:e2021052138): five milestones each (social, language, cognitive, motor), one play idea
--                     and the «when to mention it to the doctor» note, like the other bands. meta.band_source =
--                     cdc_2022; needs_review = 1 (listed in docs/engine-v6/clinical-signoff.md).
--   3 months          kept (H-15, advisor-pending), marked as Ritme's own band (not in CDC 2022): meta.band_source =
--                     ritme on its milestones, play ideas and doctor note. A band without the key is CDC.
--
-- Vaccine due ages are not here: they come from the clinical registry key child.vaccine_schedule (T-M10-16), in the
-- version in force on the child's birth date (H-17); the child_vaccines catalog keeps the names and copy.
--
-- INSERT IGNORE on (group, code), so a database whose Laravel twin ran is a no-op for the inserts. Down deletes the
-- new rows (with any checks users made on the new milestones) and removes the band_source key from the 3-month rows.
-- Twin of backend/database/migrations/2026_10_23_000102_child_milestones_cdc2022.php (same rows, `make schema-diff`
-- compares row counts). The copy needs a medical review before production.

-- +goose Up
INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('child_milestones', 'm15_shows_object', 58, 1, NULL,
   '{"fa":"چیزی را که دوست دارد به تو نشان می‌دهد","en":"Shows you an object they like"}',
   NULL,
   '{"age_months":15,"domain":"social","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestones', 'm15_tries_words', 59, 1, NULL,
   '{"fa":"غیر از ماما و بابا، یکی دو کلمه دیگر هم امتحان می‌کند","en":"Tries to say one or two words besides mama or dada"}',
   NULL,
   '{"age_months":15,"domain":"language","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestones', 'm15_points_for_help', 60, 1, NULL,
   '{"fa":"برای خواستن چیزی یا کمک گرفتن اشاره می‌کند","en":"Points to ask for something or to get help"}',
   NULL,
   '{"age_months":15,"domain":"language","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestones', 'm15_stacks_two', 61, 1, NULL,
   '{"fa":"دست‌کم دو چیز کوچک، مثل مکعب، را روی هم می‌گذارد","en":"Stacks at least two small objects, like blocks"}',
   NULL,
   '{"age_months":15,"domain":"cognitive","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestones', 'm15_takes_steps', 62, 1, NULL,
   '{"fa":"خودش چند قدم برمی‌دارد","en":"Takes a few steps on their own"}',
   NULL,
   '{"age_months":15,"domain":"motor","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestones', 'm30_look_at_me', 63, 1, NULL,
   '{"fa":"با گفتن «ببین!» کاری را که بلد است نشانت می‌دهد","en":"Shows you what they can do by saying “Look at me!”"}',
   NULL,
   '{"age_months":30,"domain":"social","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestones', 'm30_says_50_words', 64, 1, NULL,
   '{"fa":"حدود ۵۰ کلمه می‌گوید","en":"Says about 50 words"}',
   NULL,
   '{"age_months":30,"domain":"language","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestones', 'm30_action_phrase', 65, 1, NULL,
   '{"fa":"دو کلمه یا بیشتر را همراه یک فعل می‌گوید، مثل «هاپو دوید»","en":"Says two or more words together, with one action word, like “doggie run”"}',
   NULL,
   '{"age_months":30,"domain":"language","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestones', 'm30_pretend_play', 66, 1, NULL,
   '{"fa":"با چیزها بازی وانمودی می‌کند، مثلاً به عروسک مکعب می‌دهد که بخورد","en":"Uses things to pretend, like feeding a block to a doll"}',
   NULL,
   '{"age_months":30,"domain":"cognitive","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestones', 'm30_jumps_both_feet', 67, 1, NULL,
   '{"fa":"با هر دو پا از زمین می‌پرد","en":"Jumps off the ground with both feet"}',
   NULL,
   '{"age_months":30,"domain":"motor","band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestone_activities', 'm15_build_knock', 18, 1, NULL,
   '{"fa":"بساز و بریز","en":"Build and topple"}',
   '{"fa":"دو سه مکعب یا لیوان را روی هم بگذار و بگذار او خرابشان کند. بعد نوبت خودش است که بسازد.","en":"Stack two or three blocks or cups and let them knock them over. Then it’s their turn to build."}',
   '{"age_months":15,"band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestone_activities', 'm30_jump_line', 19, 1, NULL,
   '{"fa":"پرش از روی خط","en":"Jump the line"}',
   '{"fa":"با نوار چسب روی زمین خطی بکش و با هم از رویش بپرید. هر پرش را بلند بشمار.","en":"Put a strip of tape on the floor and jump over it together, counting each jump out loud."}',
   '{"age_months":30,"band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestone_notes', 'm15', 12, 1, NULL,
   '{"fa":"مراحل رشد ۱۵ ماهگی","en":"15-month milestones"}',
   '{"fa":"اگر تا ۱۵ ماهگی هیچ کلمه‌ای نمی‌گوید، برای نشان دادن چیزی اشاره نمی‌کند یا مهارتی را که داشت از دست داده، در مراجعه بعدی با پزشک کودک در میان بگذار.","en":"If by 15 months they say no words, don’t point to show you things, or lose a skill they had, mention it to the paediatrician at the next visit."}',
   '{"age_months":15,"band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('child_milestone_notes', 'm30', 13, 1, NULL,
   '{"fa":"مراحل رشد ۳۰ ماهگی","en":"30-month milestones"}',
   '{"fa":"اگر تا ۳۰ ماهگی دو کلمه را با هم نمی‌گوید، کارهای ساده را تقلید نمی‌کند یا مهارتی را که داشت از دست داده، با پزشک کودک صحبت کن.","en":"If by 30 months they don’t put two words together, don’t copy simple actions, or lose a skill they had, talk to the paediatrician."}',
   '{"age_months":30,"band_source":"cdc_2022"}', 1,
   CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

UPDATE `catalog_items` SET `meta` = JSON_SET(COALESCE(`meta`, '{}'), '$.band_source', 'ritme')
WHERE (`group` = 'child_milestones' AND `code` IN ('m3_social_smile', 'm3_head_up_tummy', 'm3_reacts_to_sounds', 'm3_hands_to_mouth', 'm3_coos_vowels', 'm3_tracks_objects', 'm3_hands_together'))
   OR (`group` = 'child_milestone_activities' AND `code` IN ('m3_tummy_time', 'm3_sound_talk'))
   OR (`group` = 'child_milestone_notes' AND `code` = 'm3');

-- +goose Down
UPDATE `catalog_items` SET `meta` = JSON_REMOVE(`meta`, '$.band_source')
WHERE (`group` = 'child_milestones' AND `code` IN ('m3_social_smile', 'm3_head_up_tummy', 'm3_reacts_to_sounds', 'm3_hands_to_mouth', 'm3_coos_vowels', 'm3_tracks_objects', 'm3_hands_together'))
   OR (`group` = 'child_milestone_activities' AND `code` IN ('m3_tummy_time', 'm3_sound_talk'))
   OR (`group` = 'child_milestone_notes' AND `code` = 'm3');

DELETE FROM `child_milestone_checks` WHERE `milestone_code` IN ('m15_shows_object', 'm15_tries_words', 'm15_points_for_help', 'm15_stacks_two', 'm15_takes_steps', 'm30_look_at_me', 'm30_says_50_words', 'm30_action_phrase', 'm30_pretend_play', 'm30_jumps_both_feet');
DELETE FROM `catalog_items` WHERE `group` = 'child_milestones' AND `code` IN ('m15_shows_object', 'm15_tries_words', 'm15_points_for_help', 'm15_stacks_two', 'm15_takes_steps', 'm30_look_at_me', 'm30_says_50_words', 'm30_action_phrase', 'm30_pretend_play', 'm30_jumps_both_feet');
DELETE FROM `catalog_items` WHERE `group` = 'child_milestone_activities' AND `code` IN ('m15_build_knock', 'm30_jump_line');
DELETE FROM `catalog_items` WHERE `group` = 'child_milestone_notes' AND `code` IN ('m15', 'm30');
