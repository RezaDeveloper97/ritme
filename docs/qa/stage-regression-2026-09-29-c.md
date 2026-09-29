# Stage regression 2026-09-29 — part C (pregnancy v2 + admin panel)

The target was `https://stage.ritmeapp.ir`, compose project `ritme-stage`, running `stage` @ `e3aea8d` (goose v8). There was
no redeploy, production was not touched and no code was changed.

**Setup.** Headless Chrome ran locally on CDP port 9303 with its own profile. A Node driver sent every CDP call with a 20 s
timeout, and shell calls ran behind a perl `alarm`. The viewport was 390×844 @2x, mobile + touch, locale `fa`. Full-page
captures were made by growing the viewport to the height of the scroll container. The gate cookie `ritme_stage` came from
one Basic-auth page load (curl with a chmod-600 creds file, never printed). The cookie was then set in Chrome, and no
Basic header was sent after that. The admin login was typed from a chmod-600 file. OTPs were read from
`otp_verifications` in the `ritme-stage` MariaDB.

**Test users:**
- `09900001301` (light, uid 14) and `09900001302` (dark, uid 19) both signed up through the real UI.
- The dark user signed up in the same tab after a UI logout.
- Both were removed with `DELETE /api/v1/account` at the end. Their OTP rows were deleted from the stage DB.

**Dark mode** was turned on with `localStorage.ritme_theme = 'dark'` in the app and with the panel's «حالت تیره» button in
admin-web.

## Result

**PASS with findings.** Every step of the pregnancy v2 flow and the admin checks worked in light and dark. There were no
blockers and no high-severity bugs. I found 1 medium bug, 8 low bugs and 3 info notes (below).

## Checklist

### Pregnancy v2 (fa, light ✔ / dark ✔ unless noted)

| # | Step | Result |
|---|---|---|
| 1 | `/signup` → OTP (DB) → name / birthday / weight / height → «باردارم» → manual ۹ هفته + ۲ روز → conditions → «تمام» | ✔ lands on `/fa/pregnancy`. `POST /profile` has no `last_period_start` |
| 2 | Setup v2 welcome reads the admin copy (`GET /pregnancy/v2/setup-copy`) | ✔ |
| 3 | Admin edits the welcome title (`/panel/messages/279`, title + « QA-C») → app `/fa/pregnancy/setup` | ✔ shows «به حالت بارداری خوش اومدی QA-C» right away (`c-06`). Restored byte-exactly (see *Restore*); the app then showed the original title again |
| 4 | Setup: dating (LMP ۲ مرداد ۱۴۰۵; ultrasound tab looked at) → history (هیچ‌کدام, O, مثبت) → result | ✔ ۹ هفته و ۴ روز, due ۱۰ اردیبهشت ۱۴۰۶, «دقت تخمین: متوسط», range ۲۴ فروردین تا ۲۷ اردیبهشت. The math checks out: LMP 2026-07-24 + 280 d = 2027-04-30 |
| 5 | Today: age headline, carousel (next / «بازگشت به امروز»), reminders card under the hero, due card, trimester dates, quick actions, next visit, tip, checklist | ✔ matches `today-design.png`. «هفتهٔ ۱۰ از ۴۰» above «۹ هفته و ۴ روز» is the known B2 won't-fix |
| 6 | Alerts badge counts `week_entered` without opening Alerts | ✔ `v2:week_entered` row created at the first `GET /v2/today`; badge «هشدارها، ۱ پیام تازه» (see bug L1 for the count after a log) |
| 7 | Week 10: tabs جنین / بدن تو / کارهای هفته, bookmark (`aria-pressed=true`), tick `folic_acid` → `PUT /v2/weeks/10/state` 200 | ✔ matches the three week designs (week 10 has one highlight, which is seed content) |
| 8 | Log online: mood, تهوع متوسط + خستگی خفیف, water ۳, weight ۶۲٫۵ (fa digits in the field), visit note → save | ✔ `PUT /v2/days/2026-09-29` 200. The DB has mood 4, water 3, nausea moderate, fatigue mild and the note |
| 9 | Log offline: `Network.emulateNetworkConditions offline` → سردرد + 1 glass → save → online | ✔ «بدون اینترنت ذخیره شد — با وصل شدن فرستاده می‌شه», button not «ذخیره شد» while queued; synced ≤1 s after going online (DB headache mild, water 4) → «ذخیره شد» |
| 10 | Dark «ذخیره شد» contrast | ✔ #fff on rgb(15,123,108) = **5.16:1**, 15px/800 (AA) |
| 11 | لکه‌بینی → save → urgent alert | ✔ «این ثبت یک پیام تازه ساخت · پیگیری زودتر · لکه‌بینی ثبت شده» + the spotting info box; `v2:critical_symptom` (emergency) row |
| 12 | Alerts: actions per level, fact dates | ✔ urgent card: «دیدم، ممنون» only + ۱۱۵ contact box; info card: no actions. Fact dates «امروز» / «۵ مهر ۱۴۰۵». Ack → `POST …/actions/ack` 200, row read + dismissed |
| 13 | Calendar: care plan «هفتهٔ ۶ تا ۱۰، حدود ۷ مهر» … «حدود ۲ بهمن»; source note = admin `calendar_note` + LMP basis | ✔ |
| 14 | NT «رزرو» → form prefilled via `?prefill=<id>` (topic سونوگرافی, title, ۱۰ مهر, ۱ روز قبل) → doctor + time → «ذخیره نوبت» | ✔ `POST /care/appointments` 201 with `care_item_key=nt_scan` → **returns to `/fa/pregnancy/calendar`** (9b fixed), NT «نوبت داری», next visit «۳ روز دیگه» |
| 15 | Appointment bell off (`PUT /care/appointments/{id} {"is_active":false}`) → calendar / Today | ✔ still «نوبت داری» and still the next visit (see bug L4 for the label) |
| 16 | «ویزیت جدید» → form (`return_to=/pregnancy/calendar`) → ۲۰ مهر 10:00 → save | ✔ 201 → back on the calendar |
| 17 | «گزارش علائم برای پزشک (PDF)» | ✔ 73.8 KB / 72.4 KB, 1 page, Persian digits, footer «ریتمی · صفحهٔ ۱ از ۱» (`c-30`). See Info I1 |
| 18 | Profile → «برگرد به حالت چرخه» (native confirm accepted) → `/fa/home` | ✔ empty state «هنوز پریودی ثبت نشده / تاریخ آخرین پریودت رو ثبت کن», and there is no fake period. **Stage DB: 0 `cycle_histories` rows for uid 14 and 19**, and `user_profiles.last_period_start` is NULL for both |

### Admin `/panel` (light ✔ / dark ✔)

| # | Check | Result |
|---|---|---|
| A1 | All 19 nav screens open (dashboard, users, articles, affirmations, challenges, challenge-completions, task-templates, phase-contents, recommendations, checkup-types, banners, info-sections, pregnancy-weeks, pregnancy-care-plan, pregnancy-alert-rules, messages, languages, account/password, admins) | ✔ they all render, with no raw keys, no broken images and no API errors. ✘ At 390 px, 14 of them scroll horizontally (bug M1) |
| A2 | Week details (week 10, «جزئیات ساختاریافته»): headline + « QA-C» → app `/fa/pregnancy/weeks/10` | ✔ visible in the app (`c-56`). Restored |
| A3 | Care plan `nt_scan` fa title + « QA-C» → app calendar care plan | ✔ visible (`c-57`). Restored |
| A4 | Alert rule `weight_missing_week`: label «از روز هفته (۰ = روز اول)» is present; `from_weekday` 5 → 4 | ✔ written to the fa **and** en rows (behaviour in every locale); nothing else in the payload changed. It is not visible in the app for these users (the rule needs a missing weight on weekday ≥ 5). Restored |
| A5 | `pregnancy_setup/calendar_note` `plan_note` → «QA-C …» → app calendar source note | ✔ visible (`c-57`). Restored |
| A6 | fa digits in admin | ✘ partly: see bug L3. Dashboard / users mobile numbers are Latin in `dir="ltr"` on purpose |

### Account deletion

| # | Check | Result |
|---|---|---|
| D1 | `DELETE /api/v1/account` (uid 19, then uid 14) | ✔ 200 `X-Backend: go`, message in the request locale |
| D2 | Stage DB afterwards | ✔ 0 `oauth_access_tokens` rows for uid 14 / 19 (uid 14 had 1 revoked token before; uid 19 had 1 live token). 0 rows in every table with a `user_id` column, 0 `users` rows |
| D3 | App after deletion | ✔ reload → `/fa/signup`; only `ritme_theme` and `ritme-install-dismissed` are left in localStorage |

### Cross-cutting

| Check | Result |
|---|---|
| `X-Backend: go` on every `/api` response | ✔ 499 `/api/*` responses were recorded. All of them had `X-Backend: go` except the Next.js route handler `POST /api/session/flag` (204, frontend, expected) |
| No unexpected 4xx/5xx | ✔ The only non-2xx responses were expected: 400 `GET /messages/daily` in cycle mode with no period (known, it drives the empty state); 401s right after account deletion and on `GET /api/admin/v1/auth/me` before the admin login; 404 on `GET /api/admin/v1/pregnancy-weeks/10` (I opened that URL by hand, and week 10 has no v1 text row) |
| Console errors / CSP violations | ✔ no JS exceptions, no `console.error` and no CSP issues. Only network-level logs of the 4xx above, plus DevTools a11y hints (`FormEmptyIdAndNameAttributes…`, `FormLabelHasNeither…`) |
| Raw i18n keys | ✔ none on any screen (app or admin) |
| Latin digits in fa | ✘ app: onboarding pregnancy basis (bug L2). Week page «(T-M7-01)» is seed source text (info). Admin: bug L3 |
| Horizontal overflow | ✔ app: none on any screen, light or dark. ✘ admin: bug M1 |
| Design comparison | ✔ Setup (welcome / dating / history / result), Today, Week × 3 tabs, Log + saved, Alerts and Calendar all match `docs/pregnancy-v2/screenshots/audit/*-design.png`, apart from the resolved or won't-fix audit rows (B2 week numbering, toggles instead of checkboxes, C3 content). There is no dark artboard; dark uses the token theme and looked right |

## Bugs

| # | Sev | Where | Repro / evidence |
|---|---|---|---|
| **M1** | med | `admin-web/src/app/globals.css:359` (`.table-wrap { overflow-x: auto; }`), used by `admin-web/src/shared/ui/DataTable.tsx:51` | At 390 px wide, 14 admin list screens get a page-level horizontal scroll: users, articles, affirmations, challenges, task-templates, recommendations, checkup-types, banners, info-sections, pregnancy-care-plan, pregnancy-alert-rules, messages, languages and admins. The document grows to 461–972 px, `innerWidth` becomes 852 on messages, and the whole shell (topbar included) shifts. **Root cause:** `.table-wrap` has no containing block, so the absolutely positioned `.sr-only` in a header cell (the actions column) escapes the scroll container and widens the page. Injecting `.table-wrap{position:relative}` brings it back to 390 (checked live). Evidence: `c-41`, `c-50-admin-*-light`, `c-60-admin-*-dark` |
| L1 | low | `backend-go/db/queries/pregnancy/v2_read.sql:9-10` (`GetV2TodayExtras.unread_alerts`) vs `internal/messages/pregnancyalerts/engine.go:33,526` (list = `v2:` rows of the last 7 days) | After a spotting log the Today badge says «هشدارها، ۳ پیام تازه», but Alerts shows only 2 cards. The count also includes the v1 `symptom_based` row that the legacy log hook writes. Seen in light and dark. Once Alerts is opened, `read-all` clears everything |
| L2 | low | `frontend/src/screens/onboarding-pregnancy-basis/ui/PregnancyBasisPage.tsx:133,144` (`NumberField`, `type=number`) | Onboarding → «باردارم» → «خودم هفته رو وارد می‌کنم» → type 9: the field shows a Latin «9». Setup v2 already uses `LocaleNumberField` (T-M7-20) |
| L3 | low | admin-web ICU args passed as raw numbers: `screens/pregnancy-weeks/ui/WeekDetailsForm.tsx:610`, `screens/pregnancy-alert-rules/ui/AlertRuleFormScreen.tsx:188,327`, `screens/messages/ui/SchemaForm.tsx:196`, `screens/checkup-types/ui/CheckupTypeFormScreen.tsx:397`, `screens/checkup-types/ui/CheckupTypesScreen.tsx:262,279`, `screens/pregnancy-care-plan/ui/CareItemFormScreen.tsx:162` + `CarePlanScreen.tsx:216`; preview sample `screens/pregnancy-alert-rules/lib/level.ts:34` | fa admin shows «1 از 10», «(2 از 4)», «1 تا 42», «0 تا 6», «ثبت‌های 30 روز اخیر», «1 نوبت به این مورد اشاره دارد…» and «در هفتهٔ 12 …» in the alert preview. Number inputs also show Latin digits. Screens that use `n()` (coverage «۰ از ۴۰», ids) are fine |
| L4 | low | `frontend/src/screens/pregnancy-calendar/ui/PregnancyCalendarPage.tsx:352-353,422` | Book NT → open the appointment → turn the bell off → back on the calendar, the next-visit card still says «یادآور: ۱ روز قبل» (the label ignores `is_active`). The cycle home reminders card also shows «۱ روز قبل یادآوری» for the bell-off appointment |
| L5 | low | `backend-go/internal/messages/pregnancyalerts` (`week_entered` persisted text/`fact_date`) | Sign up with manual dating (week 10 entered on ۵ مهر) → Setup v2 with LMP (week 10 started ۳ مهر). Alerts still shows «مبنای محاسبه هنوز ورود دستیه…» and fact date ۵ مهر ۱۴۰۵, because the alert text was frozen before re-dating |
| L6 | low | `frontend/src/screens/pregnancy-log/ui/DayLogPage.tsx:478` | The «این ثبت یک پیام تازه ساخت» box shows «پیگیری زودتر» as a neutral violet pill, while Alerts / the legend colour that level pink. The level colour is not carried over |
| L7 | low | admin-web week grid `screens/pregnancy-weeks` (links to `/pregnancy-weeks/new?week=N`) | Stage has 0 `pregnancy_weekly_content` rows, so all 40 cells read «افزودن» (coverage «۰ از ۴۰»), although structured details exist for weeks 1–42. Opening `/panel/pregnancy-weeks/10` directly gives «مورد خواسته‌شده پیدا نشد» with the structured tab disabled (`c-51`). The details are only reachable through the «افزودن» (create) page |
| L8 | low | admin save path (messages, care items, week details) | Every admin save rewrites the JSON columns with `\uXXXX` escapes (seed rows are raw UTF-8). The meaning is unchanged (checked with `jq -S`), but the stored size is about 6× larger and `LIKE '%فارسی%'` searches on the DB stop matching. Likely json_encode parity; flagging it only |
| **M2** | med (product) | `frontend/src/features/manage-account` — `DeleteAccountConfirm` is exported but mounted nowhere (the import was dropped from `ProfilePage` in `42f9b94` "pwa") | The web app has no in-app way to delete an account. `DELETE /account` works (D1–D3), but a user cannot reach it. This matters for store policy and privacy. It may be a deliberate product decision; please confirm |

### Info

- **I1:** In headless Chrome, `navigator.canShare({files})` is true, and `share()` never settles, so the PDF button stays on «در حال ساخت PDF…». This is a test-environment artefact. With `canShare` stubbed the download path works. If a real browser ever leaves `share()` pending, the button has no timeout (`shared/lib/pdf/share.ts:9-16`).
- **I2:** The Week page's «منابع» shows the seed `[needs review] … (T-M7-01)` with Latin digits. This is placeholder content that needs clinical sign-off (T-M7-15), not UI.
- **I3:** The trimester start dates on Today (2nd = 13 completed weeks, 3rd = 28) follow `calc.Trimester`. They are consistent with the engine but asymmetric relative to week numbering. I'm noting it for the clinical review.

## Restore / cleanup

- Before any edit I took a byte snapshot (md5 + `updated_at`) of every `pregnancy%` `message_contents` row, every
  `pregnancy_week_details` row and every `pregnancy_care_items` row.
- Edited rows: `message_contents` 279 (welcome), 267/268 (`weight_missing_week` fa/en), 295 (`calendar_note`),
  `pregnancy_care_items` 2 and `pregnancy_week_details` week 10.
- Each was put back byte-exactly from the snapshot. A plain admin re-save would leave `\u` escapes (L8) and a new
  `updated_at`, so the rows were restored with a hex-literal UPDATE on the stage DB. The final snapshot diff is
  **identical, including `updated_at`**, and the app no longer shows «QA-C».
- Test users 14 and 19 were deleted through `DELETE /account`, and their OTP rows were deleted. A sweep of every
  `user_id` table found 0 rows.
- I logged out of the admin, stopped Chrome and the driver, and deleted the profile, the creds files and the gate cookie
  from the scratchpad.

## Screenshots (`docs/qa/screenshots/c/`, 585 px, pngquant, 110 files, 4.8 MB)

- **App:**
  - `c-01` signup
  - `c-02` onboarding basis
  - `c-03` Today after onboarding
  - `c-05`–`c-11` Setup v2 (`c-06` = admin-edited welcome)
  - `c-12`/`c-13` Today (+ carousel next)
  - `c-14`–`c-16` Week tabs
  - `c-17`–`c-22` Log (filled, saved, offline queued, synced, spotting)
  - `c-23`/`c-24` Alerts (+ acked)
  - `c-25`–`c-29a` Calendar, booking, bell off, new visit
  - `c-30` PDF page 1
  - `c-31` Profile
  - `c-32` cycle home empty state
  - `c-56`/`c-57` admin edits visible in the app
  - Each step has `-light` and `-dark`, except `c-03`, `c-06`, `c-07`, `c-13`, `c-21`, `c-30`, `c-56`, `c-57` (light
    only) and `c-28a`, `c-29a` (dark only).
- **Admin:**
  - `c-40`–`c-42`, `c-50-admin-*-light` (19 nav screens), `c-51`–`c-55` (editors light)
  - `c-60-admin-*-dark` (19 nav screens + the 4 pregnancy editors)
  - List screens that overflow (M1) are captured as the browser lays them out, shifted sideways.
