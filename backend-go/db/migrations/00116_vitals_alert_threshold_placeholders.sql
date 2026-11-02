-- 00116_vitals_alert_threshold_placeholders.sql — T-M10-19b (D-170): the seeded vitals_alert copy (goose 00035) spells
-- the thresholds («۱۸۰/۱۲۰», «۵۴ mg/dL (۳٫۰ mmol/L)»); replace them with the placeholders the API fills from the
-- clinical config version of the reading's day ({systolic}, {diastolic}, {mgdl}, {mmol_l}; vitals.CopyParams). Only
-- rows still holding the seeded numbers change: REPLACE matches the seeded wording, so an admin edit that already
-- rewrote the sentence is left alone. The wording is otherwise as-is (user decision 2026-10-23: no D-90/D-160 safety
-- approval flow for this move; later wording changes go through it). Data only; Laravel twin
-- 2026_11_02_000116_vitals_alert_threshold_placeholders.php. Down puts the numbers back.

-- +goose Up
UPDATE `message_contents`
SET `payload` = REPLACE(`payload`, '۱۸۰/۱۲۰', '{systolic}/{diastolic}')
WHERE `group` = 'vitals_alert' AND `item_key` = 'bp_crisis' AND `locale` = 'fa';

UPDATE `message_contents`
SET `payload` = REPLACE(`payload`, '180/120', '{systolic}/{diastolic}')
WHERE `group` = 'vitals_alert' AND `item_key` = 'bp_crisis' AND `locale` = 'en';

UPDATE `message_contents`
SET `payload` = REPLACE(`payload`, '۵۴ mg/dL (۳٫۰ mmol/L)', '{mgdl} mg/dL ({mmol_l} mmol/L)')
WHERE `group` = 'vitals_alert' AND `item_key` = 'glucose_low' AND `locale` = 'fa';

UPDATE `message_contents`
SET `payload` = REPLACE(`payload`, '54 mg/dL (3.0 mmol/L)', '{mgdl} mg/dL ({mmol_l} mmol/L)')
WHERE `group` = 'vitals_alert' AND `item_key` = 'glucose_low' AND `locale` = 'en';

-- +goose Down
UPDATE `message_contents`
SET `payload` = REPLACE(`payload`, '{systolic}/{diastolic}', '۱۸۰/۱۲۰')
WHERE `group` = 'vitals_alert' AND `item_key` = 'bp_crisis' AND `locale` = 'fa';

UPDATE `message_contents`
SET `payload` = REPLACE(`payload`, '{systolic}/{diastolic}', '180/120')
WHERE `group` = 'vitals_alert' AND `item_key` = 'bp_crisis' AND `locale` = 'en';

UPDATE `message_contents`
SET `payload` = REPLACE(`payload`, '{mgdl} mg/dL ({mmol_l} mmol/L)', '۵۴ mg/dL (۳٫۰ mmol/L)')
WHERE `group` = 'vitals_alert' AND `item_key` = 'glucose_low' AND `locale` = 'fa';

UPDATE `message_contents`
SET `payload` = REPLACE(`payload`, '{mgdl} mg/dL ({mmol_l} mmol/L)', '54 mg/dL (3.0 mmol/L)')
WHERE `group` = 'vitals_alert' AND `item_key` = 'glucose_low' AND `locale` = 'en';
