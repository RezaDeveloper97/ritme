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

## C. New milestone copy (T-M10-22, D-173) — `needs_review = 1`, review before production

Goose `00102_child_milestones_cdc2022.sql` (Laravel twin `2026_10_23_000102_child_milestones_cdc2022.php`) adds the
CDC 2022 15- and 30-month bands (Zubler JM et al., *Pediatrics* 2022;149:e2021052138, «Learn the Signs. Act
Early.»), adapted for observation, not scoring. Every row is `needs_review = 1` in `catalog_items`; the advisor
marks them reviewed in the admin child-content screen (or edits the copy there). The 3-month band stays and is
marked `meta.band_source = ritme` (H-15); the app labels it «این مرحله را ریتمی اضافه کرده و در فهرست CDC ۲۰۲۲ نیست».
Growth (§3.5): crossing ≥ `child.growth.cross_lines` (2) major lines between two measurements → «با پزشک کودک در
میان بگذار» (API copy `growth.discuss` / `growth.crossing`, app copy `children.growth.crossingBody`).

| Band | Kind | Code | Domain | fa | en | Review |
|---|---|---|---|---|---|---|
| 15 mo | milestone | `m15_shows_object` | social | چیزی را که دوست دارد به تو نشان می‌دهد | Shows you an object they like | |
| 15 mo | milestone | `m15_tries_words` | language | غیر از ماما و بابا، یکی دو کلمه دیگر هم امتحان می‌کند | Tries to say one or two words besides mama or dada | |
| 15 mo | milestone | `m15_points_for_help` | language | برای خواستن چیزی یا کمک گرفتن اشاره می‌کند | Points to ask for something or to get help | |
| 15 mo | milestone | `m15_stacks_two` | cognitive | دست‌کم دو چیز کوچک، مثل مکعب، را روی هم می‌گذارد | Stacks at least two small objects, like blocks | |
| 15 mo | milestone | `m15_takes_steps` | motor | خودش چند قدم برمی‌دارد | Takes a few steps on their own | |
| 30 mo | milestone | `m30_look_at_me` | social | با گفتن «ببین!» کاری را که بلد است نشانت می‌دهد | Shows you what they can do by saying “Look at me!” | |
| 30 mo | milestone | `m30_says_50_words` | language | حدود ۵۰ کلمه می‌گوید | Says about 50 words | |
| 30 mo | milestone | `m30_action_phrase` | language | دو کلمه یا بیشتر را همراه یک فعل می‌گوید، مثل «هاپو دوید» | Says two or more words together, with one action word, like “doggie run” | |
| 30 mo | milestone | `m30_pretend_play` | cognitive | با چیزها بازی وانمودی می‌کند، مثلاً به عروسک مکعب می‌دهد که بخورد | Uses things to pretend, like feeding a block to a doll | |
| 30 mo | milestone | `m30_jumps_both_feet` | motor | با هر دو پا از زمین می‌پرد | Jumps off the ground with both feet | |
| 15 mo | play idea | `m15_build_knock` | — | **بساز و بریز** — دو سه مکعب یا لیوان را روی هم بگذار و بگذار او خرابشان کند. بعد نوبت خودش است که بسازد. | **Build and topple** — Stack two or three blocks or cups and let them knock them over. Then it’s their turn to build. | |
| 30 mo | play idea | `m30_jump_line` | — | **پرش از روی خط** — با نوار چسب روی زمین خطی بکش و با هم از رویش بپرید. هر پرش را بلند بشمار. | **Jump the line** — Put a strip of tape on the floor and jump over it together, counting each jump out loud. | |
| 15 mo | doctor note | `m15` | — | **مراحل رشد ۱۵ ماهگی** — اگر تا ۱۵ ماهگی هیچ کلمه‌ای نمی‌گوید، برای نشان دادن چیزی اشاره نمی‌کند یا مهارتی را که داشت از دست داده، در مراجعه بعدی با پزشک کودک در میان بگذار. | **15-month milestones** — If by 15 months they say no words, don’t point to show you things, or lose a skill they had, mention it to the paediatrician at the next visit. | |
| 30 mo | doctor note | `m30` | — | **مراحل رشد ۳۰ ماهگی** — اگر تا ۳۰ ماهگی دو کلمه را با هم نمی‌گوید، کارهای ساده را تقلید نمی‌کند یا مهارتی را که داشت از دست داده، با پزشک کودک صحبت کن. | **30-month milestones** — If by 30 months they don’t put two words together, don’t copy simple actions, or lose a skill they had, talk to the paediatrician. | |

## D. Pregnancy alert rules (T-M10-08, D-165) — seeded `message_rules`, review before production

Goose `00113_pregnancy_alert_rules.sql` (data only) seeds one approved rule per pregnancy alert in the `alerts_inbox`
slot (`rule_key = preg.<name>`), generated from `internal/messages/pregnancyalerts/inboxalerts` (`Defs` +
`copy/fa.json`, `copy/en.json`; the int test pins the rows to them). The copy is today's `pregnancy_alert` /
emergency copy (00005, 00008, T-M9-05) with the variables renamed (`{preg_*}`); rows marked **new** are new text.
The thresholds stay in the registry (section B and T-M9-05); a rule only reads the fact. Locked rules: only the
medical advisor edits them, and if a locked row is deleted, inactive or not approved the code-held definition with
the embedded fa/en copy still raises the alert (SPEC §4.2). Stored level `worth_followup` = follow_up, `follow_soon`
= urgent (contact line shown, safety urgent rows go to `urgent_modal`).

| Rule | Kind | Level | Locked | Condition | fa text (title — body · what we saw · how sure · contact) | Sign-off |
|---|---|---|---|---|---|---|
| `preg.vomiting` | clinical | worth_followup (follow_up) | no | any of: `preg.vomiting_days_7 ≥ 4` · `preg.vomiting_severe_7 ≥ 2` · `preg.vomiting_streak ≥ 3` | **استفراغ در {preg_vomit_days} روز از ۷ روز اخیر** — تهوع و استفراغ در این هفته‌ها رایجه. اگه نمی‌تونی مایعات رو نگه داری، ادرارت خیلی کم شده، یا وزن کم می‌کنی، بهتره با پزشک یا مامات در میون بذاری.<br>دیدیم: استفراغ در {preg_vomit_days} روز از ۷ روز اخیر، {preg_vomit_severe} روزش با شدت «شدید»<br>اطمینان: این الگو از ثبت‌های خودته، نه تشخیص | |
| `preg.cant_keep_fluids` | safety | follow_soon (urgent) | yes | `preg.cant_keep_fluids = true` — **no log field yet: the fact is always false, the rule never fires** (seeded so the advisor can sign the copy) | **مایعات در بدنت نمی‌ماند** — وقتی هیچ مایعی در بدنت نمی‌ماند، زود کم‌آب می‌شوی. همین امروز با پزشک یا ماما تماس بگیر.<br>دیدیم: ثبت کردی که نمی‌توانی مایعات را نگه داری.<br>اطمینان: این از ثبت خودت است؛ فقط پزشک می‌تواند علتش را بگوید.<br>تماس: همین حالا با پزشک یا ماما تماس بگیر؛ اگر در دسترس نیستند یا حالت بدتر شد، با اورژانس ۱۱۵ تماس بگیر. | |
| `preg.severe_symptoms` | clinical | worth_followup (follow_up) | no | `preg.severe_symptoms_7 ≥ 3` (severe (day, symptom) pairs of the 7 days: nausea, vomiting, fatigue, headache, back pain, breast pain, heartburn, constipation + the v1 fields dizziness, lower abdominal pain, cramping, pelvic pressure) | **{preg_severe_count} علامت شدید در یک هفته** — علائم شدیدِ تکرارشونده ارزش داره که در ویزیت بعدی مطرح بشه. اگه روزمره‌ات رو مختل کرده، زودتر با پزشکت صحبت کن.<br>دیدیم: در ۷ روز اخیر {preg_severe_count} بار علامتی با شدت «شدید» ثبت کردی<br>اطمینان: این الگو از ثبت‌های خودته، نه تشخیص | |
| `preg.critical_symptom` | safety | follow_soon (urgent) | yes | `preg.critical_symptom in [bleeding, fluid_leakage, severe_sudden_pain, spotting]` — logged in the last 7 days; spotting only up to week 12 or when severe | **{preg_symptoms} ثبت شده است** — این علامت را نادیده نگیر. همین حالا با پزشک یا ماما تماس بگیر.<br>دیدیم: «{preg_symptoms}» را ثبت کردی ({preg_symptom_on}).<br>اطمینان: این علامت از ثبت خودت است؛ فقط پزشک می‌تواند علتش را بگوید.<br>تماس: همین حالا با پزشک یا ماما تماس بگیر؛ اگر در دسترس نیستند یا حالت بدتر شد، با اورژانس ۱۱۵ تماس بگیر. | |
| `preg.weight_missing` | product | suggestion | no | `preg.days_since_weight ≥ 7` (since the pregnancy start when none was ever logged) | **وزن این هفته ثبت نشده** — اگه یک بار در هفته وزن رو ثبت کنی، روند وزن برای ویزیت‌هات کامل‌تر می‌شه. ثبت نکردنش هم اشکالی نداره.<br>دیدیم: در هفتهٔ {preg_week} هنوز وزنی ثبت نکردی<br>اطمینان: فقط یک یادآوره | |
| `preg.week_entered` | content | info | no | `pregnancy_week ≥ 1` (once a week: the week and its first day are the evidence) | **وارد هفتهٔ {preg_week} شدی** — مبنای محاسبه هنوز {preg_basis}ه. بعد از سونوگرافی NT می‌تونی مبنا رو به‌روز کنی تا بازهٔ تخمین محدودتر بشه.<br>دیدیم: هفتهٔ {preg_week} بارداری از {preg_week_on} شروع شده<br>اطمینان: بر اساس مبنای محاسبهٔ فعلی ({preg_basis}) | |
| `preg.next_visit` | content | info | no | `preg.next_visit_in_days ≤ 3` (**new**, SPEC §5.4.1) | **ویزیت بعدی‌ات نزدیک است** — اگه سؤالی داری یا علامتی هست که می‌خوای بپرسی، از همین حالا یادداشتش کن تا یادت نره.<br>دیدیم: ویزیت بعدی: {preg_visit_on}<br>اطمینان: از ویزیت‌هایی که خودت ثبت کردی | |
| `preg.fetal_movement` | safety | follow_soon (urgent) | yes | `preg.fetal_movement in [reduced, none]` — today's manual log from week `pregnancy.fetal_movement.manual_from_week` (24) | **حرکات جنین کم شده است** — کم‌شدن یا قطع حرکات جنین نباید تا فردا صبر کند. همین حالا با پزشک یا ماما تماس بگیر یا به بیمارستان برو.<br>دیدیم: حرکات جنین را «{preg_fetal_status}» ثبت کردی ({preg_fetal_on}).<br>اطمینان: این از ثبت خودت است؛ فقط معاینه می‌تواند حال جنین را مشخص کند.<br>تماس: همین حالا با پزشک یا ماما تماس بگیر؛ اگر در دسترس نیستند یا حالت بدتر شد، با اورژانس ۱۱۵ تماس بگیر. | |
| `preg.fetal_movement_kicks` | safety | follow_soon (urgent) | yes | `preg.fetal_movement = low_session` — a kick session short of `pregnancy.kicks` (10 in 120 min, from week 28) | **در {preg_kick_minutes} دقیقه کمتر از {preg_kick_target} حرکت شمردی** — همین حالا با پزشک یا ماما تماس بگیر. منتظر نمان که حرکت‌ها خودشان برگردند و شمارش را به فردا نینداز.<br>دیدیم: در شمارشِ {preg_kick_at}، {preg_kick_count} حرکت ثبت کردی؛ هدف، {preg_kick_target} حرکت در {preg_kick_minutes} دقیقه است.<br>اطمینان: این از شمارش خودت است و تشخیص نیست؛ فقط معاینه می‌تواند حال جنین را مشخص کند.<br>تماس: همین حالا با پزشک یا ماما تماس بگیر؛ اگر در دسترس نیستند یا حالت بدتر شد، با اورژانس ۱۱۵ تماس بگیر. | |
| `preg.bp_high` | clinical | worth_followup (follow_up) | no | `preg.bp_last_category = high` ∧ `preg.bp_high_level = follow_up` (≥ `pregnancy.bp.high` 140/90 while `pregnancy.bp.high.urgent = 0`) | **فشار خون بالا ثبت شده است** — فشار ۱۴۰/۹۰ یا بالاتر در بارداری باید زود بررسی شود. چند دقیقه استراحت کن، دوباره اندازه بگیر و امروز با پزشکت تماس بگیر.<br>دیدیم: فشار {preg_bp_sys}/{preg_bp_dia} ثبت کردی ({preg_bp_on}).<br>اطمینان: این یک عدد از ثبت خودت است؛ اندازه‌گیری دوباره در حال استراحت کمک می‌کند. | |
| `preg.bp_high_urgent` | clinical | follow_soon (urgent) | no | `preg.bp_last_category = high` ∧ `preg.bp_high_level = urgent` (≥ 140/90 while `pregnancy.bp.high.urgent = 1`, today's value, H-02) | **فشار خون بالا ثبت شده است** — فشار ۱۴۰/۹۰ یا بالاتر در بارداری باید زود بررسی شود. چند دقیقه استراحت کن، دوباره اندازه بگیر و با پزشکت تماس بگیر.<br>دیدیم: فشار {preg_bp_sys}/{preg_bp_dia} ثبت کردی ({preg_bp_on}).<br>اطمینان: این یک عدد از ثبت خودت است؛ اندازه‌گیری دوباره در حال استراحت کمک می‌کند.<br>تماس: همین حالا با پزشک یا ماما تماس بگیر؛ اگر در دسترس نیستند یا حالت بدتر شد، با اورژانس ۱۱۵ تماس بگیر. | |
| `preg.bp_severe` | safety | follow_soon (urgent) | yes | `preg.bp_last_category = severe` (≥ `pregnancy.bp.severe` 160/110; pregnancy logs and `/vitals` readings of the 7 days) | **فشار خونت در محدودهٔ شدید بارداری است** — همین حالا با پزشک یا ماما تماس بگیر یا به بیمارستان برو. اگر سردرد شدید، تاری دید، درد بالای شکم یا تنگی نفس داری، با اورژانس ۱۱۵ تماس بگیر.<br>دیدیم: فشار {preg_bp_sys}/{preg_bp_dia} ثبت کردی ({preg_bp_on})؛ ۱۶۰/۱۱۰ یا بالاتر در بارداری فوری است.<br>اطمینان: این یک عدد از ثبت خودت است؛ با این عدد منتظر اندازه‌گیری دوباره نمان.<br>تماس: همین حالا با پزشک یا ماما تماس بگیر؛ اگر در دسترس نیستند یا حالت بدتر شد، با اورژانس ۱۱۵ تماس بگیر. | |
| `preg.gdm` | clinical | worth_followup (follow_up) | no | `preg.gdm_last = above_target` (any value of the 7 days at or above `pregnancy.gdm.targets`: fasting 95, 1 h 140, 2 h 120) | **قند خون بالاتر از محدودهٔ معمول** — قند بالاتر از محدودهٔ معمول بارداری ارزش پیگیری داره. عددها رو در ویزیت بعدی به پزشکت نشون بده یا زودتر تماس بگیر.<br>دیدیم: قند ناشتا {preg_fasting} یا بعد از غذا {preg_post_meal} ثبت کردی ({preg_gdm_on})<br>اطمینان: این الگو از ثبت‌های خودته، نه تشخیص | |
| `preg.contractions_511` | safety | follow_soon (urgent) | yes | `preg.contractions_511 = true` (the running session meets `pregnancy.contractions_511`) | **انقباض‌ها به الگوی ۵-۱-۱ رسیده** — در {preg_labor_minutes} دقیقهٔ گذشته {preg_labor_count} انقباض ثبت کردی؛ فاصلهٔ میانگین {preg_labor_interval} و مدت میانگین {preg_labor_duration}. وقت تماس با زایشگاه یا مامای خودت است؛ وسایل بیمارستان را آماده کن. اگر کیسهٔ آب پاره شد، خون‌ریزی داری یا حرکات جنین کم شده، منتظر نمان.<br>دیدیم: انقباض‌های جلسهٔ {preg_labor_at} به الگوی ۵-۱-۱ رسیده‌اند.<br>اطمینان: این محاسبه از زمان‌هایی است که خودت ثبت کردی؛ فقط معاینه می‌تواند شروع زایمان را تأیید کند<br>تماس: اگر خون‌ریزی شدید، درد مداوم یا کم‌شدن حرکات جنین داری، همین حالا با اورژانس (۱۱۵) تماس بگیر. | |
| `preg.depression` | clinical | worth_followup (follow_up) | no | `preg.depression_severe = true` (weekly log of the 7 days: depression feelings «شدید»; the v1 weekly rule) | **این هفته حالت سخت بوده** — لازم نیست تنهایی تحملش کنی. حرف زدن با پزشک، ماما یا یک مشاور کمک می‌کنه. اگه فکر آسیب زدن به خودت به ذهنت می‌رسه، همین حالا با ۱۲۳ یا اورژانس ۱۱۵ تماس بگیر.<br>دیدیم: در ثبت هفتگی ({preg_mood_on}) احساس غم و افسردگی رو «شدید» ثبت کردی<br>اطمینان: این از ثبت خودته، نه تشخیص | |

New fa text to review (not in today's copy): `preg.cant_keep_fluids` (all), `preg.next_visit` (all), `preg.depression`
(all; the v1 rule had a short generic note), `preg.week_entered` «دیدیم» / «اطمینان» now carry the week's first day and
the dating basis.

Differences from the retired engines (parity table `inboxalerts/parity_test.go`; none lowers an escalation except
the one marked):

| Case | v1 (save-time) | v2 (detectors) | Now |
|---|---|---|---|
| Mild spotting after week 12 | info «لکه‌بینی خفیف» | — | — (**dropped**: an info note; SPEC §5.4.1 has no such rule — advisor to confirm) |
| Bleeding / fluid / pain / spotting ≤ 12 | warning or emergency | urgent (today only) | urgent, any log of the last 7 days |
| 3 severe symptoms | warning (one log, 10 fields) | follow_up (8 symptoms, 7 days) | follow_up, 12 symptoms, 7 days |
| Vomiting | — | streak ≥ 3 or 2 severe | + 4 of the last 7 days (SPEC §5.4.1) |
| Weight missing | — | day 5 of a week without weight | ≥ 7 days since the last weight (SPEC §5.4.1) |
| Next visit ≤ 3 days | — | — | info (**new**) |
| BP ≥ 140/90 | warning | level from `pregnancy.bp.high.urgent` | two rules on `preg.bp_high_level` (same) |
| Reduced / no fetal movement (≥ 24 w) | warning / emergency | urgent | urgent |
| Severe depression feelings | warning | — | follow_up |
| One BP alert at a time | one row per save | one row per reading | one open row per rule; the 7-day worst reading picks severe over high |

## E. Menopause message and condition nudge rules (T-M10-09, D-166) — seeded `message_rules`, review before production

Goose `00115_menopause_condition_rules.sql` (data only) seeds one approved rule per message of the two retired
per-request engines (D-43 menopause messages, D-41 condition nudges), generated from
`internal/messages/menomessages` and `internal/messages/conditionnudges` (`Defs` + `copy/fa.json`, `copy/en.json`; the
int test pins the rows to them). The copy is today's (menopause / nudge embedded copy, catalog `meno_alerts`
`postmenopausal_bleeding` and `meno_tips` `stage_*`) with the variables renamed (`{meno_*}`, `{cond_*}`); rows marked
**new** carry new text. `GET /messages/menopause` and `GET /messages/nudges` keep their texts (catalog /
`message_contents` / embedded copy); the rule copy below is what the alerts inbox (`/alerts`) and the feed render.
Thresholds that were code constants are now rule conditions (score rise ≥ 4, review within 14 days, 2 qualifying
days); the day rules (pain ≥ 7 or level severe, flow heavy / very heavy) and the 30-day bleeding window stay in code.

**Advisor flag — postmenopausal bleeding.** Today's behaviour is kept: an «alert, high» card, no modal, no
acknowledgement required → stored level `worth_followup` (follow_up), kind clinical, **locked** (a deleted or inactive
row still raises it from the code-held definition). The advisor should confirm follow_up, or raise it to urgent
(`follow_soon`, which would need kind safety and a contact line).

| Rule | Slot | Kind | Level | Locked | Condition | fa text (title — body · what we saw · how sure) | Sign-off |
|---|---|---|---|---|---|---|---|
| `meno.postmenopausal_bleeding` | alerts_inbox | clinical | worth_followup (follow_up) | yes | `meno.postmenopausal_bleeding = true` (bleeding / spotting in the last 30 days at stage meno or post) | **خونریزی بعد از یائسگی** — هر خونریزی یا لکه‌بینی، حتی کم، بعد از ۱۲ ماه قطع پریود باید بررسی شود. بیشتر وقت‌ها علت ساده‌ای دارد، اما بررسی زودهنگام مهم است.<br>دیدیم (**new**): خونریزی یا لکه‌بینی ثبت کردی؛ آخرین بار {meno_bleeding_on}.<br>اطمینان (**new**): این از ثبت خودت است، نه تشخیص؛ فقط معاینه می‌تواند علتش را بگوید. | |
| `meno.score_worsened` | alerts_inbox | clinical | worth_followup (follow_up) | no | `meno.score_rise ≥ 4` (newest questionnaire of this or the previous Jalali month vs the one before) | **امتیاز علائمت بیشتر شده** — امتیاز آخرت {meno_score_total} است؛ {meno_score_rise} واحد بیشتر از دفعه قبل ({meno_score_previous}). ببین چه چیزی تغییر کرده و اگر بالا ماند با پزشکت حرف بزن.<br>دیدیم (**new**): امتیاز ماهانه‌ات از {meno_score_previous} به {meno_score_total} رسید.<br>اطمینان (**new**): این از پرسش‌نامه‌ای است که خودت پر کردی، نه تشخیص. | |
| `meno.checkup_overdue` | alerts_inbox | content | suggestion | no | `meno.checkups_overdue ≥ 1` | **یک چکاپ عقب افتاده** — زمان {meno_overdue_checkup} گذشته است. اگر زودتر نوبت بگیری، برنامه سلامتت مرتب می‌ماند. | |
| `meno.hrt_review` | alerts_inbox | content | suggestion | no | `meno.hrt_review_in_days between 0 and 14` (nearest review of an active HRT item; the GET matches each item) | **بازبینی درمان نزدیک است** — بازبینی {meno_hrt_name} با پزشکت ({meno_hrt_on}) نزدیک است. حال این مدت و عوارض احتمالی را یادداشت کن تا با خودت ببری. (the relative «{when}:» lead-in of today's copy dropped: the inbox stores language-free evidence) | |
| `meno.checkup_due` | alerts_inbox | content | info | no | `meno.checkups_due ≥ 1` ∧ `meno.checkups_overdue = 0` | **وقت یک چکاپ رسیده** — زمان {meno_due_checkup} رسیده است. بعد از یائسگی چکاپ منظم برای استخوان و قلب اهمیت بیشتری دارد. | |
| `meno.tip_stage_peri` / `_meno` / `_post` / `_unsure` | daily_advice | content | info | no | `meno.stage = peri / meno / post / unsure` | the catalog `meno_tips` `stage_*` title and body (unchanged) | |
| `cond.heavy_pain` | alerts_inbox | content | suggestion | no | `cond.heavy_pain_days ≥ 2` ∧ `cond.programs nin [endo]` (cycle, ttc, teen; current cycle) | **درد شدید در چند روز** — در این چرخه {cond_pain_days} روز درد ۷ یا بیشتر ثبت کرده‌ای. دفترچه درد کمک می‌کند تو و پزشکت الگوی آن را ببینید. | |
| `cond.heavy_bleeding` | alerts_inbox | content | suggestion | no | `cond.heavy_bleeding_days ≥ 2` ∧ `cond.programs nin [heavy_bleeding]` | **خونریزی سنگین در چند روز** — در این چرخه {cond_bleeding_days} روز خونریزی سنگین ثبت کرده‌ای. جدول نوار بهداشتی (PBAC) مقدار خونریزی را می‌سنجد تا بشود بررسی‌اش کرد. | |

Differences from the retired engines (parity tables `menomessages/parity_test.go`, `conditionnudges/parity_test.go`):
none in what is raised. New: one inbox row per rule (a repeated bleeding signal on a new day merges, count + 1), rows
resolve when the pattern ends or she leaves the mode, and the admin can switch a rule off or change its threshold
(except the locked bleeding rule's code-held fallback).
