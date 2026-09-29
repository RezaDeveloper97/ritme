# Stage final smoke 2026-09-29

This run deployed `stage` @ `63deeac` to `https://stage.ritmeapp.ir` (compose project `ritme-stage`). It then smoke-tested
every fix since `e3aea8d`:
- T-M2-35: regression A and B
- T-M7-22: regression C
- T-M7-23: last polish

Production was not touched apart from the graceful proxy reload that the deploy script does itself. No code was changed
and nothing was committed.

## Result

**PASS with 1 finding (medium).** 27 of the 28 checklist items pass. One item is only partly fixed: B-3 is fixed in
`/messages/daily`, but the home «توصیه‌های امروز» section still shows «روزهای اوج باروری…» on O+1 to an avoiding user
(bug F-1).

## 1. verify-all (before deploy)

| Command | Result |
|---|---|
| backend-go `go vet ./... && go test ./... && golangci-lint run` | ✔ `0 issues.` |
| backend-go `make schema-diff` | ✔ `OK — laravel and goose baseline are identical (47 tables, seed rows equal)` |
| backend-go `make contract ROUTES=all` | ✔ `998 passed, 0 failed, 714 allow-listed differences` (16 groups) |
| frontend `typecheck, lint, fsd:lint, lint:styles, lint:dark, test` | ✔ 90 files / 699 tests |
| admin-web `typecheck, lint, fsd:lint, test` | ✔ 30 files / 91 tests |
| Laravel `backend/` | skipped (`backend/` unchanged since `e3aea8d`) |

## 2. Deploy evidence

- Ran `./deploy-stage.sh` (branch `stage` @ `63deeac`). It exited 0 and ended with «✅ Staging deploy done».
- **Production containers were identical before and after.** The `Created` timestamps did not change:

  | Container | Created |
  |---|---|
  | `ritme-backend-1` | 2026-08-31T08:37:28Z |
  | `ritme-frontend-1` | 2026-09-01T13:14:46Z |
  | `ritme-mysql-1` | 2026-08-16T13:39:17Z |
  | `ritme-proxy-1` | 2026-09-23T11:32:14Z |
  | `ritme-queue-1` | 2026-08-31T08:37:32Z |
  | `ritme-redis-1` | 2026-08-16T13:39:17Z |

  `diff` of the two snapshots was empty. The proxy was only reloaded gracefully (`nginx -t` ok, `signal process started`).
- **Goose is at version 8**, as before; there are no new migrations (8 files). Log line:
  `{"msg":"migrations","action":"goose_managed","applied":null,"version":8}`.
- **Stage containers were all healthy:**
  - `backend-go` Up (healthy)
  - `admin-web` Up (healthy)
  - `frontend` Up
  - `mysql` and `redis` Up 4 weeks (healthy)
- **The frontend build was not cached.** `[frontend builder 3/3] RUN echo "frontend build rev: 63deeac-20260929T104922Z" && npm run build`
  ran and finished in `DONE 293.3s`. There was no `CACHED` on that step. admin-web was rebuilt too (`DONE 228.7s`).
- **The script's own assertions passed:**
  - `/up` 200 + `X-Backend: go`
  - `/` 401 without credentials
  - `/api/v1/banners`, `/panel/login` and `/api/admin/v1/auth/me` 401
  - manifest and `sw.js` 200
  - `/admin` 301 → `/panel/`
  - `/oauth/token` 404
  - languages served by Go

## 3. Method

- **Browser:** headless Chrome 154 on CDP port 9310 with its own profile, in a private scratch folder `final/`.
- **Driver:** a small Node CDP driver. Every CDP call had a 20 s timeout, and every shell call had a perl `alarm`.
- **Viewport:** 390×844 @1.5 (585 px PNGs), mobile + touch, locale `fa`, light and dark (`ritme_theme`).
- **Gate:** one Basic-auth page load (curl with a chmod-600 netrc; credentials never printed) minted the `ritme_stage`
  cookie. That cookie was then set in Chrome.
- **Login:** `send-otp`, then the code was read from `otp_verifications` in the stage DB, then `verify-otp`. The token
  went into `localStorage.ritme_token` + `ritme_auth=1`.
- **Admin:** login was typed from a chmod-600 file.

**Test users.** All were deleted afterwards.

| Mobile | uid | State |
|---|---|---|
| 09900001401 | 22 | Trying (TTC). Periods 06-26, 07-24, 08-21, 09-18. Cycle day 12, O = 2026-10-02 |
| 09900001402 | 23 | Avoiding. Periods 06-22 … 09-14. Cycle day 16 = **O+1** (`post_ovulation`, level low) |
| 09900001403 | 24 | Care user with 3 items: folic acid daily 08:00; a cancelled appointment (Dr Ahmadi, 10-07); an ultrasound on 09-30 18:00 with the bell turned **off** |
| 09900001404 | 25 | Pregnant: LMP 2026-07-21, week 11. Spotting (moderate) logged today |
| 09900001405 | 26 | Throwaway: PWA banner, export, and delete account through the UI |

## 4. Checklist

| # | Check | Result |
|---|---|---|
| 1 | Calendar day sheet chance word = Log / `/fertility/days` word (1401) | ✔ ۵ مهر «متوسط» = medium, ۷ مهر (today) «متوسط», ۹ مهر «زیاد» = high, ۱۰ مهر «خیلی زیاد» = peak, ۱۲ مهر «کم» = low. Log for 09-27 says «شانس بارداری در این روز / متوسط». Dark checked too (`f-04`) |
| 2 | PMS 3 days on the calendar | ✔ aria labels «پی‌ام‌اس» on ۲۱, ۲۲, ۲۳ مهر only |
| 3 | PMS 3 days on the home timeline | ✔ «دوره PMS ۲۱ مهر تا ۲۳ مهر» (1403: «۱۷ مهر تا ۱۹ مهر», also 3 days) |
| 4 | PMS 3 days in the insights strips | ✔ 3 violet dots per cycle row (`f-02-fert-insights-light`) |
| 5 | No console 404 on a day without a log | ✔ calendar reads `GET /health-logs?from_date=D&to_date=D` → 200, with 0 console errors |
| 6 | BBT / Log / Insights lavender background (light) | ✔ `.view` = `rgb(242, 236, 255)` on all three (BBT included) |
| 7 | `/messages/daily` on O+1 is not "fertile" for an avoiding user (1402) | ✔ `context_info`: `phase luteal`, `subphase post_ovulation`, `is_fertile_window false` |
| 8 | Post-ovulation card note (1402 home, light + dark) | ✔ «…پنجره باروری احتمالاً تمام شده و احتمال باروری از این روز کمتر است…». The old «…بالاتر است» is gone |
| 8a | Home «توصیه‌های امروز» on O+1 (avoiding) | ✘ still «باروری: روزهای اوج باروری. اگر قصد بارداری دارید، بهترین زمان است.» (bug **F-1**) |
| 9 | Legacy reminders sheet: cancelled row disabled | ✔ switch off + `disabled`, aria «این نوبت لغو شده و یادآورش روشن نمی‌شه», meta starts «لغو شده، …». API `PUT /care/appointments/125 {"is_active":true}` → 422 |
| 10 | Legacy sheet separator «، » and «ساعت ۸:۰۰» | ✔ «دکتر رضایی، سونوگرافی، ۸ مهر ۱۴۰۵», «۴۰۰ میکروگرم، هر روز ساعت ۸:۰۰» (light `f-07`, dark `f-08`) |
| 11 | Checkup detail «ثبت نوبت» prefilled via `?prefill=` | ✔ `/fa/checkups/3` → `/fa/reminders/appointment/new?kind=in_person&prefill=<opaque id>`, and «توضیح کوتاه» = «پاپ‌اسمیر / HPV». Nothing readable in the URL |
| 12 | Profile shows «حذف حساب» and «گرفتن خروجی داده‌ها» | ✔ both under «حریم خصوصی و داده‌ها» (`f-15`) |
| 13 | Export downloads JSON | ✔ `ritme-data-export.json` (1.4 KB, valid JSON). Keys: `account, cycle_histories, exported_at, health_logs, pregnancy, profile, reminders`. `account.mobile` = the user's own. `GET /profile/export` 200 go |
| 14 | Delete account ends the session | ✔ dialog → «حساب رو برای همیشه حذف کن» → `DELETE /account` 200 → `/fa/signup`. Token removed from localStorage, `ritme_auth` cookie gone, old token → 401, user row count 0 |
| 15 | Home reminders card makes no `GET /care/appointments/{id}` | ✔ only `GET /care/today` (light + dark loads) |
| 16 | Bell-off appointment shows no reminder chip | ✔ «سونوگرافی، دکتر رضایی / فردا، ساعت ۱۸:۰۰» with no chip. Control: bell on → «۱ روز قبل یادآوری» appears; then restored to off |
| 17 | 61st write in a minute → 429 with the Persian message in the appointment form | ✔ `POST /care/appointments` 429 go. The form shows «یه لحظه صبر کن و دوباره ذخیره کن؛ توی این یک دقیقه تغییرهای زیادی فرستاده شده.» (`f-11`). API body: `{"success":false,"message":"…","error_code":"too_many_requests","retry_after":48}`, `Retry-After: 48` |
| 18 | PWA install banner hidden while a dialog or sheet is open | ✔ with an iOS UA: banner `display:flex` on Profile (`f-12`). With the reminders sheet open → `display:none` (`f-13`). With the delete-account dialog open → `display:none` (`f-14`) |
| 19 | Admin lists don't scroll sideways at 390 px | ✔ all 19 nav screens: `scrollWidth 390 / innerWidth 390`, light. users, messages, pregnancy-weeks, checkup-types and admins also checked in dark |
| 20 | Admin numeric inputs show Persian digits | ✔ checkup type 3: `text/inputmode=numeric`, values «۳۶, ۲۱, ۶۵, ۱۰, ۲۰, ۳۰, ۳». Typing Latin «12» shows «۱۲»; typing «۴۵» is kept. Not saved |
| 21 | Admin week grid 42/42 | ✔ «۴۲ از ۴۲», 42 cells, 0 «افزودن» (`f-20`) |
| 22 | Alert badge count = Alerts cards (pregnant + spotting) | ✔ Today badge «هشدارها، ۲ پیام تازه» = 2 cards on `/fa/pregnancy/alerts` («لکه‌بینی ثبت شده» urgent + «وارد هفتهٔ ۱۱ شدی»). `unread_alerts` 2 = 2 in the API |
| 23 | Every `/api` response `X-Backend: go` | ✔ browser: 266 / 266 `/api/v1` and `/api/admin` responses. curl: every call. The only non-Go responses were `/api/session/flag` (the Next.js route, 204, by design) |
| 24 | No unexpected 4xx/5xx | ✔ 0 × 5xx. The 4xx seen were all expected: 1 × 401 `admin/v1/auth/me` before login, 1 × 429 (the throttle test), 1 × 401 after delete, 1 × 422 cancelled-reminder guard. The curl-side 404 on `/cycle/periods` and the 422s on `verify-otp` were harness mistakes (wrong path, empty code), not app faults |
| 25 | No CSP violations or console errors | ✔ 0 CSP reports, 0 JS exceptions. The only console lines were the network errors of the two expected responses above (the admin pre-login 401 and the provoked 429) |
| 26 | Dark mode renders (calendar, home ×3, BBT, Log, reminders sheet, appointment form, pregnancy Today and Alerts, admin) | ✔ |
| 27 | Latin digits in fa (spot checks) | ✔ none in the app captures. Admin: only «B6» inside a recommendation text (content) |
| 28 | Horizontal overflow at 390 px (app) | ✔ none seen |

## 5. Bugs

| # | Sev | Where | Bug | Repro |
|---|---|---|---|---|
| **F-1** | medium | `backend-go/internal/cycle/legacy/engine.go:131` (`e.dailyTips(ctx, phase, subphase, log)` with the legacy phase), shown by `frontend/src/screens/home/ui/HomePage.tsx:1110` (`calc?.dailyTips`) | **B-3 is only half fixed.** D-28 moved `/messages/daily` to the §19 display window (`internal/messages/manager/manager.go` `displayWindow`). But the home «توصیه‌های امروز» section reads `GET /cycle/today` → `data.calculation.daily_tips`. The legacy engine still builds those for phase **ovulation** on O+1, and `calculation.is_fertile_window` is also still `true` there. So an **avoiding** user on a low-chance day still sees «باروری: روزهای اوج باروری. اگر قصد بارداری دارید، بهترین زمان است.» and the ovulation energy tip («احساس اعتماد به نفس و اجتماعی بودن بیشتری…»). This is exactly the text the part A report quoted for B-3. The fix probably needs the same `displayWindow` applied before `dailyTips` (or in `/cycle/today`), plus a contract allow-list entry | 1402 (avoiding, periods 06-22 … 09-14, today cd 16 = O+1) → `/fa/home`, bottom. `GET /cycle/today` → `data.calculation.daily_tips[0]` = fertility/peak. `GET /messages/daily` → `context_info.is_fertile_window: false` (`f-06-home-avoid-1402-*`) |

Info:
- The admin theme and PWA banner state are per-browser only. They were restored or removed with the Chrome profile.
- After Alerts is opened, the Today badge drops to «هشدارها» (read-all). This is the known, intended behaviour.

## 6. Cleanup

- Test users 1401–1404 were deleted with `DELETE /api/v1/account`. 1405 was deleted through the UI.
- Their `otp_verifications` rows were deleted (stage DB only).
- A sweep of every `user_id` table for uid 22–26 found 0 rows. `users` has 0 rows for `099000014%`.
- **No `/panel` data was edited.** Numbers were typed into checkup type 3 but not saved; `checkup_types.updated_at` is
  unchanged at 2026-09-26 13:09:03. The admin theme was toggled back to light, then I logged out
  (`POST /api/admin/v1/auth/logout` 200).
- Care test data went away with the user. Appointment 126's bell was left off; it was deleted with the user anyway.
- Chrome (:9310) and the driver are stopped. The Chrome profile, the gate and admin credential files, the cookie jar,
  the tokens and the export download were deleted from the scratchpad.

## Screenshots (`docs/qa/screenshots/final/`, 585 px, pngquant, 31 files, 1.4 MB)

- **Calendar:** `f-01` day sheet light, `f-04` day sheet dark
- **Fertility screens:** `f-02-*` Log (today and 09-27), BBT, Insights (light); `f-05-*` home, BBT, Log (dark)
- **Homes:** `f-03` TTC home light; `f-06` avoiding home O+1 light and dark (F-1 at the bottom); `f-10` reminders card
  with the bell-off appointment, light and dark
- **Reminders and care:** `f-07` / `f-08` legacy reminders sheet; `f-09` checkup → appointment form prefilled; `f-11`
  appointment form 429 (dark)
- **PWA banner:** `f-12` visible (iOS UA), `f-13` hidden under a sheet, `f-14` hidden under a dialog
- **Profile:** `f-15` privacy rows; `f-16` signup after delete
- **Pregnancy:** `f-17` / `f-18` Today badge + Alerts (light); `f-19` Today + Alerts (dark)
- **Admin:** `f-20` week grid; `f-21` numeric fa digits; `f-22` / `f-23` dark lists

## T-M2-36 stage check

Re-check of **F-1** after T-M2-36 (commit `90876a3`, D-30). The fix reads `/cycle/today|date` by the §19 display
window. On O+1 that means: luteal, not fertile, and early-luteal tips.

**Result: F-1 fixed.** On O+1 the home shows no fertile or "peak" copy.

### Deploy

- `./deploy-stage.sh` shipped branch `stage` @ `90876a3`. It exited 0 and ended with «✅ Staging deploy done».
  - The frontend build layer re-ran (`frontend build rev: 90876a3-20260929T112954Z`).
  - Goose stayed at version 8 with no new migrations (`{"msg":"migrations","action":"goose_managed","applied":null,"version":8}`).
  - Every assertion in the script passed: `/up` 200 + `X-Backend: go`, `/` 401, `/admin` 301 → `/panel/`, `/oauth/token` 404, languages from Go.
- **Stage containers:** `backend-go` Up (healthy), `admin-web` Up (healthy), `frontend` Up, `mysql` and `redis` Up (healthy).
- **Production was not touched.** `deploy.sh` was not run. The `Created` timestamps before and after were identical (`diff` empty):

  | Container | Created | State |
  |---|---|---|
  | `ritme-proxy-1` | 2026-09-23T11:32:14Z | running |
  | `ritme-frontend-1` | 2026-09-01T13:14:46Z | running |
  | `ritme-backend-1` | 2026-08-31T08:37:28Z | running |
  | `ritme-queue-1` | 2026-08-31T08:37:32Z | running |
  | `ritme-mysql-1` | 2026-08-16T13:39:17Z | running (healthy) |
  | `ritme-redis-1` | 2026-08-16T13:39:17Z | running (healthy) |

  After the deploy, `https://api.ritme.app/up` returned 200. The only production-side action was the script's own
  graceful proxy reload (`nginx -t` ok).

### Test user

- **09900001501** (uid 27), avoiding: `user_goal non_ttc`, `pregnancy_intention avoiding`. Profile: cycle 28, period 5.
- Periods logged through the API: 06-22, 07-20, 08-17 and 09-14, each 5 days.
- Today, 2026-09-29, is cycle day 16. O = 15, so today is **O+1**.

### API (`X-Backend: go`, `Accept-Language: fa`)

| Call | Result |
|---|---|
| `GET /cycle/today` → `calculation` | ✔ `cycle_day 16`, `estimated_ovulation_day 15`, `phase luteal`, `subphase post_ovulation`, `is_fertile_window false`, `final_probability 7.14` |
| `calculation.text_flags` | ✔ only `probability_message` and `phase_info` «فاز فعلی: لوتئال (پس از تخمک‌گذاری)». There is no `fertility_status` |
| `calculation.daily_tips` (4) | ✔ early-luteal: تغذیه «روی غذاهای غنی از منیزیم و ویتامین B6…», ورزش «به حرکات ملایم‌تر…», خواب «کمی زودتر بخوابید…», سلامت روان «زمان خوبی برای تمام‌کردن کارهای نیمه‌تمام…» |
| `cycle_view` | ✔ `main_phase luteal`, `fertility_level low` |
| `GET /messages/daily` → `context_info` | ✔ `luteal`, `post_ovulation`, `is_fertile_window false` (D-28, unchanged) |
| Control: `GET /cycle/date/2026-09-28` (O) | ✔ still `ovulation` / `ovulation_likely` / fertile, with tips `fertility, energy, hydration, sleep` |

### Home (`/fa/home`, 390 px, light + dark)

- **«توصیه‌های امروز»** shows the same 4 early-luteal tips in both themes.
- **Whole page:** a DOM check of the page text found no «اوج باروری», «بهترین زمان است» or «اعتماد به نفس و اجتماعی».
- **Console:** 0 JS exceptions.
- **Screenshots:** `f-24-t36-home-o1-avoid-light.png`, `f-24-t36-home-o1-avoid-dark.png`. The 4th tip is under the PWA install
  banner in the capture, but it is present in the DOM.

### Method and cleanup

- **Browser:** headless Chrome 154 on CDP port 9311 with its own profile in a private scratch folder. 390×844 @1.5,
  mobile, `ritme_theme` light/dark.
- **Gate:** a chmod-600 netrc; the credentials were never printed. One gated page load minted the `ritme_stage` cookie.
- **Login:** `send-otp`, then the OTP was read from `otp_verifications` in the **stage** DB, then `verify-otp`.
- **Cleanup:**
  - The test user was deleted with `DELETE /api/v1/account` (200); the old token then returned 401.
  - Its `otp_verifications` rows were deleted (stage DB only).
  - `users` has 0 rows. A sweep of all 23 `user_id` tables for uid 27 found 0 rows.
  - Chrome was stopped. The netrc, cookie jar, token, OTP and profile were deleted.
