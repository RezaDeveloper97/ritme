-- 00NNN_vitals_alert_threshold_placeholders.sql — T-M10-19 (D-170): the seeded vitals_alert copy (goose 00035) spells
-- the thresholds; replace them with the placeholders the API fills from the registry version of the reading's day
-- ({systolic}, {diastolic}, {mgdl}, {mmol_l}). Only rows still holding the seeded numbers change (an admin edit that
-- already rewrote the sentence is left alone). Data-only: no Laravel twin needed for schema-diff (prod Laravel does not
-- read vitals_alert). Down puts the numbers back.

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
UPDATE `message_contents` SET `payload` = REPLACE(`payload`, '{systolic}/{diastolic}', '۱۸۰/۱۲۰')
WHERE `group` = 'vitals_alert' AND `item_key` = 'bp_crisis' AND `locale` = 'fa';
UPDATE `message_contents` SET `payload` = REPLACE(`payload`, '{systolic}/{diastolic}', '180/120')
WHERE `group` = 'vitals_alert' AND `item_key` = 'bp_crisis' AND `locale` = 'en';
UPDATE `message_contents` SET `payload` = REPLACE(`payload`, '{mgdl} mg/dL ({mmol_l} mmol/L)', '۵۴ mg/dL (۳٫۰ mmol/L)')
WHERE `group` = 'vitals_alert' AND `item_key` = 'glucose_low' AND `locale` = 'fa';
UPDATE `message_contents` SET `payload` = REPLACE(`payload`, '{mgdl} mg/dL ({mmol_l} mmol/L)', '54 mg/dL (3.0 mmol/L)')
WHERE `group` = 'vitals_alert' AND `item_key` = 'glucose_low' AND `locale` = 'en';
