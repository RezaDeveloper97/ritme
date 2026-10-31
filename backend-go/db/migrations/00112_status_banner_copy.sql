-- 00112_status_banner_copy.sql — Today status banner copy from the message engine (T-M10-06; DECISIONS.md M-14,
-- D-163). The engine decides *which* banner (today_ui.banner, E-17); one approved, locked `engine` rule per banner value
-- in the `status_banner` slot supplies its copy and CTA. rule_key = banner.<value>, condition today_ui.banner = <value>.
-- The fa text (the default language) and content.translations.en are T-M9-42's built-in copy
-- (frontend/messages/{fa,en}/home.json → home.nb.endMissing, home.nb.banner.overdue, home.nb.banner.askLast), with the
-- overdue banner's {n} spelled {days_late} (a fact). The overdue rule carries the base tier copy with the pregnancy
-- test button, so it only matches while the test is suggested and the period is at most 14 days late; the no-test and
-- 15+ day texts stay with the client's built-in copy (an empty slot falls back to it, M-14).
--
-- Data only (no schema change), so no Laravel twin: Laravel never serves the feed. ON DUPLICATE KEY keeps a database
-- whose rules were already seeded (or edited by an admin) as it is.

-- +goose Up
INSERT INTO `message_rules` (rule_key, title, level, kind, priority, placement, content, conditions, exceptions,
  variants_enabled, ab_split, cap_count, cap_period, cooldown_days, audience_percent, status, is_active, version,
  approved_version, locked, created_at, updated_at)
VALUES
('banner.end_missing', 'Status banner: period end missing', 'info', 'engine', 920, 'status_banner', '{"a":{"title":"پایان پریود هنوز ثبت نشده","body":"اگر تمام شده، ثبتش کن؛ اگر ادامه دارد، کاری لازم نیست.","cta_label":"ثبت پایان پریود","cta_target":"ritme://period/end"},"translations":{"en":{"a":{"title":"Period end not logged yet","body":"If it has ended, log it; if it''s still going, there''s nothing to do.","cta_label":"Log period end","cta_target":"ritme://period/end"}}}}', '[{"op":"and","items":[{"fact":"today_ui.banner","op":"eq","value":"end_missing"}]}]', '[]', 0, 50, NULL, NULL, 0, 100, 'approved', 1, 1, 1, 1, NOW(), NOW()),
('banner.overdue', 'Status banner: period overdue (pregnancy test offered, 1–14 days late)', 'info', 'engine', 910, 'status_banner', '{"a":{"title":"پریودت {days_late} روز دیر کرده","body":"اگر پریودت شروع شده، ثبتش کن. اگر احتمال بارداری هست، می‌توانی نتیجهٔ آزمایش را ثبت کنی.","cta_label":"ثبت تست بارداری","cta_target":"ritme://log/pregnancy-test"},"translations":{"en":{"a":{"title":"Your period is {days_late} days late","body":"If your period has started, log it. If you might be pregnant, you can log a test result.","cta_label":"Log pregnancy test","cta_target":"ritme://log/pregnancy-test"}}}}', '[{"op":"and","items":[{"fact":"today_ui.banner","op":"eq","value":"overdue"},{"fact":"suggest_pregnancy_test","op":"eq","value":true},{"fact":"days_late","op":"lte","value":14}]}]', '[]', 0, 50, NULL, NULL, 0, 100, 'approved', 1, 1, 1, 1, NOW(), NOW()),
('banner.ask_last_period', 'Status banner: ask for the last period date', 'info', 'engine', 900, 'status_banner', '{"a":{"title":"تاریخ آخرین پریودت را بگو","body":"پیش‌بینی‌ها از تاریخ آخرین پریودت حساب می‌شوند و فعلاً به‌روز نیستند.","cta_label":"ثبت تاریخ آخرین پریود","cta_target":"ritme://period/last"},"translations":{"en":{"a":{"title":"Tell us your last period date","body":"Predictions are counted from your last period date, and right now they are out of date.","cta_label":"Log last period date","cta_target":"ritme://period/last"}}}}', '[{"op":"and","items":[{"fact":"today_ui.banner","op":"eq","value":"ask_last_period"}]}]', '[]', 0, 50, NULL, NULL, 0, 100, 'approved', 1, 1, 1, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE `id` = `id`;

-- +goose Down
DELETE FROM `message_rules` WHERE rule_key IN ('banner.end_missing', 'banner.overdue', 'banner.ask_last_period');
