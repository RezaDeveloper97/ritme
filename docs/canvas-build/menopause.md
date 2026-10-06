# Menopause mode — data model (CB-MENO-01)

Shared reference for the `CB-MENO-*` tasks (roadmap/E02-meno). Boards: `nbl_Meno_*` + `Main` (text digest
`docs/design/canvas-v1/text/meno.txt`). Built **on top of bloom** (roadmap/DECISIONS.md #2): nothing here re-implements a
`B-N*` deliverable. All clinical/list copy is admin-editable catalog content, fa + en, `needs_review = 1`
(`[needs clinical review]`, DECISIONS #7).

## 1. Where each piece lives

| Data | Lives in | Owner |
|---|---|---|
| Mode = menopause | `user_life_profiles.life_mode` | bloom B-N2-01 / B-N2-03 |
| Stage peri · meno · post · unsure, approx. last period (first of the month), surgical yes/no, HRT yes/no | `user_life_profiles.menopause_stage / menopause_last_period / menopause_surgical / menopause_hrt` (asked by Onb_Meno, B-N2-02; `GET|PUT /profile/life-stage`) | bloom B-N2-01 — **no `menopause_profiles` table** (README C4) |
| Daily symptoms (13), bleeding none/spot/bleed, triggers | log taxonomy v2 slots in `health_log_entries` (`GET /logs/taxonomy?mode=menopause`, `PUT /logs/days/{date}`) | bloom B-N3-01, items added here (§3) |
| Hot flashes (timer) | `hot_flashes` | CB-MENO-01 (API CB-MENO-02) |
| Monthly score | `menopause_scores` + catalog `meno_score_items` / `meno_score_bands` | CB-MENO-01 (API CB-MENO-02) |
| HRT / supplements / lifestyle, intakes, side effects | `treatment_items`, `treatment_intakes`, `side_effect_logs` (+ care `reminders`) | CB-MENO-01 (API CB-MENO-03) |
| Alerts, tips, notes | catalog `meno_alerts`, `meno_tips` | CB-MENO-01 (admin CB-MENO-04, engine CB-MENO-12) |
| Checkups | M4 `checkup_types` rows with `audiences = ["menopause"]` + catalog `meno_checkup_groups` | CB-MENO-01 (UI CB-MENO-09) |

Migrations: goose `00022_menopause.sql` + Laravel twin `2026_10_01_000022_create_menopause_tables.php` (00021 is
bloom's). Queries: `db/queries/menopause/menopause.sql` → `internal/menopause/store`.

## 2. Stage rule

`menopause_last_period` is the approximate last period (first day of that month). Let *m* = whole months from it to
today (Tehran).

| Stored stage | Shown |
|---|---|
| `peri` | perimenopause (periods still come, irregular) — if *m* ≥ 12 the API suggests «meno» but never changes the stored answer |
| `meno` | **12 months or more without a period ⇒ menopause**; *m* drives «بدون پریود N ماه» |
| `post` | postmenopause (several years after the last period) |
| `unsure` / NULL | derived: *m* ≥ 12 → meno, else peri; no last period → ask (stage screen) |

`menopause_surgical = 1` (ovaries removed / treatment-induced) counts as menopause from the surgery date regardless
of *m*. From stage meno or post on, **any bleeding or spotting** (taxonomy `bleeding.presence` ≠ none, or
`bleeding.flow`, or `bleeding.spotting` = yes) raises the `postmenopausal_bleeding` alert (CB-MENO-02 today flag,
CB-MENO-12 message). The rule is computed in the API (CB-MENO-02), never stored.

## 3. Daily log = taxonomy slots (B-N3-01)

The board's groups are a preset (`taxonomy.MenopausePreset()`, labels `log-taxonomy` → `presets.menopause.<group>`).
Symptom rows use the levels `no · mild · moderate · severe` (joints: pain levels; «ندارم» = no entry).

| Group | Slot (`category.param.item`) | New in CB-MENO-01? |
|---|---|---|
| vasomotor — گرگرفتگی و قلب | `symptoms.general.hot_flashes`, `symptoms.general.night_sweats`, `symptoms.general.palpitations` | no (bloom; last two menopause-only) |
| sleep — خواب | `symptoms.general.insomnia` | no |
| mind — حال و ذهن | `symptoms.general.anxiety`, `.irritability`, `.low_mood` (menopause-only), `symptoms.general.brain_fog` | anxiety/irritability/low_mood |
| body — بدن | `pain.location.joints`, `symptoms.general.fatigue` | no |
| urogenital — ادراری و تناسلی | `urogenital.symptoms.vaginal_dryness`, `.bladder_symptoms`, `.low_libido` (last two menopause-only) | bladder_symptoms, low_libido |
| bleeding | `bleeding.presence` = none \| spotting \| bleeding (single, menopause-only, `alert`) | yes |
| triggers — امروز چه چیزهایی داشتی؟ | `menopause.triggers` (multi): caffeine, spicy_food, stress, exercise, warm_room, hot_drink — new mode category `menopause` (group `mode`) | yes |

Adding or renaming an item is a code change in `internal/healthlog/taxonomy` plus labels in
`resources/translations/<code>/log-taxonomy.json` (frontend copy byte-identical, `internal/i18n/testdata` re-synced).

## 4. Tables

| Table | Columns | Notes |
|---|---|---|
| `hot_flashes` | `user_id`, `started_at` datetime, `duration_s` (NULL = timer running), `severity` 1–4 (mild, moderate, severe, very severe; NULL until set), `night`, `sweat`, `triggers` JSON list | trigger codes = `menopause.triggers` options + `unknown` (board chips قهوه · استرس · غذای تند · گرما · نمی‌دانم → caffeine, stress, spicy_food, warm_room, unknown). Index (user_id, started_at). |
| `menopause_scores` | `user_id`, `month` date, `answers` JSON `{meno_score_items code: 0–4}`, `total` /44, `somatic` /16, `psychological` /16, `urogenital` /12 | `month` = first day of the Jalali month; unique (user_id, month) — refilling replaces. Subtotals stored by the API from the items' `meta.domain`. |
| `treatment_items` | `kind` hrt \| supplement \| lifestyle, `name`, `dose` (free text), `schedule` morning \| noon \| evening \| night \| weekly, `started_on`, `review_on`, `weekly_goal` + `goal_unit` sessions \| minutes, `stopped_on` (NULL = active), `reminder_id` → `reminders` (SET NULL), `sort_order` | names/doses are what the user types — never prescribed by the app. |
| `treatment_intakes` | `treatment_item_id` (cascade), `intake_date`, `amount` (minutes/sessions for lifestyle), `taken_at` | unique (item, day): weekly dots + adherence %. |
| `side_effect_logs` | `log_date`, `code` breast_tenderness \| spotting \| headache \| bloating \| mood_change, optional `treatment_item_id` (SET NULL) | unique (user, day, code); code labels are UI strings (CB-MENO-10). |

All user-scoped, FK cascade on account deletion, no analytics. `checkup_types.audiences` JSON (NULL = everyone)
scopes M4 catalog rows to life modes.

## 5. Catalog groups (`catalog_items`, audiences `["menopause"]`, needs_review = 1)

| Group | Items | `meta` |
|---|---|---|
| `meno_score_items` | 11 questions (hot_flashes, heart_discomfort, sleep_problems, joint_muscle · depressive_mood, irritability, anxiety, exhaustion · sexual_problems, bladder_problems, vaginal_dryness); title = question, body = hint | `{domain: somatic\|psychological\|urogenital, max: 4, log: ["category.param.item", …]}` — `log` links the related taxonomy slots (patterns, report) |
| `meno_score_bands` | none 0–4 · mild 5–8 · moderate 9–16 · severe 17–44 | `{min, max}` on the total |
| `meno_alerts` | postmenopausal_bleeding (primary), heavy_perimenopause_bleeding, chest_pain_palpitations (115), one_sided_leg_swelling, breast_change, persistent_low_mood, fragility_fracture | `{severity: caution\|urgent, primary?, emergency?, hotline?, hrt?, stages?: [stage codes], cta?: {lang: text}}` |
| `meno_tips` | stage_peri/meno/post/unsure, log_bleeding_note, hot_flash_breathing, hrt_review, hrt_spotting, doctor_only, lifestyle_resistance/brisk_walk/relaxation, checkups_intro, patterns_disclaimer | `{placement: home\|log\|hot_flash\|treatment\|treatment_lifestyle\|checkups\|score, stages?, weekly_goal?, goal_unit?, review_after_months?}` |
| `meno_checkup_groups` | heart_metabolic, bone, cancer_screening, other (board sections) | `{checkups: [checkup_types keys]}` |

`meno_score_bands` and `meno_checkup_groups` are two small groups beyond the task's list: the band thresholds and the
board's checkup sections are clinical/list content, so they are admin-editable rather than code.

## 6. Checkups (M4)

Nine `checkup_types` rows with `audiences = ["menopause"]`, `hide_in_pregnancy = 1`, **`is_active = 0`**: blood
pressure, fasting sugar/HbA1c, lipids, weight & waist, bone density (from 65), vitamin D & calcium, bowel screening
(from 45), thyroid, eye exam. Existing rows mammography, pap_smear and dentist stay for everyone and appear in the
menopause groups. The board's «معاینه چشم و دندان» is split into `meno_eye_exam` + the existing `dentist`.

**CB-MENO-01b (done):** the checkups user API (`internal/checkups`: plan, home card, detail, records, settings, and
search's checkup source) shows a shared row only when `audiences` is NULL or lists the user's resolved life mode
(`enums.ResolveLifeMode`: active pregnancy → pregnancy, stored postpartum/menopause/teen, else ttc/cycle by
`user_goal`); outside the audience a type id answers 404. The admin checkup-types API reads/writes `audiences`
(validated against the life modes, `options.audiences`); the admin-web form field is CB-MENO-04. Goose
`00024_activate_menopause_checkups` sets the nine rows `is_active = 1` (data only, no Laravel twin).

## 7. API (CB-MENO-02)

Go-only, `/api/v1/menopause/*` (deviation D-42, OpenAPI tag `Menopause`, contract group `menopause`), package
`internal/menopause`. All rules are named constants there and [needs clinical review].

| Route | What |
|---|---|
| `GET\|PUT /menopause/profile` | §2 stage rule on the life-profile columns; PUT writes only the four menopause columns (partial, null clears) |
| `GET /menopause/today` | home: flashes today + running timer, last night's sweats ([yesterday 22:00, 06:00) sweaty flashes, else the log), sleep (today's log, else yesterday), latest score + 6-month trend, ≤ 3 upcoming checkups of `meno_checkup_groups`, active treatment with last-7-days adherence, bleeding flag (30 days, stage meno/post) |
| `GET /menopause/hot-flashes?date=` · `POST /menopause/hot-flashes` · `POST /menopause/hot-flashes/{id}/stop` | timer: start (running one returned, forgotten ≥ 1 h closed at 3600 s), log a finished flash (`duration_s`, ≤ 7 days back), stop/edit; night by default 22:00–06:00 |
| `GET\|POST /menopause/scores` | one questionnaire per Jalali month (refill replaces), total/domains/band/delta, history `?months=` (1–24) with the HRT annotation |
| `GET /menopause/patterns` | 90-day φ cards via bloom's `analysis.Binary` (min 20 days / 5 per group / 3 outcomes): each logged trigger × more flashes than her median day, night sweats × fatigue; `not_a_diagnosis`, sentences in `internal/menopause/lang` (needs review) |

### CB-MENO-03 — treatment & care, doctor report section

| Route | What |
|---|---|
| `GET /menopause/treatment?date=` | Meno_Treatment for the Saturday week of `date`: items by kind (+ `stopped`), week dots, days taken / scheduled days, adherence %, lifestyle goal progress, care reminder; next review (earliest HRT `review_on`, else start + `meno_tips` hrt_review `review_after_months`, default 3, `suggested`); the week's side effects; `treatment` / `treatment_lifestyle` tips |
| `POST /menopause/treatment/items` · `PUT\|DELETE …/items/{id}` | hrt / supplement need a schedule (morning 08:00 · noon 13:00 · evening 18:00 · night 22:00 · weekly 09:00 on the start weekday) and **are a care medication reminder** (`reminder_id`; `remind` = notify; care's 100 cap); lifestyle = weekly goal (sessions ≤ 21 / minutes ≤ 3000), no reminder. `stopped_on` stops it (reminder off). Cap 30 items. Delete removes the care reminder too |
| `PUT\|DELETE …/items/{id}/intakes/{date}` | one intake per (item, day), ≤ 30 days back, active day only; lifestyle `amount` (minutes required / sessions optional). Ticks / unticks the care dose; a /care tick counts here too |
| `PUT /menopause/treatment/side-effects/{date}` | the day's codes as a set (optional own `treatment_item_id`) |
| `GET /menopause/report?months=1\|3\|6` | owner preview of the report; the same data is the provider section `menopause` of bloom's builder (`GET /health-record/report?sections=…,menopause`, share links): stage, score first → last, flashes/day and night-sweat nights/week over tracked days (a flash or a log entry), sleep avg, bleeding events (runs of days), BP avg (merged vitals), top 5 symptoms % of tracked days, per-item adherence + pooled HRT %, lifestyle per week, side effects, supplements. Menopause mode only, report-only (not on the summary), no ids / notes / loss data. The questions are the builder's `question` |
