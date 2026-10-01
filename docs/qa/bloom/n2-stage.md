# N2 stage smoke — B-N2-11

| | |
|---|---|
| Date | 2026-10-01 |
| Target | https://stage.ritmeapp.ir (staging only — production `/opt/ritme` / compose `ritme` untouched) |
| Commits deployed | backend-go `9baafc1` ("fix(ui): N2 design-fidelity audit fixes and --on-danger token (B-N2-10)"); frontend build rev `46fdb93` ("feat(log): log taxonomy v2 entries … (B-N3-01)") |
| Deploy | `./deploy-stage.sh` → exit 0 (run by the orchestrator before this smoke) |
| Migration line | `{"level":"INFO","msg":"migrations","action":"goose_managed","applied":[14,15,17,18,19],"version":19}` — `00016` intentionally unused, `00020` (B-N3-01) not in the deployed backend |
| Stage env (non-secret) | `APP_ENV=staging`, `PAYMENT_PROVIDER=fake`, `PLUS_TRIAL_DAYS=7` |
| Backend logs during smoke | `ritme-stage-backend-go-1`, last 2 h: **0 `ERROR`/`WARN` lines, 0 5xx, no panics**. 4xx only: 7× 401 `/api/admin/v1/auth/me` (admin login page probe), 7× 400 `/messages/daily` (no-period users, O-1), 5× 404 `/pregnancy/profile` (pregnancy setup before activation, O-2), 1× 422 `/plus/verify` (simulated gateway failure — expected), 1× 422 `/plus/checkout` (deactivated code rejected — expected). Payment audit: 4× `payments: create`, 4× `payments: verify`; 2× `admin audit`. `ritme-stage-frontend-1` / `ritme-stage-admin-web-1`: no error lines. |
| Method | New `bloom/bin/stage-smoke.mjs` (headless Chrome over CDP, 390×844 @2x, `fa`, light + dark via `prefers-color-scheme` + `ritme_theme`; admin 1440×900 + `ritme_admin_theme`). Gate cookie + admin creds read over ssh at runtime, OTP read read-only from `ritme_stage`. Click flows scripted with the module's exported helpers (real UI clicks/typing, not API shortcuts). Each page records CDP Network ≥ 400 and console errors / exceptions → `n2-stage/summary.{json,md}` (147 captures, 0 console errors, 0 exceptions). |
| Screenshots | `docs/qa/bloom/n2-stage/` (`<user>-<step>.<theme>.png`; phone numbers masked in the page before capture; the 5 signup shots had the typed number covered afterwards) |

## Test users (created via the public signup / OTP flow)

| | Number | Branch | Theme | End state |
|---|---|---|---|---|
| S1 | `0990•••••47` | woman → «پیگیری سیکل» (last period picked in calendar) | light | mode switches + pregnancy enter/normal exit; trial → paid 3-month (trial offer) → cancelled |
| S2 | `0990•••••16` | woman → «اقدام به بارداری» | dark | mode switches back to TTC; paid 3-month with code `QAN2SMOKE` → cancelled |
| S3 | `0990•••••41` | woman → «باردارم» → «هفته فعلی» 20w | light | pregnancy home → calm loss exit (dark) → cycle mode |
| S4 | `0990•••••25` | woman → «یائسگی» → پیش‌یائسگی | dark | menopause home; failed payment, then paid 1-month (no code), active |
| S5 | `0990•••••29` | man → «کد همدم» stub → «بعداً وصل می‌شوم» | light | lands on cycle home (B-1) |
| — | `0990•••••02/74/96/91` | aborted runs (driver arg bug) | — | stuck at `/onboarding/intention`; harmless leftovers |

## Checks

Network/console = 4xx/5xx and console errors captured on that page.

| Page / flow | Light | Dark | Result | Network / console | Notes |
|---|---|---|---|---|---|
| Signup `/fa/signup` (+98, two consent ticks) | `S1/S3/S5-01-signup` | `S2/S4-01-signup` | PASS | clean | «دریافت کد» enables only after both ticks |
| OTP `/fa/otp` | `S1-02-otp` … | `S2-02-otp` … | PASS | clean | masked number, resend timer |
| Onboarding name → gender → goal | `S1-03…05` | `S2-03…05` | PASS | clean | progress bar per step |
| Branch: cycle (calendar + steppers) → conditions → health → ready | `S1-06…09` | `S2-06…09` (TTC) | PASS | clean | ready card: next period, fertile window, mode, PMS reminder |
| Branch: pregnancy (`/onboarding/pregnancy-basis`, week basis) | `S3-06-pregnancy` | — | PASS | clean | lands on pregnancy home week 21, 140 days, EDD ۲۹ بهمن ۱۴۰۵ (correct for 20w0d) |
| Branch: menopause (`/onboarding/menopause`) | — | `S4-06-menopause` | PASS | clean | `life_stage.mode=menopause`, stage `peri` stored |
| Branch: male → partner stub | `S5-04-gender`, `S5-05-partner` | — | PASS (stub) / see B-1 | clean | connect disabled, «به‌زودی» note |
| Landing per branch | `S1-10` cycle home, `S3-10` pregnancy, `S5-10` | `S2-10` TTC home, `S4-10` menopause home | PASS / WARN | 400 `/messages/daily` on S4/S5 (O-1) | onboarding API confirms goal / mode for every branch |
| `/profile/mode` cycle → menopause → teen → cycle | `M1-mode-*`, `M1-home-menopause`, `M1-home-teen`, `M1-home-cycle-back` | `M2-mode-*`, `M2-home-*` (TTC → … → TTC) | PASS | clean | `GET /profile/life-stage` matches after each switch; «حالت «…» فعال شد» live text |
| Teen: home + Me hub | `M1-home-teen`, `M1-profile-teen` | `M2-home-teen`, `M2-profile-teen` | PASS (B-3) | clean | no banners/fertility rows/Plus card; `/fa/plus` redirects teen to `/fa/profile` |
| Menopause home | `M1-home-menopause`, `R4-fa_home` | `S4-10-landing`, `M2-home-menopause`, `R4-fa_home` | PASS (B-4) | 400 `/messages/daily` (O-1) | hero «امروز چطوری؟», quick tiles, «روند علائم» → `/cycle/symptoms` (`R4-fa_cycle_symptoms`) |
| Pregnancy enter from mode switcher (confirm sheet → 4-step `/pregnancy/setup`, LMP basis) | `M1-mode-pregnancy-confirm`, `M1-pregnancy-setup-1…4`, `M1-pregnancy-after-setup` | — | PASS | 404 `/pregnancy/profile` before activation (O-2) | 8w5d, EDD ۱۸ اردیبهشت ۱۴۰۶ (correct for LMP 10 Mordad); `/fa/home` now redirects to `/fa/pregnancy` (N1 B-2 fixed) |
| Pregnancy normal exit (leave sheet «تغییر حالت») | `M1-mode-in-pregnancy`, `M1-mode-leave-confirm`, `M1-home-after-leave` | — | PASS | clean | mode cycle, pregnancy profile kept with `pregnancy_mode=false` |
| Calm loss exit `/profile/mode/loss` | — | `M3-mode-pregnancy`, `M3-loss`, `M3-loss-done`, `M3-home-after-loss` | PASS | clean | no celebration; 115 note; mode → cycle; revisiting shows «حالت بارداری روشن نیست» (`M3-loss-not-pregnant`) |
| Paywall `/fa/plus` | `P1-paywall` | `P2-paywall` | PASS | clean | |
| Trial start → success | `P1-trial-started` | — | PASS | clean | «دوره رایگان تا ۱۶ مهر ۱۴۰۵» (7 days) |
| Trial banner on home + trial sheet | `P1-home-trial-banner`, `P1-home-trial-full`, `P1-trial-sheet` | — | PASS | clean | live countdown 06:23:59, 50% offer, featured 3-month plan |
| Plans `/fa/plus/plans` | `P1-plans` | `P2-plans` | PASS | clean | |
| Checkout without code (offer price during trial) | `P1-checkout` | — | PASS | clean | 237,000 − 118,500 offer + 10% VAT = 130,350 T |
| Checkout without code (no trial) | `P4-checkout-nocode` | — | PASS | clean | 1-month |
| Checkout with code `QAN2SMOKE` (30%) | `P1-checkout-code-applied` (in trial) | `P2-checkout`, `P2-checkout-code-applied` | PASS (S2) / WARN (S1, B-2) | clean | S2: −71,100 T, total 182,490 T, invoice carries the code |
| Fake gateway page → success → return → success screen | `P1-fake-gateway`, `P1-after-gateway`, `P4-fake-gateway`, `P4-success` | `P2-fake-gateway`, `P2-success` | PASS | clean | amounts on the TEST page match the quote; subscription end ۱۱ دی / ۱۰ آبان correct |
| Fake gateway «Simulate failure» | `P4-gateway-failed` | — | PASS | 422 `/plus/verify` (expected) | «پرداخت انجام نشد» + restore hint; retry then succeeded |
| Manage `/plus/manage` + payment history + cancel | `P1-manage`, `P1-cancel-confirm`, `P1-manage-cancelled`, `P4-manage`, `R1-fa_plus_manage` | `P2-manage`, `P2-cancel-confirm`, `P2-manage-cancelled`, `R1-fa_plus_manage` | PASS | clean | cancel keeps Plus until period end, `auto_renew=false`, `status=canceled` |
| Me hub Plus card (free / plus) | `P1-profile-free`, `P4-profile-plus` | `P2-profile-plus` | PASS | clean | |
| Admin: create discount code in `/panel/plus/discount-codes/new` | `ADM-discount-new-filled`, `ADM-discount-codes-list` | — | PASS | clean | |
| Admin: deactivate code (cleanup) | `ADM-discount-deactivate-confirm`, `ADM-discount-codes-after-deactivate` | — | PASS | clean | used once (1 of 5) → «غیرفعال»; checkout with it now 422 `discount_invalid` |
| Admin `/panel/plus/subscriptions` | `ADM-plus_subscriptions` | same `.dark` | PASS | clean | 3 rows, masked mobiles, auto-renew state |
| Admin `/panel/plus/payments`, `/payments/1` | `ADM-plus_payments`, `ADM-plus_payments_1` | same `.dark` | PASS | clean | amounts/VAT/discount match invoice, refund buttons present (not exercised) |
| Admin `/panel/plus/plans`, `/plans/2` | `ADM-plus_plans`, `ADM-plus_plans_2` | same `.dark` | PASS | clean | |
| Admin `/panel/plus/discount-codes` | `ADM-plus_discount-codes` | same `.dark` | PASS | clean | |
| Admin `/panel/plus/settings` | `ADM-plus_settings` | same `.dark` | PASS | clean | |
| N1 regression: home | `R1-fa_home` | `R1-fa_home` | PASS | clean | |
| N1 regression: calendar | `R1-fa_calendar` | `R1-fa_calendar` | PASS | clean | |
| N1 regression: cycle settings | `R1-fa_cycle_settings` | `R1-fa_cycle_settings` | PASS | clean | |
| N1 regression: notifications | `R1-fa_profile_notifications` | `R1-fa_profile_notifications` | PASS | clean | PMS «روز ۲۶ سیکل» now equals cycle settings (N1 B-1 fixed) |
| N1 regression: privacy | `R1-fa_profile_privacy` | `R1-fa_profile_privacy` | PASS | clean | |

**Summary: 39 checks — 35 PASS, 4 PASS-with-finding / WARN, 0 FAIL.** No 5xx, no console errors or uncaught exceptions on any page in either theme; dark mode readable on every captured screen. N1 bugs B-1 (PMS day), B-2 (pregnancy deep link to /home) and B-3 (setup guard path) verified fixed on stage.

## Bugs / findings

| ID | Severity | Finding | Repro | Suspected file |
|---|---|---|---|---|
| B-1 | Medium | Man → partner stub → «بعداً وصل می‌شوم» → «ورود به ریتمی» lands on the **women's cycle home**: «هنوز پریودی ثبت نشده — ثبت پریود», breast self-exam and Pap smear checkups, period tab. Stored `life_mode` null → resolves to `cycle`. | New user, gender «مرد», partner later → enter. (`S5-10-landing.light`) | `frontend/src/screens/onboarding-flow/model/flow.ts` `landingRoute()` (goal null → `/home`); `frontend/src/screens/home/ui/HomePage.tsx` (no male branch). Product: until B-N4-05, show a partner placeholder instead of the cycle home. |
| B-2 | Low | During a trial the 50% offer beats a 30% code (by design: larger discount wins, no stacking — `backend-go/internal/plus/service.go:325`), but the checkout still shows «کد تخفیف … اعمال شد» with no code line and no explanation; invoice has `discount_code=NULL`. User believes the code was used. | S1 in trial → `/fa/plus/checkout?plan=2` → enter an active code → «اعمال». (`P1-checkout-code-applied.light`) | `frontend/src/screens/plus-checkout/ui/CheckoutPage.tsx:63` (`codeAccepted` = any successful quote; should check `discount_source === 'code'` and say the offer is better) |
| B-3 | Low | Teen home phase card still talks about fertility: «بر اساس پیش‌بینی چرخه، پنجره باروری از حدود ۱ روز دیگر شروع می‌شود» (fertility rows/markers are hidden elsewhere). | Switch to «نوجوان» → `/fa/home`. (`M1-home-teen.light`, `M2-home-teen.dark`) | `frontend/src/screens/home/ui/HomePage.tsx:725` (phase description from `/messages/daily` `primary.shortMessage`); message engine maps teen → cycle (`backend-go/internal/messages`) |
| B-4 | Low | Menopause home «چکاپ‌های دوره‌ای» shows cycle-day timing — «خودآزمایی سینه · روز ۷ تا ۱۰ سیکل», «پاپ‌اسمیر · روز ۱۰ تا ۲۰ سیکل» — for a user with no cycle tracking. | S4 (menopause) → `/fa/home`. (`S4-10-landing.dark`) | `frontend/src/widgets/checkups-card/ui/CheckupsCard.tsx` / `backend-go/internal/checkups` (catalog timing not mode-aware) |
| B-5 | Low | Me hub footer shows «نسخه ۱.۰.۰» while the app is 1.1.0 (`package.json`, `/version.json`). | `/fa/profile`, bottom. (`M2-profile-teen.dark`) | `frontend/src/screens/profile/ui/ProfilePage.tsx:35` (hard-coded `APP_VERSION = '1.0.0'`) |
| O-1 | Info | `GET /messages/daily` answers 400 «لطفاً ابتدا پروفایل خود را تکمیل کنید (تاریخ آخرین پریود)» for menopause, male and post-loss users; the client treats it as "no message" (`entities/message/api/queries.ts`), so only console-network noise. | S4/S5 home | `backend-go/internal/messages` (Laravel-compat 400) |
| O-2 | Info | `GET /pregnancy/profile` 404 while `/pregnancy/setup` runs before activation — handled. | Mode switcher → pregnancy | — |
| O-3 | Info | Success screen after purchase lists «تحلیل کامل سیکل‌هایت» also for a menopause user. | S4 → buy → success (`P4-success.light`) | `frontend/src/screens/plus-success` (copy not mode-aware) |

## Not covered

- **Trial expiry** — needs the test clock (`X-Test-Now`), which is only enabled for `APP_ENV` local/testing/contract; stage is `staging`. Covered by B-N2-06 unit/int tests only.
- Admin refunds (gateway / manual) and subscription «تمدید» — buttons rendered, not clicked (would mutate paid test data; refund path tested in B-N2-09).
- Restore purchase («بازیابی خرید»), plan upgrade from manage.
- Locale `en`, Android (never), admin-web theme fidelity (N9).

## Test data left on stage

Users `0990•••••47/16/41/25/29` (+ four stuck at onboarding: `•••••02/74/96/91`), 4 fake invoices (3 paid, 1 failed; subscriptions: 2 cancelled, 1 active), one trial (S1), discount code `QAN2SMOKE` (**deactivated**, used once).
