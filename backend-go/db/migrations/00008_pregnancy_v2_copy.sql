-- 00008_pregnancy_v2_copy.sql — data only (task T-M7-20): pregnancy v2 copy leftovers of T-M7-18 / T-M2-34.
--
--  1. Design audit F4: the seeded `pregnancy_alert/week_entered` text «هفتهٔ {week} بارداری از امروز شروع شده»
--     ("… started today") is wrong on any day but the week's first (the rule fires on the first evaluation of
--     the week); the card already shows the fact date, so the text drops «از امروز» / "today".
--     Guarded: a row changes only while its `what_we_saw` is byte-for-byte the 00005 seed (CAST AS BINARY:
--     utf8mb4_unicode_ci would ignore ZWNJ and case); an admin-edited text is left alone.
--  2. Backend review #11: `pregnancy_alert/weight_missing_week` gets `params.from_weekday` = 5 (the engine
--     default, pregnancyalerts.WeightMissingFromWeekday; now editable in admin). Guarded: only while
--     `params` is still exactly the seed {"from_week":1}, so no behaviour changes (5 is what applied).
--  3. Design audit E2: the calendar's source note becomes admin-editable — new item
--     `pregnancy_setup/calendar_note` (registry: plan_note + basis_<source>), seeded fa + en with the Go
--     lang text (internal/pregnancy/v2/calendar/lang), which stays the fallback. INSERT IGNORE: an existing
--     row (admin-created) is kept.
--
-- `updated_at` is not touched by the updates (a data fix, not an edit). Re-running is a no-op. Down reverses
-- each step with the same guard: texts only while they are still the new value, from_weekday only while
-- params are {"from_week":1,"from_weekday":5}, calendar_note rows only while untouched (payload = seed and
-- updated_at = created_at).
-- Laravel twin for step 3 only (row counts for `make schema-diff`):
-- backend/database/migrations/2026_09_29_000001_seed_pregnancy_calendar_note.php. Steps 1 and 2 are guarded
-- updates with no row-count effect; prod (Laravel) gets them at cutover (T-M2-27).
-- Test: db/migrations/pregnancy_v2_copy_int_test.go (`make test-int PKG=./db/migrations/...`).

-- +goose Up
UPDATE `message_contents`
   SET `payload` = JSON_SET(`payload`, '$.what_we_saw', 'هفتهٔ {week} بارداری شروع شده')
 WHERE `group` = 'pregnancy_alert' AND `item_key` = 'week_entered' AND `locale` = 'fa'
   AND CAST(JSON_VALUE(`payload`, '$.what_we_saw') AS BINARY) = CAST('هفتهٔ {week} بارداری از امروز شروع شده' AS BINARY);

UPDATE `message_contents`
   SET `payload` = JSON_SET(`payload`, '$.what_we_saw', 'Pregnancy week {week} has started')
 WHERE `group` = 'pregnancy_alert' AND `item_key` = 'week_entered' AND `locale` = 'en'
   AND CAST(JSON_VALUE(`payload`, '$.what_we_saw') AS BINARY) = CAST('Pregnancy week {week} started today' AS BINARY);

UPDATE `message_contents`
   SET `payload` = JSON_SET(`payload`, '$.params.from_weekday', 5)
 WHERE `group` = 'pregnancy_alert' AND `item_key` = 'weight_missing_week'
   AND JSON_LENGTH(`payload`, '$.params') = 1 AND JSON_VALUE(`payload`, '$.params.from_week') = '1';

INSERT IGNORE INTO `message_contents` (`group`, `item_key`, `locale`, `label`, `payload`, `is_active`, `is_approved`, `sort_order`, `created_at`, `updated_at`) VALUES
  ('pregnancy_setup', 'calendar_note', 'fa', 'pregnancy_setup / calendar_note', '{"plan_note":"زمان‌ها بر اساس برنامهٔ رایج مراقبت‌های بارداری‌اند و ممکنه پزشکت برنامهٔ متفاوتی بده.","basis_lmp":"تاریخ‌ها بر اساس اولین روز آخرین قاعدگی محاسبه شده‌اند.","basis_ultrasound":"تاریخ‌ها بر اساس سونوگرافی محاسبه شده‌اند.","basis_manual":"تاریخ‌ها بر اساس سن بارداری واردشده محاسبه شده‌اند."}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('pregnancy_setup', 'calendar_note', 'en', 'pregnancy_setup / calendar_note', '{"plan_note":"Timings follow the usual pregnancy care schedule; your doctor may give you a different plan.","basis_lmp":"Dates are based on the first day of your last period.","basis_ultrasound":"Dates are based on your ultrasound.","basis_manual":"Dates are based on the pregnancy age you entered."}', 1, 1, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `message_contents`
 WHERE `group` = 'pregnancy_setup' AND `item_key` = 'calendar_note' AND `locale` = 'fa'
   AND `label` = 'pregnancy_setup / calendar_note' AND `updated_at` <=> `created_at`
   AND CAST(`payload` AS BINARY) = CAST('{"plan_note":"زمان‌ها بر اساس برنامهٔ رایج مراقبت‌های بارداری‌اند و ممکنه پزشکت برنامهٔ متفاوتی بده.","basis_lmp":"تاریخ‌ها بر اساس اولین روز آخرین قاعدگی محاسبه شده‌اند.","basis_ultrasound":"تاریخ‌ها بر اساس سونوگرافی محاسبه شده‌اند.","basis_manual":"تاریخ‌ها بر اساس سن بارداری واردشده محاسبه شده‌اند."}' AS BINARY);
DELETE FROM `message_contents`
 WHERE `group` = 'pregnancy_setup' AND `item_key` = 'calendar_note' AND `locale` = 'en'
   AND `label` = 'pregnancy_setup / calendar_note' AND `updated_at` <=> `created_at`
   AND CAST(`payload` AS BINARY) = CAST('{"plan_note":"Timings follow the usual pregnancy care schedule; your doctor may give you a different plan.","basis_lmp":"Dates are based on the first day of your last period.","basis_ultrasound":"Dates are based on your ultrasound.","basis_manual":"Dates are based on the pregnancy age you entered."}' AS BINARY);

UPDATE `message_contents`
   SET `payload` = JSON_REMOVE(`payload`, '$.params.from_weekday')
 WHERE `group` = 'pregnancy_alert' AND `item_key` = 'weight_missing_week'
   AND JSON_LENGTH(`payload`, '$.params') = 2 AND JSON_VALUE(`payload`, '$.params.from_week') = '1'
   AND JSON_VALUE(`payload`, '$.params.from_weekday') = '5';

UPDATE `message_contents`
   SET `payload` = JSON_SET(`payload`, '$.what_we_saw', 'هفتهٔ {week} بارداری از امروز شروع شده')
 WHERE `group` = 'pregnancy_alert' AND `item_key` = 'week_entered' AND `locale` = 'fa'
   AND CAST(JSON_VALUE(`payload`, '$.what_we_saw') AS BINARY) = CAST('هفتهٔ {week} بارداری شروع شده' AS BINARY);

UPDATE `message_contents`
   SET `payload` = JSON_SET(`payload`, '$.what_we_saw', 'Pregnancy week {week} started today')
 WHERE `group` = 'pregnancy_alert' AND `item_key` = 'week_entered' AND `locale` = 'en'
   AND CAST(JSON_VALUE(`payload`, '$.what_we_saw') AS BINARY) = CAST('Pregnancy week {week} has started' AS BINARY);
