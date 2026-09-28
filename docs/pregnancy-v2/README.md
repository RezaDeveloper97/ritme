# Pregnancy mode v2 (M7) — redesign, admin-defined content, message-engine integration

Source design: Design canvas «ریتمی — ماژول بارداری» (claude.ai artifact `19aCpFNXeaJbWh8g48KswX`), row
**«نسخه ۲ — بازبینی UI/UX بر پایهٔ کد و اسناد ریتمی»** (`project/v2/*`). Local copies: [`docs/design/pregnancy-v2/`](../design/pregnancy-v2/).
The artboards are interactive (`sc-for`/`sc-if` + a small script): read the `renderVals()` script too — it holds
the option lists, labels and state styles.

**There is no dark artboard for this module.** Dark mode = the app's implemented dark theme: every design color maps
to an existing token (both `:root` and `[data-theme="dark"]`), new pairs only where nothing fits, `lint:dark` green.
Design palette → tokens: `#F2ECFF` page · `#FFFFFF` surface · `#E7E1F4`/`#E3DDF2` line · `#F4F0FF`/`#E6DEFF` soft
surface / segmented track · `#2F2F35` ink · `#4E4E59` muted · `#7B61FF` brand · `#6A4FE8`/`#5B41E6` brand-deep ·
`linear-gradient(135deg,#7B61FF,#FF6FAE)` = the brand gradient token · `#FFD9E9`/`#B02A63` pink soft/deep ·
`#E3F9FE`/`#0C6F89` teal soft/deep · `#22B07D` success · `#FFF4DD`/`#F5DDA6`/`#9A5B00`/`#5C3D00` warning box ·
focus ring `#3DD6F3` = `--ring`. Illustrations (welcome, fetus/fruit size) are inline SVG components whose fills are
tokens, so they flip with the theme. Numbers: Lalezar where the rest of the app uses it.

## Current state (v1) — what we build on

- Backend (Go, `internal/pregnancy`, `/api/v1/pregnancy/*`): profile + dating (LMP / ultrasound / manual, due date
  = +280 d, confidence, uncertainty), onboarding, symptom logs, weekly logs (weight, BP, sugar, swelling, mood),
  fetal movement, hard-coded alerts, week content (10 bilingual text columns in `pregnancy_weekly_content`).
- Admin: `pregnancy-weeks` screen edits those 10 text columns; `messages` screen edits existing `message_contents`
  rows (no create).
- Message engine (`internal/messages`): pregnancy = trimester message + nearest milestone week (4, 8, 12, 20, 28,
  36, 40) + overrides (nausea, fatigue, backache, anxiety) + trimester nutrition/sleep/exercise.
- Frontend: all pregnancy screens exist but are **hidden** — `entities/user/model/steps.ts` (intention +
  pregnancy-basis steps commented out), `widgets/bottom-nav/model/nav-items.ts` (forced cycle), `ProfilePage.tsx`
  (~228–300, app-mode section commented out).

## Screens

| Artboard | Content | Route |
|---|---|---|
| `Setup` «ورود و راه‌اندازی» | welcome (illustration, 3 benefits, «حالت بارداری رو روشن کن» / «فعلاً نه») → step 1/3 dating source segmented (LMP calendar / ultrasound date + week + day / manual week + day, hint per source) → 2/3 optional history (miscarriage, high-risk, condition chips with exclusive «هیچ‌کدام», blood group, Rh) → 3/3 result (weeks+days, due date, confidence pill, usual birth range, basis sentence, «تمومه، بریم», «مبنای محاسبه رو عوض می‌کنم») | `/pregnancy/setup` (also the continuation of onboarding intention = pregnant, and profile → «حالت بارداری») |
| `Main` «امروز» | date strip; week carousel (prev/current/next: trimester·week, title, size line, «بازگشت به امروز»); due-date card (date, days left, usual range); 40-week progress with trimester segments and the 2nd/3rd start dates; quick actions (ثبت علائم روزانه, چکاپ هفتگی, مرور هفته‌ها, هشدارها); next-visit card; smart tip of the week (+ read time, «درباره هفتهٔ N بیشتر بخون»); «مراقبت‌های این هفته» checklist with done count; due-date disclaimer | `/pregnancy` |
| `Week` «هفته‌به‌هفته» | header «هفتهٔ N از ۴۰» + trimester · date range, bookmark; week chip strip (past / current / future dashed) + legend; hero (size illustration, «تقریباً هم‌اندازهٔ یک …», headline, length / weight / heart-rate stats, averages note); tabs جنین / بدن تو / کارهای هفته (highlights with icon+title+body · symptom chips + text + «علائم امروزت رو ثبت کن» · checklist); warning box; reviewer + date + sources | `/pregnancy/weeks/[n]` |
| `Log` «ثبت علائم» | header date · week, other-day picker; mood (5 faces); 9 symptom toggles (تهوع, استفراغ, خستگی, سردرد, کمردرد, حساسیت پستان, سوزش سر دل, یبوست, لکه‌بینی) with severity (خفیف/متوسط/شدید) per selected; water glasses stepper; weight (last entry shown); note for next visit; spotting info box; sticky save («ذخیره شد» state) + offline note | `/pregnancy/log?date=` |
| `Calendar` «تقویم و ویزیت‌ها» | month calendar (visit markers, week-start markers, today), legend, selected-day panel; next-visit card with status stepper (نوبت گرفته شد → انجام شد → نتیجه ثبت شد), prep text, reminder, «مسیریابی»; «برنامهٔ مراقبت‌های بارداری» list with week window, date and state (انجام شد / نوبت داری / رزرو); source note; «گزارش علائم برای پزشک (PDF)»; «ویزیت جدید» | `/pregnancy/calendar` |
| `Alerts` «هشدارها» | «بر اساس ثبت‌های ۷ روز اخیر»; alert cards by level with «چی دیدیم» / «چقدر مطمئنیم» / advice / actions («افزودن به یادداشت ویزیت», «دیدم، ممنون», «ثبت وزن»); legend of the 4 levels (اطلاع, پیشنهاد, ارزش پیگیری, پیگیری زودتر — the last with a contact path); disclaimer | `/pregnancy/alerts` |

Pregnancy-mode bottom nav: امروز · تقویم · (+ ثبت امروز, gradient FAB) · بارداری · پروفایل.

## What the admin defines (user requirement: «این قسمتا باید از ادمین تعریف بشن»)

| Area | Where | Stored in |
|---|---|---|
| Week hero + tabs: size label, illustration key, length, weight, heart rate, headline, baby highlights (icon, title, body), body symptoms chips + text, week tasks (key, text), warning text, reviewer name + review date + sources | admin-web `pregnancy-weeks` (new structured tab next to the 10 text fields) | new `pregnancy_week_details` (one row per week, bilingual JSON) |
| Care plan (first visit, NT, anomaly scan, GTT, Tdap, …): title, kind (visit/test/scan/vaccine), week window, prep text, reminder default, order, active | admin-web `pregnancy-care-plan` (new) | new `pregnancy_care_items` |
| Alert rules: enabled, level (info / suggestion / follow_up / urgent), thresholds (e.g. vomiting streak days, severe count, weight-log interval), window days, texts («چی دیدیم», «چقدر مطمئنیم», advice, action labels), contact line for urgent | admin-web `pregnancy-alert-rules` (new) — texts are `message_contents` rows | `message_contents` group `pregnancy_alert` (payload = params + texts) |
| Smart tip of every week (title, body, read minutes, article link) | admin-web `messages` (group `pregnancy_week_tip`) | `message_contents` group `pregnancy_week_tip`, item_key `1..42` |
| Setup copy: benefits, per-source hints, history disclaimer, result basis sentence templates | admin-web `messages` (group `pregnancy_setup`) | `message_contents` |
| Condition chips for setup | admin-web `pregnancy-care-plan` (small list) or enum-backed with editable labels | existing `PreExistingCondition` enum labels via translations |

## Message engine integration («پیام‌ها رو به انجین پیام‌های اصلی سیستم اضافه کن»)

1. `pregnancy_week_tip/{week}` becomes a first-class layer of the pregnancy base message: exact week → nearest
   milestone (existing) → trimester (existing). The same tip feeds `GET /messages/daily` (SmartTip everywhere) and the
   v2 Today card — one source.
2. Alerts v2 are produced by a rule registry inside the message engine (`internal/messages/pregnancyalerts`):
   detectors in Go (vomiting streak, severe-symptom count, spotting/bleeding/fluid/severe pain = existing critical
   rules, weight not logged this week, week entered, BP/sugar/fetal-movement = existing weekly rules), **params and
   texts from `pregnancy_alert` rows**, output persisted to `pregnancy_alerts` (level mapped onto the existing
   `alert_level` enum + a v2 `level4` key in `recommended_actions`/payload). v1 alert endpoints keep working.
3. Existing symptom overrides read `pregnancy_symptom_logs` in pregnancy mode (today they read the general daily log).
4. The admin messages screen can **create** a row for a known group/key that has no row yet (needed for week tips and
   new rules); groups are listed from a registry, not free text.

## Data — new tables (goose + schema-only Laravel twins, docs/go-migration/migrations.md)

- `pregnancy_week_details`: `week_number` unique, `size_label`, `illustration_key`, `length_cm` (text range),
  `weight_g`, `heart_rate`, `headline`, `highlights`, `body_symptoms`, `body_text`, `tasks`, `warning`,
  `reviewer_name`, `reviewed_at`, `sources` — text fields bilingual JSON; seeded for weeks 1–42 with
  **placeholder copy that needs medical review** (the artboard itself shows «[نام متخصص زنان و زایمان]»).
- `pregnancy_care_items`: `key` unique, `title`/`prep` bilingual, `kind`, `week_from`, `week_to`,
  `remind_before`, `sort_order`, `is_active`; seeded with the 5 items in the artboard.
- `pregnancy_daily_extras` (user_id, log_date unique pair): `mood` (1–5), `water_glasses`, `heartburn_severity`,
  `constipation_severity`, `visit_note` — the columns `pregnancy_symptom_logs` lacks, kept separate so the Laravel
  contract of `/pregnancy/symptoms` stays unchanged.
- `pregnancy_week_user_state` (user_id, week unique pair): `bookmarked`, `done_task_keys` json.
- Shipped in T-M7-01 as goose `00005_pregnancy_v2.sql` + Laravel twin `2026_09_26_000001_create_pregnancy_v2_tables.php`
  (identical seeds, insert-or-ignore). sqlc: `db/queries/pregnancy/v2_*.sql` → `internal/pregnancy/store`
  (`GetWeekDetails`, `ListWeekDetailsRange`, `UpsertWeekDetails`, care-item CRUD + `SetCareItemSortOrder`/`SetCareItemActive`,
  `UpsertDailyExtras`, `SetDailyVisitNote`, `ListDailyExtrasRange`, `UpsertWeekUserState`, `ListWeekUserStatesRange`, …).

### Stored shapes (T-M7-01)

`{L}` = translatable object keyed by language code (`{"fa": "…", "en": "…"}`, read with `i18n.Pick`).

| Column | Shape |
|---|---|
| `size_label` | `{L}` noun only («تمشک» / "raspberry"); the sentence frame («تقریباً هم‌اندازهٔ یک …») is UI copy. NULL for weeks 1–3 |
| `illustration_key` | slug (`raspberry`, `poppy_seed`, …); NULL for weeks 1–3 |
| `length_cm` / `weight_g` / `heart_rate` | ASCII text: `"1.6"`, `"<1"`, `"150-170"` (UI localises digits and adds «~» / units); NULL when not meaningful |
| `headline`, `body_text`, `warning` | `{L}` (the warning's lead line «این موارد رو با پزشکت در میون بذار:» is UI copy) |
| `highlights` | `[{icon, title: {L}, body: {L}}]` — icons seeded: `hand`, `heart`, `face`, `baby` |
| `body_symptoms` | `[{key, label: {L}}]`; keys that match the Log screen (`nausea`, `vomiting`, `fatigue`, `headache`, `back_pain`, `breast_pain`, `heartburn`, `constipation`, `spotting`) can deep-link to it; others (`frequent_urination`, `smell_sensitivity`, `morning_sickness`, `braxton_hicks`, …) are display-only |
| `tasks` | `[{key, text: {L}}]`; done state = `pregnancy_week_user_state.done_task_keys` (`["folic_acid", …]`) |
| `reviewer_name` / `reviewed_at` / `sources` | `{L}` / date / `[{title: {L}, url}]`. **`reviewed_at IS NULL` = not clinically reviewed**; every seeded row has it NULL and a `[needs review]` entry in `sources` (replace on sign-off, T-M7-15) |
| `pregnancy_care_items.kind` | `visit` \| `test` \| `scan` \| `vaccine`; `remind_before` in days. Seeded keys: `first_visit` (6–10), `nt_scan` (11–14), `anomaly_scan` (18–22), `gtt` (24–28), `tdap` (27–36) |
| `pregnancy_daily_extras` | `mood` 1–5, `water_glasses`, `heartburn_severity` / `constipation_severity` ∈ `mild|moderate|severe` |

`message_contents` payloads (one row per locale; placeholders are `{name}`):

- `pregnancy_week_tip` / `1..42`: `{title, body, read_minutes, article_url}` (`article_url` null → the week page).
- `pregnancy_alert` / rule key (`vomiting_streak`, `severe_symptom_count`, `critical_symptom`, `weight_missing_week`,
  `week_entered`, `bp_high`, `sugar_high`, `fetal_movement`): `{enabled, level, window_days, params{…}, title, what_we_saw,
  how_sure, advice, actions: [{key, label}], contact}`. `level` ∈ `info|suggestion|follow_up|urgent`; action keys seeded:
  `ack`, `add_to_visit_note`, `log_weight`, `open_week`, `call`. `enabled` / `level` / `window_days` / `params` are
  behaviour, not copy — the engine should read them from the **default-language** row and only the texts from the request
  locale (the seed keeps them identical in every locale). Seeded params: vomiting_streak `{min_streak_days: 3,
  severe_min_count: 2}`, severe_symptom_count `{min_count: 3, symptoms: […]}`, critical_symptom `{symptoms: [bleeding,
  fluid_leakage, severe_sudden_pain, spotting], spotting_until_week: 12}`, weight_missing_week `{from_week: 1}`,
  bp_high `{systolic_min: 140, diastolic_min: 90}` (v1 thresholds), sugar_high `{fasting_max: 95, post_meal_max: 140}`
  (v1), fetal_movement `{from_week: 24, statuses: [reduced, none]}` (v1). Placeholders per rule: `{days}`,
  `{severe_count}`, `{count}`, `{symptom}`, `{week}`, `{basis}`, `{systolic}`, `{diastolic}`, `{fasting}`, `{post_meal}`,
  `{status}`.
- `pregnancy_alert` / `legend` (not a rule): `{window_note, title, levels: {info|suggestion|follow_up|urgent: {label,
  description}}, disclaimer}`.
- `pregnancy_setup` / `welcome` `{title, body, benefits[], primary, secondary}`, `dating` `{title, body}`,
  `source_lmp|source_ultrasound|source_manual` `{label, hint}`, `history` `{title, body, disclaimer, skip}` (disclaimer
  already rewritten for server storage — open point 1), `result` `{lead, suffix, due_label, confidence, range,
  basis_lmp, basis_ultrasound, basis_manual, primary, secondary}`, `due_disclaimer` `{title, body}`.

- Visits reuse **M3 appointments** (`/api/v1/care/appointments`, docs/care-reminders/README.md) with
  `meta.care_item_key` and `meta.stage` (`booked|done|result`) + `meta.result_note`.

## API (Go only, new prefix `/api/v1/pregnancy/v2/` so nginx can route it to Go independently)

| Method | Path | Notes |
|---|---|---|
| POST | `/pregnancy/v2/dating-preview` | body = dating source fields → weeks, days, due date, range, confidence, basis sentence (no write) |
| GET | `/pregnancy/v2/today` | week carousel (prev/current/next summaries), due card, progress + trimester start dates, next visit, week tip, this week's tasks with done state, alert badge count |
| GET | `/pregnancy/v2/weeks/{n}` | details + date range + bookmark + task state + tip |
| PUT | `/pregnancy/v2/weeks/{n}/state` | `{bookmarked?, done_task_keys?}` |
| GET/PUT | `/pregnancy/v2/days/{date}` | mood, symptoms {key: severity}, water, weight (→ weekly log of that week), visit_note; response includes last weight + alerts raised |
| GET | `/pregnancy/v2/calendar?month=` | day markers (visit, week start), visits, care plan items with state + suggested dates |
| GET | `/pregnancy/v2/report?from=&to=` | data for the doctor PDF (symptom timeline, weights, notes) |
| GET | `/pregnancy/v2/alerts` | last 7 days + level legend texts |
| POST | `/pregnancy/v2/alerts/{id}/actions/{action}` | `ack`, `add_to_visit_note` |
| admin | `/api/admin/v1/pregnancy-weeks/{n}/details`, `/pregnancy-care-items[/{id}]`, `/pregnancy-alert-rules[/{key}]`, `POST /messages` (create in registered group) | editor + super |

Activation/onboarding keep the v1 endpoints (`/pregnancy/activate`, `/onboarding`, `/profile`).

## Open points for the user

1. The Setup artboard says history is «فقط روی گوشی خودت و به‌صورت رمزنگاری‌شده ذخیره می‌شه» but v1 stores it on the
   server (`pregnancy_profiles`). Either change the copy (default in the tasks) or move those fields on-device.
   T-M7-01 seeded the changed copy (`pregnancy_setup/history.disclaimer`).
2. The Log artboard promises offline save + later sync. T-M7-12 adds an outbox for this screen; if that's too much,
   the copy changes instead.
3. All seeded medical copy (week details, care plan windows, alert thresholds) needs a clinician's sign-off (T-M7-15).

## Rollout (T-M7-15) — staging 2026-09-26

**Deploy/routing:** no redeploy — the code was shipped to staging by T-M3-09 (goose version 5). Staging is Go-only, so
`/api/v1/pregnancy/v2/*` and the admin endpoints already reach `stage-backend-go`. Production untouched.

**Seeds in the stage DB:** `pregnancy_week_details` 42 rows (weeks 1–42); `pregnancy_care_items` 5
(`first_visit, nt_scan, anomaly_scan, gtt, tdap`); `message_contents` `pregnancy_week_tip` 42 fa + 42 en,
`pregnancy_alert` 9 fa + 9 en (8 rules + legend).

**API e2e** (throwaway user 09900000903, OTP read from `otp_verifications`, curl inside `ritme-edge` against
`stage-backend-go`, LMP 2026-08-01 → week 9):
```
POST v2/dating-preview 200 (8w+0d, due 2027-05-08, confidence medium)
POST /pregnancy/onboarding 201
GET  v2/today 200 (week 9, 3 tasks) | GET v2/weeks/9 200 (tasks folic_acid, small_meals, book_nt)
PUT  v2/weeks/9/state {bookmarked, done_task_keys:[folic_acid]} 200 → GET reflects both
PUT  v2/days/2026-09-26 {mood, water, weight 62.5, note, nausea mild, spotting mild} 200 → alert urgent critical_symptom
GET  v2/days/2026-09-26 200 (all fields round-trip)
GET  v2/alerts 200: critical_symptom (urgent) + week_entered (info)
POST alerts/{id}/actions/add_to_visit_note 200 → day visit_note gains the spotting line
POST alerts/{id}/actions/ack 200 → is_acked true; today.unread_alerts 1
GET  v2/calendar 200: 5 care items to_book
POST /care/appointments {care_item_key: nt_scan, stage: booked, 2026-10-12} 201
GET  v2/calendar?month=1405-07 → visit on 2026-10-12, next_visit nt_scan booked, care plan nt_scan state booked
GET  v2/report 200
cleanup: DELETE appointment 200; user 3 + profile/logs/alerts/state/tokens/otp rows deleted in DB
```
Notes: `month=` is in the request locale's calendar (fa → Jalali `1405-07`; `2026-10` under fa resolves to Jalali year
2026). Spotting also writes a v1 `symptom_based` alert row next to the v2 one (known T-M7-04 duplicate).

**Open (human):** clinical sign-off (list in `tasks/PROGRESS.md` T-M7-15), light/dark UI click-through (signup as
pregnant → setup → today → week → log incl. offline → alert → calendar visit → PDF → switch back to cycle) and
screenshots — the agent may not read the staging gate password.

## Local e2e (T-M7-15) — 2026-09-27

Everything below ran locally on branch `stage`. backend-go ran on `:8020` against the docker test-stack MariaDB, using a
scratch DB `ritme_m715` (goose up to v5, so `00005_pregnancy_v2` seeded 42 week details, 5 care items, 84 week tips,
18 alert rows and 16 setup rows) with `SMS_PROVIDER=log`. The personal-access OAuth client row was inserted by hand.
The Next dev server ran on `:3000` with `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8020/api/v1` passed as an env var
(`.env.local` untouched). admin-web was not needed. Headless Chrome was driven over CDP (Node's global `WebSocket`,
20 s timeout per call). The viewport was 390×844 @2x, mobile + touch, locale fa. Three throwaway users signed up
through the real UI, with the OTP read from `otp_verifications`: `09120000701` (light), `09120000702` (dark) and
`09120000703` (repro attempt for bug 7). The scratch DB was dropped afterwards and the test stack left running. No
staging or production host was contacted.

### verify-all

| Check | Result |
|---|---|
| `go vet ./...` | ✔ pass |
| `go test ./...` | ✔ pass (63 packages ok) |
| `golangci-lint run` | ✔ 0 issues |
| `npm run typecheck` | ✔ pass |
| `npm run lint` | ✔ pass |
| `npm run fsd:lint` | ✔ No problems found |
| `npm run lint:styles` | ✔ style gate passed (457 files) |
| `npm run lint:dark` | ✔ passed (57 contrast pairs, 123 tokens; only the pre-existing light-mode ⚠ pairs) |
| `npm run test` | ✔ 68 files, 559 passed |
| Laravel (`pint`, `artisan test`) | skipped: `backend/` unchanged |

### UI flow (light and dark, same steps)

Every screen got an automated DOM check. It looked for horizontal overflow (`scrollWidth` > `innerWidth`, or an
element outside the viewport), raw i18n keys in the text, Latin digits in the fa text, and console errors or
exceptions. There was **no overflow, no raw key, and no JS exception or `console.error`** on any screen. The Latin
digits and the non-2xx network lines are listed in the table and under Bugs.

| # | Step | Result |
|---|---|---|
| 1 | `/signup` → OTP → name → birthday → weight → height → intention «باردارم» → dating basis (manual 9w+2d) → conditions → setting-up | ✔ light lands on `/pregnancy`, week 10 (9w2d), due ۱۰ اردیبهشت ۱۴۰۶, confidence «کم ±۵ روز». ✘ dark (once): landed on «حالت بارداری روشن نیست» and `/v2/today` returned 409 (bug 7) |
| 2 | Setup v2 `/pregnancy/setup`: welcome → 1/3 LMP calendar (2 مرداد ۱۴۰۵) → 2/3 history (هیچ‌کدام, O, مثبت) → 3/3 result → «تمومه، بریم» | ✔ result «۹ هفته و ۲ روز», due ۱۰ اردیبهشت ۱۴۰۶, range ۲۴ فروردین–۲۷ اردیبهشت, «متوسط ±۳ روز», basis sentence. The dark user was recovered here. ✘ the step progress bar renders as a segmented control (bug 3) |
| 3 | Today `/pregnancy` | ✔ date strip, due card, quick actions (4), next visit «اولین ویزیت پزشک», reminders card, week tip + «درباره هفتهٔ ۱۰ بیشتر بخون», «مراقبت‌های این هفته ۰ از ۳», disclaimer. ✘ week carousel collapsed to a 32 px strip (bug 1). ✘ trimester bars filled wrong (bug 2) |
| 4 | Week `/pregnancy/weeks/10` via «مرور هفته‌ها»: chip strip + legend, hero, tabs جنین / بدن تو / کارهای هفته, warning box, «در انتظار بازبینی متخصص» + sources; bookmark; tick `folic_acid` | ✔ bookmark → `aria-pressed=true`; tick → API `folic_acid done`, and Today shows it checked. ✘ stats `3.1` / `4` / `140-170` in Latin digits (bug 4) |
| 5 | Log `/pregnancy/log` (FAB): mood «روبه‌راه», تهوع متوسط + خستگی خفیف, water 3, weight 62.5, visit note → save | ✔ «ذخیره شد». `GET v2/days/2026-09-27` returns mood 4, symptoms, `water_glasses 3`, weight 62.5, note and `last_weight` |
| 6 | Offline outbox: `Network.emulateNetworkConditions offline` → add سردرد + 1 glass → save → back online | ✔ `navigator.onLine=false`, status «بدون اینترنت ذخیره شد — با وصل شدن فرستاده می‌شه». The API was unchanged while offline and had `headache: mild`, `water_glasses: 4` within 4 s of going online (both themes). ✘ the status line stays «queued» after the sync (bug 6) |
| 7 | Log لکه‌بینی (خفیف) at week 10 → save | ✔ spotting info box shown; «این ثبت یک پیام تازه ساخت · پیگیری زودتر · لکه‌بینی ثبت شده · دیدن هشدارها» |
| 8 | Alerts `/pregnancy/alerts` | ✔ `critical_symptom` (urgent, contact box with ۱۱۵) + `week_entered` (info), legend of 4 levels, disclaimer. «دیدم، ممنون» on the info card removes its ack button. ✘ «وارد هفتهٔ 10 شدی» in Latin digits (bug 5) |
| 9 | Calendar `/pregnancy/calendar` → care plan «رزرو» on «سونوگرافی NT و غربالگری اول» → appointment form | ✔ prefilled title, date ۱۰ مهر ۱۴۰۵ (`date=2026-10-02`), `care_item_key=nt_scan`. Time is required. Saved → `/reminders/appointment/1` |
| 10 | Back to Calendar | ✔ NT row flips to «نوبت داری» with its date; the next-visit card shows the NT scan, «۵ روز دیگه», the stage stepper «نوبت گرفته شد» active, prep text and «یادآور: ۱ روز قبل». The visit dot is on ۱۰ مهر |
| 11 | «گزارش علائم برای پزشک (PDF)» | ✔ on-device PDF (73 KB, 1 page): title, date range, week, due date, daily moods/symptoms/water, visit note, weight, disclaimer. **Footer «ریتمی · صفحهٔ ۱ از ۱» uses Persian digits** (the T-M4-11 fix holds) |
| 12 | Profile → «برگرد به حالت چرخه» (native `confirm`, accepted) | ✔ `/home` cycle home with the cycle nav (تحلیل). The booked NT visit shows in «یادآورهای امروز» |

### Screenshots (`screenshots/`)

Each file exists as `-light` and `-dark`, except where noted. They are downscaled to 585×1266 and pngquant'ed
(60 files, 2.5 MB).

| File | Shows |
|---|---|
| `01-signup-intention-*` | signup intention step |
| `02-signup-pregnancy-basis-*` | signup dating basis (manual week) |
| `03-signup-setting-up-*` | setting-up ring |
| `04-today-after-signup-*` | first landing on Today |
| `05-setup-welcome-*` … `08-setup-result-*` | Setup v2 welcome, dating (LMP calendar), history, result |
| `09-today-*`, `11-today-bottom-*` | Today top (collapsed carousel, trimester bars) and bottom |
| `12-week-*`, `13-week-baby-tab-bottom-*`, `14-week-body-tab-*`, `15-week-tasks-tab-*` | Week 10 hero, tabs, warning, reviewer line |
| `16-log-empty-*`, `17-log-filled-*`, `19-log-saved-online-*` | Log empty, filled, saved |
| `20-log-queued-offline-*` | offline save queued in the outbox |
| `21-log-synced-online-*` | the same save after reconnect: «ذخیره شد» |
| `22-log-spotting-info-*`, `23-log-alert-raised-*` | spotting info box, and the new alert after save |
| `24-alerts-*`, `25-alerts-legend-*`, `26-alerts-after-ack-*` | Alerts screen, legend, after ack |
| `27-calendar-*`, `28-calendar-care-plan-*` | Calendar and care plan before booking |
| `29-appointment-form-prefilled-*` | appointment form prefilled from the care plan |
| `30-calendar-after-booking-*`, `31-calendar-care-plan-booked-*` | next-visit card and «نوبت داری» |
| `33-pdf-report-page1-light` | doctor PDF page 1 (rendered with `sips`) |
| `34-profile-*` | Profile with the app-mode section |
| `36-cycle-home-after-switch-*` | cycle home after switching back |

The round «N» badge bottom-left is the Next.js dev-mode indicator, not app UI.

### Bugs found

Items 1–8 and 9f were fixed in T-M7-16. The screenshots of signup, Setup, Today, Week, Log, Alerts, Calendar
(`27-*`) and Profile were re-taken locally afterwards (scratch DB `ritme_m716`, users `09120000712`/`09120000713`),
with the same 585×1266 + pngquant compression. The other files are still from T-M7-15. 9a (outside T-M7-16's
paths) and 9b–9e, 9g–9j are still open.

1. **Today week carousel collapses to a 32 px strip.** Only «سه‌ماههٔ اول · هفتهٔ ۱۰ از ۴۰» shows (clipped). The
   illustration, size line, prev/next buttons and «مرور هفته‌ها» link are hidden (`scrollHeight` 370 vs height 32).
   The cause is the `<section className="mx-4 overflow-hidden …">` in
   `frontend/src/widgets/pregnancy-week-carousel/ui/PregnancyWeekCarousel.tsx:61`. It is a flex item of the
   `scroll flex flex-col` column (`frontend/src/screens/pregnancy/ui/PregnancyPage.tsx:41`). With `overflow-hidden` its
   min-height is 0, so it shrinks when the page is taller than the viewport (needs `shrink-0`). To reproduce: open
   `/pregnancy` with a dated pregnancy (`09-today-*.png`).
2. **Trimester progress bars show the wrong fill.** At week 10 the bars read T1 0 %, T2 32 %, T3 70 %. The API's
   `progress.trimesters[].percent` is the *start position on the 40-week bar*
   (`backend-go/internal/pregnancy/v2/service.go:148`, OpenAPI "position of the start on the 40-week bar"). The
   frontend uses it as a per-trimester fill width (`PregnancyPage.tsx:132-135`; the type comment in
   `frontend/src/entities/pregnancy/model/v2-types.ts:197` says "how much of this trimester is behind the user"). The
   «از <date>» label rule `s.percent === 0` (`PregnancyPage.tsx:143`) has the same misreading. The contract has to be
   settled on one side.
3. **The Setup v2 step progress renders as a segmented control.** The progress bar uses `className="seg"`
   (`frontend/src/screens/pregnancy-onboarding/ui/PregnancyOnboardingPage.tsx:114`), which is the segmented-tab style
   (`frontend/src/app/globals.css:507-513`). You get three large white pill buttons edge to edge, with no gutter and
   no visible fill for the done steps (`06-…`, `08-setup-result-*`).
4. **Week stats show Latin digits.** `3.1` سانتی‌متر · `4` گرم · `140-170` ضربان: `d.length` / `d.weight` / `d.heartRate` are
   rendered raw (`frontend/src/screens/pregnancy-week/ui/PregnancyWeekPage.tsx:144-146`). They need
   locale-digit formatting (the admin stores plain strings).
5. **The `week_entered` alert has Latin digits in fa.** «وارد هفتهٔ 10 شدی», «هفتهٔ 10 بارداری از امروز شروع شده». The
   `{week}` var is `strconv.Itoa(f.Week)` (`backend-go/internal/messages/pregnancyalerts/rules.go:123`) and is not
   localized for fa. Other rules' numeric vars may have the same issue.
6. **The log status stays «queued» after the outbox syncs.** After going back online the entry is sent (the API has
   the data), but the line still reads «بدون اینترنت ذخیره شد — با وصل شدن فرستاده می‌شه». While the save was only
   queued, the button already said «ذخیره شد» (green). `useOutboxReplay`'s `onSent` in
   `frontend/src/screens/pregnancy-log/model/use-save-day.ts` only invalidates queries and never moves `status` off
   `queued`; `DayLogPage.tsx:92` treats `queued` as `saved`.
7. **A pregnant signup was once saved as a cycle user (not reproduced).** For `09120000702` the setting-up screen
   sent `POST /profile` with the cycle-branch defaults (`pregnancy_intention NULL`, `last_period_start` = today,
   5/28, `users.name NULL`) and **no** `/pregnancy/activate` or `/pregnancy/onboarding`. The user landed on
   `/pregnancy` → `v2/today` 409 → «حالت بارداری روشن نیست» (`04-today-after-signup-dark.png`), even though the
   persisted `ritme-onboarding` store held `intention: pregnant` + basis. The page also stalled about 36 s before
   redirecting. This run followed a `localStorage.clear()` logout in the same tab, and a clean re-run with
   `09120000703` was correct. `SettingUpPage.tsx:52-75` reads the store once on mount and swallows every error, so a
   non-hydrated or reset store silently produces a non-pregnant account. The screen should at least guard that
   `intention` is set before saving.
8. **Profile shows Latin digits (outside this module, pre-existing).** «60 کیلوگرم», «165 سانتی‌متر», «28 روز», «5 روز».
   `measureOrEmpty` / `daysOrEmpty` pass raw numbers (`frontend/src/screens/profile/ui/ProfilePage.tsx:209-212`).

Minor / design:
- 9a. The care-plan booking link sends no category, so NT (a scan) is prefilled as «ویزیت دوره‌ای» instead of «سونوگرافی»
  (`frontend/src/screens/pregnancy-calendar/model/view.ts:21`).
- 9b. After booking from the care plan the app goes to `/reminders/appointment/{id}`, not back to the calendar
  (`reminder-appointment-form` `router.push`, lines 182/186).
- 9c. The urgent `critical_symptom` card has no «تماس با پزشک» button. The seed has no `contact.phone`, so
  `resolveAlertAction` drops `call` (`frontend/src/screens/pregnancy-alerts/model/alerts.ts:41-44`). This is content
  or config for the sign-off.
- 9d. The Alerts list puts the info card above the urgent one. Urgent should probably come first.
- 9e. The Week «کارهای هفته» tab uses native square checkboxes (`WeekTabs.tsx:146-152`), while Today uses round ones.
- 9f. The Log weight field shows the stored value as `62.5` (Latin) (`DayLogPage.tsx:341`).
- 9g. «برگرد به حالت چرخه» uses the native `window.confirm` (`ProfilePage.tsx:234`), which is off-brand.
- 9h. After switching back to cycle, a user without period data gets `GET /messages/daily` 400 («تاریخ آخرین پریود») and a
  home full of «—» with no prompt to enter the last period.
- 9i. PDF: in «(خفیف) · ۴ لیوان آب» the `·` sits next to a Persian digit and reads like «۴۰».
- 9j. Copy (for the clinician): the LMP hint says «حدود ±۵ روز» but the result says «متوسط · حدود ±۳ روز». The spotting
  info box says mild first-trimester spotting is common, while the rule raises it as «پیگیری زودتر» (urgent).

### Still open (human)

- Clinical content sign-off: the list is in `tasks/PROGRESS.md` T-M7-15 (week details 1–42 + tips, care-plan windows,
  alert texts and thresholds), plus 9c and 9j above.
- Staging UI click-through: the agent has no server access.
- Production when asked.

## Staging UI e2e (T-M7-15) — 2026-09-28

This run used staging at `stage` @ `ea8a1eb` (goose v6, includes the T-M7-16 fixes). There was no redeploy, and
production was not touched. Headless Chrome ran locally (port 9243, its own profile) and was driven over CDP, with a
20 s timeout on every call. The viewport was 390×844 @2x, mobile + touch, locale fa. The gate cookie `ritme_stage` came
from one Basic-auth page load (`Network.setExtraHTTPHeaders`), and the header was then dropped, so the app's Bearer
token worked on every call. Two throwaway users signed up through the real UI: `09900000971` (light) and
`09900000972` (dark). The dark user signed up in the same tab after a UI logout. The OTP was read from
`otp_verifications` in the `ritme-stage` MariaDB. Afterwards both users and all their rows were deleted from the stage
DB (tokens, reminders, pregnancy profile/logs/extras/alerts/week state, user profile, OTP rows).

**API:** 185 calls were recorded. Every `/api/v1/*` response carried `X-Backend: go` (0 exceptions). All of them
returned 2xx except `GET /messages/daily` 400 after switching back to cycle (known 9h). Main calls: `send-otp`/`verify-otp`
200, `POST /profile` 200, `pregnancy/activate` 200, `pregnancy/onboarding` 201, `v2/dating-preview` 200, `v2/today` 200,
`v2/weeks/10` 200, `PUT v2/weeks/10/state` 200, `GET/PUT v2/days/2026-09-28` 200, `v2/alerts` 200, `alerts/{id}/actions/ack` 200,
`v2/calendar?month=1405-07` 200, `POST care/appointments` 201, `v2/report` 200, `pregnancy/deactivate` 200.

The DOM check ran on 65 screens and found no horizontal overflow, no off-screen element, no raw i18n key and no Latin
digit in fa text. The only JS console errors were the 400 above.

| # | Step | Light | Dark |
|---|---|---|---|
| 1 | `/signup` → OTP → name/birthday/weight/height → «باردارم» → manual 9w+2d → conditions → setting-up | ✔ `/fa/pregnancy`, week 10, due ۱۱ اردیبهشت ۱۴۰۶ | ✔ (signup after logout in the same tab — bug 7 not reproduced) |
| 2 | Setup v2: welcome → LMP ۲ مرداد ۱۴۰۵ → history (هیچ‌کدام, O, مثبت) → result → «تمومه، بریم» | ✔ ۹ هفته و ۳ روز, due ۱۰ اردیبهشت ۱۴۰۶, «متوسط ±۳ روز»; step bar is a real progress bar (bug 3 fixed) | ✔ |
| 3 | Today | ✔ full week carousel (bug 1 fixed), trimester bars correct (bug 2 fixed), next visit, reminders, tip, 3 tasks | ✔ |
| 4 | Week 10: tabs, bookmark, tick `folic_acid` | ✔ `aria-pressed=true`, tick → PUT state 200, Today shows it checked; stats in Persian digits (bug 4 fixed) | ✔ |
| 5 | Log: mood, تهوع متوسط + خستگی خفیف, water, weight 62.5, note → save | ✔ API round-trips every field | ✔ |
| 6 | Offline: `emulateNetworkConditions offline` → سردرد + 1 glass → save → online | ✔ `onLine=false`, «بدون اینترنت ذخیره شد…», button *not* «ذخیره شد» while queued, API unchanged; synced in 0.7 s → «ذخیره شد» | ✔ synced in 1.3 s; the UI still showed «queued» 1.5 s later and flipped to «ذخیره شد» a few seconds after (bug 6 fixed, with a short lag) |
| 7 | لکه‌بینی → save | ✔ info box + «این ثبت یک پیام تازه ساخت · پیگیری زودتر · لکه‌بینی ثبت شده» | ✔ |
| 8 | Alerts | ✔ `critical_symptom` (urgent, ۱۱۵ box) + `week_entered` «وارد هفتهٔ ۱۰ شدی» (bug 5 fixed); ack 200 | ✔ |
| 9 | Calendar → care plan «رزرو» NT → form (title, ۱۰ مهر ۱۴۰۵, `care_item_key=nt_scan`) → name + time → «ذخیره نوبت» | ✔ 201 | ✔ 201 → `/reminders/appointment/7` |
| 10 | Back to Calendar | ✔ NT «نوبت داری», next visit «۴ روز دیگه», stage «نوبت گرفته شد», prep, «یادآور: ۱ روز قبل», dot on ۱۰ مهر | ✔ |
| 11 | «گزارش علائم برای پزشک (PDF)» | ✔ 73 KB, 1 page, footer «ریتمی · صفحهٔ ۱ از ۱» in Persian digits | ✔ 73 KB |
| 12 | Profile → «برگرد به حالت چرخه» (native confirm accepted) | ✔ deactivate 200 → `/fa/home` cycle home, booked NT in «یادآورهای امروز» | ✔ |

**Screenshots:** they are in `screenshots/stage-*.png` (67 files, 585 px wide, pngquant, 2.5 MB). Each step has a `-light`
and a `-dark` file. The PDF image `stage-33-pdf-report-page1-light` exists in light only. The names follow the local run: `00-signup`,
`01`–`04` signup, `05`–`08` Setup v2, `09`/`11` Today, `12`–`15` Week, `16`/`17`/`19` Log, `20` queued offline,
`21` synced, `22`/`23` spotting + alert, `24`–`26` Alerts, `27`–`31` Calendar + booking, `32` after PDF, `34` Profile, and `36`
cycle home.

**Still open on stage:**
- 9a: the NT booking is prefilled as «ویزیت دوره‌ای».
- 9b: saving goes to `/reminders/appointment/{id}`.
- 9c: no call button.
- 9d: the info alert is listed above the urgent one.
- 9h: `/messages/daily` 400 on the cycle home.
- 9i: the PDF «(خفیف) · ۴ لیوان آب» reads like «۴۰».
- 9j: copy.
- Minor: the Log weight field shows «۶۲.۵» with an ASCII dot, while «آخرین ثبت» shows «۶۲٫۵».

**New:** logout leaves the `ritme-onboarding` store in `localStorage`. It still holds the previous user's name, phone,
birth date, weight, height, intention and pregnancy basis. `useLogout` only clears the token and the query cache
(`frontend/src/features/auth/api/mutations.ts:126-129`), and the store is persisted in
`frontend/src/entities/user/model/store.ts:156`. The next login resets it, so it did not leak into the next account in
this run. On a shared device, though, it is personal data that stays after logout.

**Fixed locally in T-M7-17 (not yet on stage):** every session end (logout, account deletion, a session-ending 401) now
wipes `ritme-onboarding` and drops the offline outbox (`shared/session/cleanup.ts`); 9a — «رزرو» on a care-plan row now
sends `topic` (scan → سونوگرافی, test → آزمایش, vaccine → واکسن, visit → ویزیت دوره‌ای) and the appointment form
prefills it; 9h — a `/messages/daily` 400 is "no message yet" and the cycle home shows a
«تاریخ آخرین پریودت رو ثبت کن» state; 9i — fa separators beside digits are «،» (the PDF day line included); the Log
weight field shows «۶۲٫۵».
