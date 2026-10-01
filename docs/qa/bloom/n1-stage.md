# N1 stage smoke — B-N1-17

| | |
|---|---|
| Date | 2026-10-01 |
| Target | https://stage.ritmeapp.ir (staging only — production untouched) |
| Commit deployed | `d7bc901` (`stage` branch tip at deploy: "fix(ui): N1 design-fidelity audit fixes (B-N1-16)") |
| Deploy | `./deploy-stage.sh` → exit 0, all built-in checks ok |
| Migration line | `{"level":"INFO","msg":"migrations","action":"goose_managed","applied":[9,10,11,12,13],"version":13}` — up to `00013` applied |
| Backend logs during smoke | `ritme-stage-backend-go-1`: 0 `ERROR` lines, no 5xx; only 3× 401 (the expected `/api/admin/v1/auth/me` probe on the admin login page) |
| Method | Headless Chrome over CDP, 390×844 @2x, `fa`, light + dark (`prefers-color-scheme` + `ritme_theme`); gate cookie + real OTP signup (OTP read read-only from stage MariaDB — stage has no test-OTP mode); each page: full-page screenshot + CDP Network (≥400) and console/exception capture |
| Screenshots | `docs/qa/bloom/n1-stage/` (`<user>-<route>.<theme>.png`) |

## Test users (created via the public signup/OTP flow)

| | Number | Path | State reached |
|---|---|---|---|
| A | `0990•••••01` | onboarding → «فعلاً قصد بارداری ندارم» | 1) period day 1 (onboarding default last-period = today) → 2) three periods via `/cycle/period` API, today = cycle day 15 (fertile window) |
| B | `0990•••••02` | onboarding (dark) → «دارم برای بارداری تلاش می‌کنم» | two periods via API, today = cycle day 27, "period in 2 days" (near period / luteal) — TTC home |
| C | `0990•••••03` | onboarding → «باردارم» → manual 20w3d | pregnancy mode, week 21, 137 days to EDD |

Home states covered: **during period** (A-home), **mid-cycle / fertile** (A2-home), **near period** (B-home, TTC variant), plus pregnancy home (C-pregnancy) and the no-period home (C-home).

## Checks

Network/console column = 4xx/5xx responses and console errors captured on that page (both themes).

| Page | Light | Dark | Result | Network / console | Notes |
|---|---|---|---|---|---|
| `/fa/signup` | `01-signup.light`, `C-01-signup.light` | `B-01-signup.dark` | PASS | clean | install prompt shows on first visit (expected) |
| `/fa/otp` | `02-otp.light` | `B-02-otp.dark` | PASS | clean | 4-box OTP, verify enables after 4 digits |
| Onboarding cycle path (name → birthday → weight → height → intention → period-len → cycle-duration → cycle-len → conditions → setting-up) | `04-onb-*.light` | `04-onb-*.dark` | PASS | clean | terms checkbox gates «ادامه»; lands on `/fa/home` |
| Onboarding pregnant path (pregnancy-basis manual → conditions → setting-up) | `C-04-onb-*.light` | — | PASS | clean | lands on `/fa/pregnancy`, week 21 / 137 days correct for 20w3d |
| Home — during period | `A-home.light` | `A-home.dark` | PASS | clean | ring «روز ۱», «پریودم تموم شد», predictions card |
| Home — fertile window (day 15) | `A2-home.light` | `A2-home.dark` | PASS | clean | «پریود بعدی ۱۴ روز دیگر», chance «خیلی زیاد» |
| Home — TTC near period (day 27) | `B-home.light` | `B-home.dark` | PASS | clean | «پریود تا ۲ روز دیگر», luteal pill active, LH/BBT/sex tiles |
| Calendar — month | `A-calendar`, `A2-calendar`, `B-calendar` `.light` | same `.dark` | PASS | clean | TTC user gets «تقویم باروری» |
| Calendar — year | `A2-calendar-year.light` | `A2-calendar-year.dark` | PASS (see O-1) | clean | |
| `/fa/log` | `A-log.light` | `A-log.dark` | PASS | clean | |
| `/fa/cycle` (history) | `A-cycle`, `A2-cycle` `.light` | `.dark` | PASS | clean | empty state → after 3 periods: 28 d median, ±2, «منظم» |
| `/fa/cycle/symptoms` | `A-cycle_symptoms.light` | `.dark` | PASS | clean | «needs 3 cycles» empty state |
| `/fa/cycle/settings` | `A-cycle_settings.light` | `.dark` | PASS | clean | see B-1 (PMS day differs from notifications) |
| `/fa/profile` (me hub) | `A-profile`, `B-profile`, `C-profile` `.light` | `.dark` | PASS | clean | mode chip follows intention (چرخه / اقدام به بارداری / بارداری), masked phone |
| `/fa/profile/notifications` | `A-profile_notifications.light` | `.dark` | PASS (B-1) | clean | |
| `/fa/profile/privacy` | `A-profile_privacy.light` | `.dark` | PASS | clean | |
| `/fa/profile/support` | `A-profile_support.light` | `.dark` | PASS | clean | FAQ empty on stage (O-2) |
| Support report e2e (user → admin inbox) | `C-support-report-sheet`, `C-support-report-sent` `.light` | — | PASS | clean | report sent from C appears in `/panel/support-reports` (sender masked, app version 1.1.0) |
| `/fa/profile/about` | `A-profile_about.light` | `.dark` | PASS | clean | v1.1.0 |
| TTC `/fa/fertility/log` | `B-fertility_log.light` | `.dark` | PASS | clean | |
| TTC `/fa/fertility/bbt` | `B-fertility_bbt.light` | `.dark` | PASS | clean | empty state |
| TTC `/fa/fertility/insights` | `B-fertility_insights.light` | `.dark` | PASS | clean | «بر اساس ۱ سیکل» (2 periods = 1 complete cycle — correct) |
| `/fa/pregnancy` | `C-pregnancy.light` | `.dark` | PASS | clean | EDD ۲۶ بهمن ۱۴۰۵ |
| `/fa/pregnancy/weeks`, `/weeks/21` | `C-pregnancy_weeks*.light` | `.dark` | PASS | clean | |
| `/fa/pregnancy/log` | `C-pregnancy_log.light` | `.dark` | PASS | clean | |
| `/fa/pregnancy/calendar` | `C-pregnancy_calendar.light` | `.dark` | PASS | clean | |
| `/fa/pregnancy/alerts` | `C-pregnancy_alerts.light` | `.dark` | PASS | clean | |
| `/fa/pregnancy/setup` (already pregnant) | `C-pregnancy_setup-landed-home.light` | `C-pregnancy_setup.dark` | WARN | clean | B-3 |
| `/fa/home` while in pregnancy mode | `C-home.light` | `C-home.dark` | WARN | clean | B-2 |
| `/fa/reminders` | `A-reminders`, `C-reminders` `.light` | `.dark` | PASS | clean | |
| `/fa/reminders/medication/new`, `/appointment/new` | `C-reminders_*_new.light` | `.dark` | PASS | clean | |
| `/fa/checkups` | `A-checkups`, `C-checkups` `.light` | `.dark` | PASS | clean | cycle 2/6, pregnant 0/4 |
| `/fa/checkups/history` | `C-checkups_history.light` | `.dark` | PASS | clean | |
| `/fa/checkups/self-exam` | `A-checkups_self_exam.light` | `.dark` | PASS | clean | for C (pregnant, no breast self-exam in catalog) shows «این چکاپ پیدا نشد» — only reachable by URL, not a bug |
| `/fa/services` | `C-services.light` | `.dark` | PASS | clean | all «به‌زودی» |
| admin-web `/panel/login` | `ADM-login.light` | — | PASS | 401 `/api/admin/v1/auth/me` (expected unauthenticated probe) | admin-web ignores `prefers-color-scheme` (own toggle; admin theme is N9) |
| admin-web `/panel` dashboard | `ADM-dashboard.light` | — | PASS | clean | mobiles masked in the screenshot before capture |
| admin-web `/panel/support-reports` | `ADM-support-reports.light` | — | PASS | clean | shows the QA report from user C |

**Summary: 38 checks — 36 PASS, 2 WARN, 0 FAIL.** No 4xx/5xx and no console errors on any app page in either theme; dark mode renders on every screen with no unreadable/white-flash areas found.

## Bugs / findings

| ID | Severity | Finding | Repro | Suspected file |
|---|---|---|---|---|
| B-1 | Low | PMS reminder day disagrees between screens: `/profile/notifications` always says «شروع PMS · روز ۲۴ سیکل», `/cycle/settings` shows the API value «روز ۲۶ سیکل» for the same user. | User A (28-day cycle): open both pages and compare the PMS row. | `frontend/messages/fa/me.json:137` (static `"sub": "روز ۲۴ سیکل"`, same in `en/me.json`) vs `frontend/src/screens/cycle-settings/ui/CycleSettingsPage.tsx:193` (uses `cycle_day`). The notifications screen should read the PMS `cycle_day` from the reminder settings like cycle settings does. |
| B-2 | Low | In pregnancy mode, opening `/fa/home` directly renders the cycle home with «هنوز پریودی ثبت نشده — ثبت پریود» instead of sending the user to `/fa/pregnancy`. Bottom nav/splash route correctly, so only deep links / stale tabs hit it. | User C (pregnant): navigate to `/fa/home`. | `frontend/src/screens/home/ui/HomePage.tsx` (no pregnancy-mode redirect; the no-period branch ~L735 assumes "just left pregnancy mode") |
| B-3 | Low | `/fa/pregnancy/setup` has no guard for a user already in pregnancy mode: it shows the 4-step activation wizard (reproduced 3/3). Once, on the first visit in a sweep, it landed on `/fa/home` instead (not reproducible afterwards). | User C: navigate to `/fa/pregnancy/setup`. | `frontend/src/screens/pregnancy-onboarding/ui/PregnancyOnboardingPage.tsx` |
| O-1 | Info | Calendar month/year views paint predicted periods/fertile windows for months **before** the first logged period (e.g. Farvardin–Tir for a user whose first period is 1 Mordad; Shahrivar for a brand-new user). `/cycle/month/2026/4` returns `cycle_day`/`pms` for April. This is the deliberately ported PHP backward-extrapolation quirk, not an N1 regression — product call whether to blank pre-history. | User A → `/fa/calendar` → «سال». | `backend-go/internal/cycle/legacy/engine.go:212` |
| O-2 | Info | Support FAQ is empty on stage («هنوز سؤالی ثبت نشده است») — content, not code. | `/fa/profile/support` | stage DB content (admin) |
| O-3 | Info | Period create/update responses carry the message «پروفایل با موفقیت بروزرسانی شد» (Laravel-compat copy); not shown in the UI. | `POST /api/v1/cycle/period` | `backend-go/internal/cycle/periods` |

## Not covered

- Locale switch / `en` (task asks for `fa`).
- Logging a period through the UI buttons — states were set through the same public API the app uses (`PUT/POST /api/v1/cycle/period`).
- admin-web dark theme (admin theme is N9 scope).
- Android — never (bloom decision).

Test data left on stage: users `0990•••••01/02/03` and one support report marked as QA («تست اسموک N1 … لطفاً نادیده بگیرید»).
