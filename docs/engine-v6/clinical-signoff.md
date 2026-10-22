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
| H-19 | شمارش لگد که با کمتر از ۱۰ حرکت تمام شود (حتی شروع و توقف فوری با ۰ حرکت) الان هشدار فوری می‌دهد. حداقل مدت شمارش لازم است؟ | any session ending under target → urgent (T-M9-05) | spec: < 10 in 2 h → urgent; minimum duration unspecified | ACOG kick count; SPEC §3.1 | new key (e.g. `pregnancy.kicks.min_session_min`) if the advisor wants one | | advisor-pending |
| H-20 | تب بعد از زایمان از چند درجه هشدار فوری (۱۱۵) بدهد؟ | 38.0 °C → urgent (T-M9-06) | SPEC §3.1: ≥ 38.0 | SPEC §3.1; ACOG CO 736 | `postpartum.redflags.fever_c` = 38.0 | | advisor-pending |
| H-21 | ویزیت‌های بعد از زایمان: تماس اول تا روز ۲۱، یادآوری روز ۴۲، ویزیت جامع تا روز ۸۴ — درست است؟ «درد شدید» در طبقه‌بندی فعلی (`pain_level = severe`) هم فوری باشد؟ | 21 / 42 / 84 days; only the new `severe_pain` flag is urgent | same (ACOG CO 736) | ACOG CO 736 | `postpartum.visits` = 21/42/84 | | advisor-pending |

## B. New registry keys (T-M10-16) — no behaviour change today, signature needed when changed

Version 1 of every key below is code-held (`backend-go/internal/clinicalconfig/coverage.go`) and pinned by
`TestCoverageKeys_PinnedToTheirOrigin` to the constant / seed it replaces or to the SPEC v6 table. Nothing reads
these keys until T-M10-18 … T-M10-22 switch the consumers. Rows keys are tables edited in the clinical-config screen.

| Key | Group | v1 value | Replaces / origin | Source (v1 source text) |
|---|---|---|---|---|
| `pregnancy.edd` | pregnancy | `naegele_days 280` | `pregnancy/calc/calc.go:173` (LMP + 280) | Naegele; ACOG CO 700 (2017); SPEC §3.1 |
| `pregnancy.edd.cycle_correction` | pregnancy | `enabled 0` | none today (SPEC §3.1: optional, off) | ACOG CO 700; SPEC §3.1 |
| `pregnancy.trimester` | pregnancy | `t2_from_days 98, t3_from_days 196` | **SPEC value**; today `calc.go:132-139` starts T2 at 13 completed weeks (day 91) — T-M10-18 switches | ACOG CO 700; SPEC §3.1 |
| `pregnancy.redating` (rows `ga_until_days, max_diff_days`) | pregnancy | (62, 5), (97, 7), (111, 7), (153, 10), (195, 14), (open, 21) | SPEC §3.1 table (nothing reads it today) | ACOG CO 700 Table 1: re-date when the LMP–US difference is **more than** the limit; the band is the **GA by LMP on the scan day** (table footnote "based on LMP"), which is what SPEC §3.1 «سن بارداری هنگام سونو» means |
| `pregnancy.iom` (rows `bmi_from, bmi_to, total_lo, total_hi, weekly_lo, weekly_hi`) | pregnancy | [0, 18.5) 12.5–18 kg, 0.44–0.58 kg/wk · [18.5, 25) 11.5–16, 0.35–0.50 · [25, 30) 7–11.5, 0.23–0.33 · [30, open) 5–9, 0.17–0.27 | totals/bands = `analysis/pregnancy_thresholds.go:19-24`; weekly rates = IOM 2009 Table S-1 (not in code or SPEC before) | IOM 2009 Table S-1; SPEC §3.1 |
| `pregnancy.iom.first_trimester` | pregnancy | `lo 0.5, hi 2` | `analysis/pregnancy_thresholds.go:28-29` | IOM 2009 |
| `pregnancy.spotting` | pregnancy | `until_week 12` | `messages/pregnancyalerts/rules.go:289` + seed `00005_pregnancy_v2.sql:236-237` | Ritme alerts v1 (**SPEC §3.1 has no week limit**: any bleeding → discuss) |
| `epds.schedule` | pregnancy | `full_days 14, short_days 7` | `postpartum/epds.go:38-39` | Cox 1987 (every 2 weeks, SPEC §3.1); Ritme short check-in |
| `lab.hb` | lab | `nonpregnant_low 12, pregnant_low 11, pregnant_t2_low 10.5` g/dL | SPEC §3.3 (12 = hemoglobin catalog `typical.low`, `00034_labs.sql:126`) | WHO haemoglobin 2024 |
| `lab.ferritin.inflamed` | lab | `low 70` ng/mL | SPEC §3.3 (`lab.ferritin.low 15` stays) | WHO ferritin 2020 |
| `lab.vitd` | lab | `deficient_below 12, sufficient_from 20` ng/mL | SPEC §3.3 (catalog `typical.low` is 30) | IOM 2011 |
| `lab.tsh.pregnancy` | lab | `upper 4.0` mIU/L | SPEC §3.3 | ATA 2017 |
| `lab.hba1c` | lab | `prediabetes_from 5.7, diabetes_from 6.5` % | SPEC §3.2 | ADA 2025 §2 |
| `lab.ocr` | lab | `min_confidence 0.9` (H-08, user-accepted, ships) | SPEC §3.3; today `labs/labs.go:89` = 0.7 — T-M10-20 switches | Ritme product rule (H-08) |
| `lab.borderline` | lab | `enabled 1, margin 0.10` (H-07 safe default) | `labs/status.go:28` | Ritme lab statuses v1 |
| `bmi.who` | vitals | `underweight_below 18.5, overweight_from 25, obese_from 30` | `enums/health_logic.go:27-31`, `profile/bmi.go:84-89` (boundary → higher band) | WHO BMI; SPEC §3.2 |
| `glucose.contexts` | vitals | `spec 0` (H-09 safe default = today's 140/200 bands) | today's behaviour | H-09; ADA 2025 §2, §6 |
| `glucose.diabetic` | vitals | `fasting_lo 80, fasting_hi 130, post_meal_max 180` mg/dL | SPEC §3.2 (ADA "preprandial" = fasting / pre-meal) | ADA 2025 §6 |
| `child.growth` | child | `low_pct 3, high_pct 97, cross_lines 2` | `children/growth/growth.go:24-25`; `cross_lines` SPEC §3.5 | WHO Child Growth Standards |
| `child.vaccine_schedule` (rows `vaccine_code, dose, age_months`) | child | 21 rows = the `child_vaccines` seed (`00031_children.sql:100-204`): bcg, hepb_1, opv_0 @0 · penta_1, polio_m2, pcv_1, rota_1 @2 · penta_2, opv_m4, pcv_2, rota_2 @4 · penta_3, polio_m6, pcv_3 @6 · mmr_1, pcv_booster @12 · dtp_booster_1, opv_booster_1, mmr_2 @18 · dtp_booster_2, opv_booster_2 @72 months; `vaccine_code` = catalog code, `dose` = ordinal within the vaccine | seed | وزارت بهداشت — برنامهٔ کشوری ایمن‌سازی; version in force on the birth date (H-17) |
| `screening.uspstf` (rows `type_key, age_min, age_max, interval_months, last_test_kind, min_bmi`) | screening | pap_smear 21–29 / 36 cytology · pap_smear 30–65 / 36 cytology, 60 hpv, 60 cotest · mammography 40–74 / 24 · bp_screening 18–39 / 36 · bp_screening ≥ 40 / 12 · glucose_screening 35–70, BMI ≥ 25 / 36 · dentist any age / 6 | SPEC §3.4 table (nothing reads it until T-M10-21) | USPSTF 2018 / 2021 / 2024; ADA (dental) |
