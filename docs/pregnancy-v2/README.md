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
2. The Log artboard promises offline save + later sync. T-M7-12 adds an outbox for this screen; if that's too much,
   the copy changes instead.
3. All seeded medical copy (week details, care plan windows, alert thresholds) needs a clinician's sign-off (T-M7-15).
