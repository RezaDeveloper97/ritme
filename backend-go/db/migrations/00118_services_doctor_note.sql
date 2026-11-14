-- 00118_services_doctor_note.sql — the last card of the «خدمات» hub is no longer an emergency warning with a 115 call:
-- it is a calm note that Ritme does not replace a doctor. Copy only (data, no schema); rows an admin already edited
-- are left alone. meta.phone stays in place but the card no longer shows it.

-- +goose Up
UPDATE `catalog_items`
SET `title` = '{"fa":"ریتمی جای پزشک نیست","en":"Ritme doesn''t replace your doctor"}',
    `body`  = '{"fa":"اگر نشانه‌ای دیدی که نگرانت کرد، حتماً با پزشک مشورت کن.","en":"If you notice anything that worries you, talk to a doctor."}',
    `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `group` = 'services_sections' AND `code` = 'emergency'
  AND JSON_UNQUOTE(JSON_EXTRACT(`title`, '$.fa')) = 'اورژانس است؟';

-- +goose Down
UPDATE `catalog_items`
SET `title` = '{"fa":"اورژانس است؟","en":"Is it an emergency?"}',
    `body`  = '{"fa":"ریتمی جایگزین اورژانس نیست","en":"Ritme is not a substitute for emergency care"}',
    `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `group` = 'services_sections' AND `code` = 'emergency'
  AND JSON_UNQUOTE(JSON_EXTRACT(`title`, '$.fa')) = 'ریتمی جای پزشک نیست';
