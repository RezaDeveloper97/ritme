# Engine v6 — clinical sign-off sheet (advisor)

Written by T-M10-01 on 2026-10-21. Every row below ships **now** with the safe default in "Today's value" and is
recorded as `advisor-pending` in `DECISIONS.md` (section H). Nothing waits for the advisor: when the advisor decides,
an admin proposes the new value in the clinical-config screen (two-advisor approval) — no code change.

The same rows are appended to the shared review doc «بازبینی پزشکی محتوای ریتمی» (link in `tasks/PROGRESS.md`,
T-M4-10 entry) so the advisor reviews everything in one place.

Sources are the SPEC §6.2 references.

## A. Open clinical decisions

| Id | پرسش برای مشاور | Today's value (ships) | Spec / recommended | Source | Registry key → value that switches | Advisor decision | Status |
|---|---|---|---|---|---|---|---|
| H-01 | اگر جمع نمرهٔ EPDS ۱۳ یا بیشتر باشد ولی سؤال ۱۰ (فکر آسیب به خود) صفر باشد، صفحهٔ بحران (۱۲۳/۱۱۵) باز شود یا فقط «زودتر با پزشکت صحبت کن»؟ | urgent at total ≥ 13 | not urgent; only Q10 > 0 is urgent | Cox 1987 (EPDS); ACOG CO 736 | `epds.urgent_total` = 13 → 31 (off) | | advisor-pending |
| H-02 | فشار ۱۴۰/۹۰ تا ۱۵۹/۱۰۹ در بارداری: فوری (۱۱۵) یا «امروز با پزشکت تماس بگیر»؟ (۱۶۰/۱۱۰ به بالا در هر حال فوری است) | urgent (`1`) | follow_up | ACOG PB 222 | `pregnancy.bp.high.urgent` = 1 → 0 | | advisor-pending |
| H-04 | هشدار «حرکت جنین کم یا هیچ» که خودِ کاربر ثبت می‌کند از هفتهٔ ۲۴ باشد یا ۲۸؟ («هیچ حرکت» در هفتهٔ ۲۴ تا ۲۷ هشدار بماند؟) | manual from week 24; kick-counter auto alert from week 28 | 28 for both | SPEC §3.1 | `pregnancy.fetal_movement.manual_from_week` = 24 → 28 | | advisor-pending |
| H-06 | قاعدهٔ ۵-۱-۱ انقباض: طول هر انقباض ۴۵ ثانیه یا ۱ دقیقه؟ | 45 s (more sensitive) | 60 s | SPEC §3.1 | `pregnancy.contractions_511.duration_s` = 45 → 60 | | advisor-pending |
| H-07 | دستهٔ «مرزی» (±۱۰٪ حد نرمال) در تفسیر آزمایش بماند یا حذف شود (حداکثر یک اشارهٔ دیداری زیر «خارج از محدوده»)؟ | kept (`enabled 1, margin 0.10`) | remove the class | SPEC §3.3 | `lab.borderline.enabled` = 1 → 0 | | advisor-pending |
| H-09 | قند بعد از غذا، تصادفی و قبل از خواب در گلوکومتر: باندهای فعلی ۱۴۰/۲۰۰ برای هر سه، یا قاعدهٔ اسپک (بعد از غذا < ۱۴۰ در محدوده، وگرنه «با پزشک در میان بگذار»؛ تصادفی و قبل از خواب زیر ۲۰۰ بدون دسته)؟ | `spec 0` (140/200 bands for all three) | `spec 1` | ADA Standards of Care 2025 §2, §6 | `glucose.contexts.spec` = 0 → 1 | | advisor-pending |
| H-10 | خودآزمایی پستان به یک آیتم اطلاعاتی «آگاهی از سینه» بدون موعد تبدیل شود و تاریخچه‌اش فقط‌خواندنی بماند؟ | dated checkup with reminders | informational, no due date | ACOG PB 179; USPSTF breast 2024 | catalog change in T-M10-21 (type `breast_self_exam`) | | advisor-pending (user+advisor) |
| H-11 | پاپ‌اسمیر/HPV در ۳۰ تا ۶۵ سال: یک نوع چکاپ با فاصلهٔ وابسته به نوع آخرین تست (سیتولوژی ۳۶ ماه، HPV یا هم‌آزمایی ۶۰ ماه)؟ | catalog interval | interval by last test kind | USPSTF cervical 2018 | `screening.uspstf` rows (cervical 30–65: 36 cytology / 60 HPV) | | advisor-pending |
| H-14 | بعد از زایمان، ثبت خون‌ریزی به‌عنوان پریود از روز ۴۲ باز شود (قبلش لوشیا حساب می‌شود)؟ | 42 days | 42 days (I-05) | ACOG CO 736 | `postpartum.flow_from_day` = 42 | | advisor-pending |
| H-15 | باند رشد «۳ ماهگی» (در CDC 2022 نیست) بماند یا حذف شود؟ | kept, labelled «باند خود ریتمی، نه CDC» | keep or drop | Zubler 2022 (CDC milestones) | milestone catalog (T-M10-22) | | advisor-pending |
| H-16 | چکاپ‌های خارج از اسپک (معاینهٔ بالینی پستان، آزمایش خون سالانه): بمانند، اختیاری شوند یا حذف؟ | kept unchanged | keep / optional / retire | USPSTF breast 2024 | checkup catalog (T-M10-21) | | advisor-pending |
| H-17 | کودک از کدام نسخهٔ جدول واکسن پیروی کند؟ نسخهٔ معتبر در تاریخ تولدش؛ جبران (catch-up) دستی بماند؟ | version in force on the birth date; catch-up manual | same | وزارت بهداشت — برنامهٔ کشوری ایمن‌سازی | `child.vaccine_schedule` rows | | advisor-pending |

## B. New registry keys (T-M10-16) — no behaviour change today, signature needed when changed

| Key | v1 value | Source |
|---|---|---|
| `pregnancy.edd` | `naegele_days 280` | ACOG CO 700 |
| `pregnancy.edd.cycle_correction` | `enabled 0` | ACOG CO 700 |
| `pregnancy.trimester` | `t2_from_days 98, t3_from_days 196` | ACOG CO 700 |
| `pregnancy.redating` (rows) | (62, 5), (97, 7), (111, 7), (153, 10), (195, 14), (open, 21) | ACOG CO 700 |
| `pregnancy.iom` (rows) + `pregnancy.iom.first_trimester` | today's `analysis/pregnancy_thresholds.go` values; `lo 0.5, hi 2` | IOM 2009 |
| `pregnancy.spotting` | `until_week 12` | SPEC §3.1 |
| `epds.schedule` | `full_days 14, short_days 7` | Cox 1987; ACOG CO 736 |
| `lab.hb` | `nonpregnant_low 12, pregnant_low 11, pregnant_t2_low 10.5` | WHO haemoglobin 2024 |
| `lab.ferritin.inflamed` | `low 70` | WHO ferritin 2020 |
| `lab.vitd` | `deficient_below 12, sufficient_from 20` | IOM 2011 |
| `lab.tsh.pregnancy` | `upper 4.0` | ATA 2017 |
| `lab.hba1c` | `prediabetes_from 5.7, diabetes_from 6.5` | ADA 2025 §2 |
| `lab.ocr` | `min_confidence 0.9` (H-08, user-accepted) | — (product) |
| `bmi.who` | `underweight_below 18.5, overweight_from 25, obese_from 30` | WHO BMI |
| `glucose.diabetic` | `fasting_lo 80, fasting_hi 130, post_meal_max 180` | ADA 2025 §6, §15 |
| `child.growth` | `low_pct 3, high_pct 97, cross_lines 2` | WHO Child Growth Standards |
| `screening.uspstf` (rows) | SPEC §3.4 table | USPSTF 2018/2021/2024 |
| `child.vaccine_schedule` (rows) | today's `child_vaccines` seed ages | وزارت بهداشت |
