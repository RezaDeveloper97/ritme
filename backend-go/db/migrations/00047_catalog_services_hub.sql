-- 00047_catalog_services_hub.sql — content of the «خدمات» hub (bloom B-N7-01, artboards nbl_/nbd_v17_Main,
-- internal/services): three admin-editable catalog groups (docs/canvas-build/catalog.md §5 — data only, no schema).
--
-- services_sections  one item per hub section (codes = services.SectionCodes): is_active hides it, sort_order orders
--                    it, title / body = heading / sub-heading; meta {href?, caption?, phone?, categories?}. The
--                    emergency card always shows (an inactive row only loses its copy, never the 115 call).
-- services_care      «مراقبت سلامت» tiles; meta {icon, tone, href?, counter?}.
-- services_programs  «برنامه‌های مراقبتی» cards; meta {icon, tone, href?}; audiences = life modes that see the card.
--
-- No href = «به‌زودی» (not tappable): only screens that exist today are linked (record, labs, vitals, checkups,
-- contraception, search). Admins set meta.href when a screen ships (doctors B-N7-02..05, assistant B-N7-06/07,
-- learning N8, shop / insurance / city services N10, condition programs CB-COND, pelvic floor CB-PELV-02).
-- Navigation labels, not clinical copy → needs_review 0. INSERT IGNORE never overwrites an admin edit.
-- Twin of backend/database/migrations/2026_10_07_000047_seed_services_hub_catalog.php (same rows), so
-- `make schema-diff` row counts match. Go only API (deviations.md D-69).

-- +goose Up
INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
  ('services_sections', 'search', 1, 1, NULL, '{"fa":"پزشک، آزمایش، کلاس یا برنامه…","en":"Doctors, labs, classes or programs…"}',
   NULL,
   '{"href":"/search"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_sections', 'booking', 2, 1, NULL, '{"fa":"نوبت پیش رو","en":"Upcoming visit"}',
   NULL,
   NULL, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_sections', 'care', 3, 1, NULL, '{"fa":"مراقبت سلامت","en":"Health care"}',
   NULL,
   NULL, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_sections', 'checkups', 4, 1, NULL, '{"fa":"چکاپ‌های دوره‌ای و یادآور دارو","en":"Routine checkups & medication reminders"}',
   NULL,
   '{"href":"/checkups"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_sections', 'programs', 5, 1, NULL, '{"fa":"برنامه‌های مراقبتی","en":"Care programs"}',
   '{"fa":"بر اساس چیزی که خودت فعال کنی","en":"Based on what you turn on"}',
   NULL, 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_sections', 'mother_child', 6, 1, NULL, '{"fa":"برای مادر و کودک","en":"For mother & child"}',
   NULL,
   '{"caption":{"fa":"کلاس، استخر و خانه بازی نزدیک تو","en":"Classes, pools and play centres near you"}}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_sections', 'learning', 7, 1, NULL, '{"fa":"آموزش","en":"Learning"}',
   '{"fa":"دوره‌ها و کلاس‌های آنلاین","en":"Online courses and classes"}',
   '{"caption":{"fa":"آمادگی زایمان، شیردهی، یائسگی","en":"Birth prep, breastfeeding, menopause"}}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_sections', 'shop', 8, 1, NULL, '{"fa":"فروشگاه","en":"Shop"}',
   '{"fa":"فروشگاه از داده سلامت تو جداست و پیشنهادهایش بر اساس ثبت‌هایت نیست.","en":"The shop is separate from your health data and its suggestions are not based on what you log."}',
   '{"categories":[{"code":"layette","icon":"bottle","title":{"fa":"سیسمونی و نوزاد","en":"Layette & baby"}},{"code":"beauty","icon":"dropLine","title":{"fa":"آرایشی و بهداشتی","en":"Beauty & hygiene"}}]}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_sections', 'emergency', 9, 1, NULL, '{"fa":"اورژانس است؟","en":"Is it an emergency?"}',
   '{"fa":"ریتمی جایگزین اورژانس نیست","en":"Ritme is not a substitute for emergency care"}',
   '{"phone":"115"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_care', 'assistant', 1, 1, NULL, '{"fa":"دستیار سلامت","en":"Health assistant"}',
   '{"fa":"پاسخ و ارجاع","en":"Answers and referrals"}',
   '{"icon":"sparkle","tone":"brand"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_care', 'doctors', 2, 1, NULL, '{"fa":"پزشک و ماما","en":"Doctors & midwives"}',
   '{"fa":"ویدیویی، حضوری","en":"Video or in person"}',
   '{"icon":"stetho","tone":"data"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_care', 'record', 3, 1, NULL, '{"fa":"پرونده سلامت","en":"Health record"}',
   '{"fa":"سوابق و مدارک","en":"History and documents"}',
   '{"icon":"fileDoc","tone":"brand","href":"/record","counter":"record_documents"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_care', 'labs', 4, 1, NULL, '{"fa":"تحلیل آزمایش","en":"Lab analysis"}',
   '{"fa":"عکس برگه آزمایش","en":"Photo of your lab sheet"}',
   '{"icon":"flask","tone":"period","href":"/labs"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_care', 'vitals', 5, 1, NULL, '{"fa":"علائم حیاتی","en":"Vital signs"}',
   '{"fa":"فشار، قند، ضربان","en":"Blood pressure, sugar, pulse"}',
   '{"icon":"heartLine","tone":"period","href":"/vitals"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_care', 'insurance', 6, 1, NULL, '{"fa":"بیمه","en":"Insurance"}',
   '{"fa":"پوشش و خسارت","en":"Coverage and claims"}',
   '{"icon":"shield","tone":"data"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_programs', 'pain_endometriosis', 1, 1, NULL, '{"fa":"درد و اندومتریوز","en":"Pain & endometriosis"}',
   '{"fa":"دفترچه درد و گزارش","en":"Pain diary and report"}',
   '{"icon":"flame","tone":"period"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_programs', 'pmdd', 2, 1, NULL, '{"fa":"PMDD و خلق","en":"PMDD & mood"}',
   '{"fa":"پرسشنامه ماهانه","en":"Monthly questionnaire"}',
   '{"icon":"smile","tone":"brand"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_programs', 'heavy_bleeding', 3, 1, NULL, '{"fa":"خونریزی زیاد","en":"Heavy bleeding"}',
   '{"fa":"جدول خونریزی","en":"Bleeding chart"}',
   '{"icon":"drop","tone":"period"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_programs', 'pelvic_floor', 4, 1, NULL, '{"fa":"کف لگن","en":"Pelvic floor"}',
   '{"fa":"برنامه ۸ هفته‌ای","en":"8-week program"}',
   '{"icon":"target","tone":"data"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('services_programs', 'contraception', 5, 1, '["cycle","postpartum","menopause"]', '{"fa":"پیشگیری","en":"Contraception"}',
   '{"fa":"قرص و روش‌ها","en":"Pill and methods"}',
   '{"icon":"pill","tone":"brand","href":"/contraception"}', 0, CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

-- +goose Down
DELETE FROM `catalog_items` WHERE `group` = 'services_sections' AND `code` IN ('search', 'booking', 'care', 'checkups', 'programs', 'mother_child', 'learning', 'shop', 'emergency');
DELETE FROM `catalog_items` WHERE `group` = 'services_care' AND `code` IN ('assistant', 'doctors', 'record', 'labs', 'vitals', 'insurance');
DELETE FROM `catalog_items` WHERE `group` = 'services_programs' AND `code` IN ('pain_endometriosis', 'pmdd', 'heavy_bleeding', 'pelvic_floor', 'contraception');
