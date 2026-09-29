-- 00007_checkups_catalog_icons.sql — data only (task T-M4-12, design audit B6): the default checkup
-- catalog seeded by 00003 used icons/tones that do not match the v14 artboards. The artboards use:
-- clinical breast exam = stethoscope/amber, Pap smear = shield/violet, blood test = flask/amber (the
-- artboard's teal is the data accent, frontend/CLAUDE.md §10.2), dentist = tooth/green,
-- mammography = ribbon/neutral.
--
-- Guarded: a row changes only while its icon AND tone are still the 00003 seed values, so a catalog
-- entry an admin already edited is left alone. Re-running is a no-op. `updated_at` is not touched.
-- Down reverses it with the same guard.
-- No Laravel twin: data only, `make schema-diff` (schema + seed row counts) is unaffected.
-- Test: db/migrations/checkups_catalog_icons_int_test.go (`make test-int PKG=./db/migrations/...`).

-- +goose Up
UPDATE `checkup_types` SET `icon` = 'stetho', `tone` = 'amber'
 WHERE `key` = 'clinical_breast_exam' AND `user_id` IS NULL AND `icon` = 'breast' AND `tone` = 'violet';
UPDATE `checkup_types` SET `icon` = 'shield', `tone` = 'violet'
 WHERE `key` = 'pap_smear' AND `user_id` IS NULL AND `icon` = 'flask' AND `tone` = 'violet';
UPDATE `checkup_types` SET `icon` = 'flask', `tone` = 'amber'
 WHERE `key` = 'blood_test' AND `user_id` IS NULL AND `icon` = 'blood' AND `tone` = 'amber';
UPDATE `checkup_types` SET `icon` = 'tooth', `tone` = 'green'
 WHERE `key` = 'dentist' AND `user_id` IS NULL AND `icon` = 'tooth' AND `tone` = 'teal';
UPDATE `checkup_types` SET `icon` = 'ribbon', `tone` = 'neutral'
 WHERE `key` = 'mammography' AND `user_id` IS NULL AND `icon` = 'shieldCheck' AND `tone` = 'rose';

-- +goose Down
UPDATE `checkup_types` SET `icon` = 'breast', `tone` = 'violet'
 WHERE `key` = 'clinical_breast_exam' AND `user_id` IS NULL AND `icon` = 'stetho' AND `tone` = 'amber';
UPDATE `checkup_types` SET `icon` = 'flask', `tone` = 'violet'
 WHERE `key` = 'pap_smear' AND `user_id` IS NULL AND `icon` = 'shield' AND `tone` = 'violet';
UPDATE `checkup_types` SET `icon` = 'blood', `tone` = 'amber'
 WHERE `key` = 'blood_test' AND `user_id` IS NULL AND `icon` = 'flask' AND `tone` = 'amber';
UPDATE `checkup_types` SET `icon` = 'tooth', `tone` = 'teal'
 WHERE `key` = 'dentist' AND `user_id` IS NULL AND `icon` = 'tooth' AND `tone` = 'green';
UPDATE `checkup_types` SET `icon` = 'shieldCheck', `tone` = 'rose'
 WHERE `key` = 'mammography' AND `user_id` IS NULL AND `icon` = 'ribbon' AND `tone` = 'neutral';
