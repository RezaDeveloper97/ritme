-- 00100_checkups_uspstf_catalog.sql — checkups catalog on SPEC v6 §3.4 (T-M10-21, D-172; DECISIONS.md H-10, H-11,
-- H-16). Data only: the schema columns came with 00099 (checkup_records.test_kind, checkup_types.informational).
--
--   bp_screening, glucose_screening  new catalog types (USPSTF 2021) with fa/en copy; their interval and ages come
--                                    from the clinical registry key screening.uspstf (the catalog columns are its
--                                    version 1, the fallback). Not shown to menopause mode, which has its own
--                                    meno_blood_pressure / meno_blood_sugar rows; labelled «after pregnancy» while
--                                    pregnant (hide_in_pregnancy = 1, prenatal care checks both).
--   breast_self_exam                 informational = 1, «آگاهی از سینه» / breast awareness (H-10): no due date, no
--                                    reminder, history read-only.
--   pap_smear                        copy names the HPV option (interval by the last test kind, H-11).
--   mammography                      catalog columns synced to the registry (every 24 months, 40–74, USPSTF 2024).
--   dentist                          source note only. clinical_breast_exam and blood_test unchanged (H-16).
--
-- The UPDATEs overwrite an admin edit of these columns on the four seeded rows (catalog review). INSERT IGNORE on the
-- unique key, so a database whose Laravel twin ran is a no-op for the inserts. Down deletes the two new types (with
-- any records users logged for them) and restores the 00003 values.
--
-- Twin of backend/database/migrations/2026_10_22_000100_checkups_uspstf_catalog.php (same rows, `make schema-diff`
-- compares row counts). The copy needs a medical review before production.

-- +goose Up

INSERT IGNORE INTO `checkup_types` (`key`, `category`, `title`, `subtitle`, `why`, `performed_by`, `icon`, `tone`, `interval_months`, `interval_months_max`, `age_min`, `age_max`, `remind_lead_days`, `prep_steps`, `hide_in_pregnancy`, `audiences`, `is_active`, `sort_order`, `source_note`, `created_at`, `updated_at`) VALUES
  ('bp_screening', 'annual', '{"fa":"اندازه‌گیری فشار خون","en":"Blood pressure check"}', '{"fa":"نزد پزشک یا در داروخانه","en":"With your doctor or at a pharmacy"}', '{"fa":"فشار خون بالا معمولاً هیچ علامتی ندارد و فقط با اندازه‌گیری پیدا می‌شود. از ۱۸ تا ۳۹ سالگی هر ۳ سال یک‌بار و از ۴۰ سالگی هر سال یک‌بار فشارت را اندازه بگیر. اگر فشارت قبلاً بالا بوده، پزشک ممکن است زودتر بخواهد.","en":"High blood pressure usually has no symptoms and is found only by measuring it. From 18 to 39 have it checked every 3 years, and every year from 40. If it has been high before, your doctor may want it sooner."}', 'doctor', 'heart', 'rose', 12, NULL, 18, NULL, 14, '[{"fa":"۳۰ دقیقه قبل، قهوه، چای پررنگ و سیگار را کنار بگذار","en":"Skip coffee, strong tea and smoking for 30 minutes before"},{"fa":"۵ دقیقه آرام بنشین و کف پاهایت را روی زمین بگذار","en":"Sit quietly for 5 minutes with your feet flat on the floor"},{"fa":"اگر در خانه فشارت را ثبت کرده‌ای، عددها را همراه ببر","en":"If you log readings at home, bring them along"}]', 1, '["cycle","ttc","pregnancy","postpartum","teen"]', 1, 7, 'USPSTF 2021, Hypertension in Adults: Screening (18-39 every 3 years, 40 and older yearly). Interval and ages from the clinical registry key screening.uspstf (T-M10-21, D-172). Needs medical review.', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')),
  ('glucose_screening', 'multi_year', '{"fa":"قند خون (غربالگری دیابت)","en":"Blood sugar (diabetes screening)"}', '{"fa":"قند ناشتا یا HbA1c","en":"Fasting sugar or HbA1c"}', '{"fa":"پیش‌دیابت و دیابت نوع ۲ می‌توانند سال‌ها بی‌صدا پیش بروند. اگر ۳۵ تا ۷۰ سال داری و BMI تو ۲۵ یا بیشتر است، هر ۳ سال یک آزمایش قند توصیه می‌شود. وقتی زود پیدا شود، اغلب با تغییر سبک زندگی می‌شود جلویش را گرفت.","en":"Prediabetes and type 2 diabetes can progress silently for years. If you are 35 to 70 and your BMI is 25 or higher, a blood sugar test every 3 years is recommended. Found early, it can often be turned around with lifestyle changes."}', 'lab', 'blood', 'amber', 36, NULL, 35, 70, 14, '[{"fa":"برای قند ناشتا، ۸ ساعت قبل از آزمایش فقط آب بنوش","en":"For a fasting test, have only water for 8 hours before"},{"fa":"برای HbA1c ناشتا بودن لازم نیست","en":"No fasting is needed for HbA1c"},{"fa":"داروهایی را که مصرف می‌کنی به آزمایشگاه بگو","en":"Tell the lab which medicines you take"}]', 1, '["cycle","ttc","pregnancy","postpartum","teen"]', 1, 8, 'USPSTF 2021, Prediabetes and Type 2 Diabetes: Screening (35-70 with overweight or obesity, every 3 years). BMI condition, interval and ages from the clinical registry key screening.uspstf (T-M10-21, D-172). Needs medical review.', CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));

UPDATE `checkup_types` SET `title` = '{"fa":"آگاهی از سینه","en":"Breast awareness"}', `subtitle` = '{"fa":"حالت طبیعی سینه‌هایت را بشناس","en":"Know what is normal for you"}', `why` = '{"fa":"لازم نیست هر ماه در روز مشخصی معاینه کنی. کافی است بدانی سینه‌هایت معمولاً چه شکلی‌اند و چه حسی دارند تا هر تغییری را زود ببینی: توده یا سفتی، تغییر پوست، فرورفتن نوک سینه یا ترشح. اگر چیز تازه‌ای دیدی، منتظر نمان و به پزشک سر بزن.","en":"You don''t need a monthly exam on a set day. It''s enough to know how your breasts usually look and feel, so you notice any change early: a lump or thickening, skin changes, a pulled-in nipple or discharge. If you notice something new, don''t wait; see your doctor."}', `informational` = 1, `source_note` = 'Breast awareness without a scheduled self-exam (USPSTF 2024 breast cancer screening; DECISIONS.md H-10, advisor-pending). Informational: no due date, history read-only (T-M10-21). Needs medical review.', `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `user_id` IS NULL AND `key` = 'breast_self_exam';

UPDATE `checkup_types` SET `why` = '{"fa":"پاپ‌اسمیر و تست HPV تغییرات دهانه رحم را پیش از آنکه به مشکل جدی تبدیل شوند نشان می‌دهند. از ۲۱ تا ۲۹ سالگی هر ۳ سال یک پاپ‌اسمیر توصیه می‌شود. از ۳۰ تا ۶۵ سالگی سه راه داری: پاپ‌اسمیر هر ۳ سال، تست HPV هر ۵ سال، یا هر دو با هم هر ۵ سال. موقع ثبت، نوع آزمایش را انتخاب کن تا موعد بعدی درست حساب شود.","en":"A Pap smear and an HPV test show changes in the cervix before they become a serious problem. From 21 to 29 a Pap smear every 3 years is recommended. From 30 to 65 you have three options: a Pap smear every 3 years, an HPV test every 5 years, or both together every 5 years. When you log it, pick the test type so your next date is counted right."}', `source_note` = 'USPSTF 2018 cervical cancer screening (21-29 cytology every 3 years; 30-65 cytology every 3 years, HPV or co-test every 5 years). Interval by the last test kind from the clinical registry key screening.uspstf (DECISIONS.md H-11, T-M10-21). Needs medical review.', `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `user_id` IS NULL AND `key` = 'pap_smear';

UPDATE `checkup_types` SET `why` = '{"fa":"ماموگرافی می‌تواند توده‌هایی را سال‌ها پیش از آنکه لمس شوند نشان دهد. از ۴۰ تا ۷۴ سالگی هر ۲ سال یک‌بار توصیه می‌شود. با سابقه خانوادگی ممکن است پزشک زودتر شروع کند یا زودبه‌زود بخواهد.","en":"A mammogram can show lumps years before they can be felt. From 40 to 74 it is recommended every 2 years. With a family history your doctor may start earlier or ask for it more often."}', `interval_months` = 24, `interval_months_max` = NULL, `age_max` = 74, `source_note` = 'USPSTF 2024 breast cancer screening (40-74 every 2 years). Interval and ages from the clinical registry key screening.uspstf (T-M10-21). Needs medical review.', `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `user_id` IS NULL AND `key` = 'mammography';

UPDATE `checkup_types` SET `source_note` = 'American Dental Association: a check-up every 6 months. Interval from the clinical registry key screening.uspstf (T-M10-21). Needs medical review.', `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `user_id` IS NULL AND `key` = 'dentist';

-- +goose Down

DELETE FROM `checkup_types` WHERE `user_id` IS NULL AND `key` IN ('bp_screening', 'glucose_screening');

UPDATE `checkup_types` SET `title` = '{"fa":"خودآزمایی سینه","en":"Breast self-exam"}', `subtitle` = '{"fa":"چند دقیقه در خانه، بعد از پریود","en":"A few minutes at home, after your period"}', `why` = '{"fa":"وقتی حالت طبیعی سینه‌هایت را بشناسی، هر تغییری را زودتر می‌بینی. بهترین زمان چند روز بعد از پریود است؛ سینه‌ها کمتر حساس و متورم‌اند.","en":"Knowing how your breasts normally feel helps you notice any change early. The best time is a few days after your period, when breasts are less tender and swollen."}', `informational` = 0, `source_note` = 'Default catalog (T-M4-01). Needs medical review before production.', `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `user_id` IS NULL AND `key` = 'breast_self_exam';

UPDATE `checkup_types` SET `why` = '{"fa":"پاپ‌اسمیر تغییرات سلول‌های دهانه رحم را پیش از آنکه به مشکل جدی تبدیل شوند نشان می‌دهد. برای بیشتر زنان ۲۱ تا ۶۵ ساله هر ۳ سال یک‌بار توصیه می‌شود؛ همراه با تست HPV می‌تواند هر ۵ سال شود.","en":"A Pap smear shows changes in cervical cells before they become a serious problem. For most women aged 21 to 65 it is recommended every 3 years; together with an HPV test it can be every 5 years."}', `source_note` = 'Default catalog (T-M4-01). Needs medical review before production.', `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `user_id` IS NULL AND `key` = 'pap_smear';

UPDATE `checkup_types` SET `why` = '{"fa":"ماموگرافی می‌تواند توده‌هایی را سال‌ها پیش از آنکه لمس شوند نشان دهد. برای بیشتر زنان از ۴۰ سالگی هر ۱ تا ۲ سال توصیه می‌شود؛ با سابقه خانوادگی ممکن است پزشک زودتر شروع کند.","en":"A mammogram can show lumps years before they can be felt. For most women it is recommended every 1 to 2 years from age 40; with a family history your doctor may start earlier."}', `interval_months` = 12, `interval_months_max` = 24, `age_max` = NULL, `source_note` = 'Default catalog (T-M4-01). Needs medical review before production.', `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `user_id` IS NULL AND `key` = 'mammography';

UPDATE `checkup_types` SET `source_note` = 'Default catalog (T-M4-01). Needs medical review before production.', `updated_at` = CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30')
WHERE `user_id` IS NULL AND `key` = 'dentist';
