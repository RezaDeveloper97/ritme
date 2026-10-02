# N3 stage smoke — B-N3-14

| | |
|---|---|
| Date | 2026-10-02 (run crossed midnight Tehran → some shots dated ۱۱ مهر) |
| Target | https://stage.ritmeapp.ir (staging only — production `/opt/ritme` / compose `ritme` untouched) |
| Commit deployed | `stage` branch `201eafa` ("fix(ui): N3 design-fidelity audit fixes (B-N3-13)"); frontend build rev `201eafa` |
| Deploy | `./deploy-stage.sh` → exit 0 (run by the orchestrator before this smoke) |
| Migration line | `{"level":"INFO","msg":"migrations","action":"goose_managed","applied":[26],"version":26}` |
| Stage env (non-secret) | `APP_ENV=staging`, `AI_PROVIDER=fake`, `PAYMENT_PROVIDER=fake`, `PLUS_TRIAL_DAYS=7` |
| Backend logs during smoke | `ritme-stage-backend-go-1`, last 3 h (1,459 lines): **0 `ERROR`/`WARN`, 0 5xx, no panics**. 4xx only: 1× 429 `/auth/send-otp` (my login burst hit `Throttle(5)`), 1× 402 `/logs/voice` (free user, expected `plus_required`), 2× 400 `/messages/daily` (menopause home, known N2 O-1). `ritme-stage-frontend-1` / `ritme-stage-admin-web-1`: 0 error lines. |
| Method | `bloom/bin/stage-smoke.mjs` helpers (gate cookie + OTP read at runtime over ssh, headless Chrome 390×844 @2x, `fa`, light + dark, PII masked before capture). Chrome had `--use-fake-device-for-media-stream --use-fake-ui-for-media-stream` so MediaRecorder ran headless. Click flows used real UI clicks. Each page's CDP network ≥ 400 + console errors were recorded in `n3-stage/summary.{json,md}` (137 captures: **0 console errors, 0 exceptions**, the only network finding is the known 400 `/messages/daily` on the menopause home). |
| Screenshots | `docs/qa/bloom/n3-stage/` (`<prefix>-<step>.<theme>.png`; 149 PNGs) |

## Test users

| | Number | Mode / Plus | Used for |
|---|---|---|---|
| U31 (N2 S1) | `0990•••••47` | cycle, Plus (cancelled, active to Jan) | 6 periods + ~80 days logs seeded; analysis Plus charts, voice record, customize light |
| U36 (N2 S2) | `0990•••••16` | TTC, Plus | 6 periods + ~90 days logs + BBT/LH/mucus/intercourse fertility days seeded; TTC sheet, TTC hub, fertility, customize dark |
| U37 (N2 S3) | `0990•••••41` | cycle, free → teen → postpartum → back to cycle | light seed (2 periods, 6 weeks logs); cycle sheet both themes, Plus locks, voice lock, teen + postpartum |
| U38 (N2 S4) | `0990•••••25` | menopause, Plus | 7 weeks of hot flash / insomnia logs; menopause sheet + hub |
| N1 (new) | `0990•••••72` | signup today → «باردارم», 22w0d, height 161 | onboarding regression, pregnancy sheet, pregnancy hub, weight-gain (18 weekly weights + BP seeded) |

Seeding went through the public APIs only (`POST /cycle/period`, `PUT /logs/days/{date}`, `PUT /fertility/days/{date}`, `PUT /onboarding/steps/health`), paced under the 60 writes/min throttle; 0 failed writes.

## Checks

Shot names are under `n3-stage/`, `.light.png` / `.dark.png`.

| Page / flow | Light | Dark | Result | Notes |
|---|---|---|---|---|
| Log sheet v2 — cycle: date strip, quick tiles, accordion, search | `C-log-01-sheet` | `CF-log-01-sheet` | PASS | Tiles change by phase (U31 day 10: no «پریود» tile, U37 day 38: 8 default tiles) |
| Bleeding detail panel (flow, colour, clots, odour) | `C-log-02-bleeding-panel` | `CF-log-02-bleeding-panel` | PASS | Saved as `bleeding{flow:medium,color:dark_red,clots:true,clot_size:small,odor:normal}` |
| Pain panel + body map (region → 1–10 score → relief) | — (light re-run toggled the region off; covered by U31 earlier API check) | `CF-log-03-pain-bodymap` | PASS | `pain.location.abdomen{level:moderate,score:6}`, relief heat |
| Measure panel (weight / BBT steppers, LH, pregnancy test) | `C-log-04-measure-panel` | `CF-log-04-measure-panel` | PASS | Shows the last weight + delta («۰٫۱ بیشتر از ثبت قبلی»). The panel's «ذخیره» saves the whole draft, so the main save button goes away |
| Save → day on calendar + home | `C-log-05-filled`, `C-calendar-after-log`, `C-calendar-day-10`, `C-home-after-log` | `CF-log-05-filled`, `CF-calendar-after-log`, `CF-calendar-day-10`, `CF-home-after-log` | PASS / see B-2 | Calendar day panel lists flow/colour/pain/moods/weight; home «ثبت امروز» rows updated. The home hero ignores the logged bleeding (B-2) |
| Log sheet v2 — TTC (LH + BBT tiles), save | `T-log-01…03` | `T-log-01…03` | PASS / see B-3 | `measurements.lh_test` saved; TTC hub shows «تست LH کم‌رنگ روز ۱۰». `/fertility/today` doesn't show it (B-3) |
| TTC home + calendar | `T-home`, `T-calendar` | `T-home`, `T-calendar` | PASS | |
| `/log/customize` — reorder, pin/unpin, 8-pin cap, hide | `K-customize-01`, `-02-edited` | `KD-customize-01`, `-02-edited` | PASS | 9th pin refused (counter stays ۸ از ۸). Known B-N3-04 minor confirmed: a pinned + hidden tile («رابطه») drops out of `pinned` |
| Custom item add / rename / delete | `K-customize-03…06` | `KD-customize-03…06` | PASS | «قهوه» → «قهوه تلخ» → deleted (soft delete; `GET /logs/custom-items` still lists it with `deleted_at`, UI hides it) |
| Reset to default (confirm sheet «برگردان») | `K-customize-07a`, `-07-reset` | `KD-customize-07a`, `-07-reset` | PASS | `is_default:true` afterwards; custom items kept |
| Saved prefs reflected in sheet | `K-customize-05-log-after` | `KD-customize-05-log-after` | PASS | |
| Voice tab — free user → Plus lock | `VF-voice-01-tab` | `VF-voice-01-tab` | PASS | «این بخش در ریتمی پلاس باز می‌شود»; API answers 402 `plus_required` |
| Voice — Plus user records (fake mic) → review chips → commit | `V-voice-01-tab`, `-02-recording`, `-03-review`, `-04-committed` | `V-voice-01…03` | PASS | The headless fake device recorded OK; fake provider gave the default fixture → chips درد شکم · متوسط / نفخ / حال «مطمئن نیستیم» (بی‌حوصله, غمگین, خستگی). «تأیید و ثبت ۳ مورد» → «۳ مورد ثبت شد», saved to the day |
| `POST /api/v1/logs/voice` with a webm carrying `RITME-FAKE:<key>` | — | — | PASS | `weight` → sleep good + weight 58.5 (cycle); `menopause` → hot_flash count/night/sweat (canvas targets). See B-8 for the label digits |
| Pregnancy sheet `/pregnancy/log` | `PG-pregnancy_log`, `PG-sheet-open`, `PG-sheet-saved` | `PG-pregnancy_log`, `PG-sheet-open`, `PG-sheet-saved` | PASS | Week/trimester heading, kicks / contractions «به‌زودی», bleeding alert row; nausea + heartburn saved → back to `/pregnancy` |
| Postpartum sheet (lochia panel) | `PP-log`, `PP-lochia-panel`, `PP-log-saved`, `PP-home_sheet_log` | same | PASS | `bleeding{lochia_amount,lochia_color,clots}`, no period row created. The postpartum home is still the cycle home (B-4) |
| Menopause sheet | `MN-log` | `MN-log` | PASS | Flashes/sweats/palpitations, sleep rows inline |
| Analysis hub — cycle (Plus) | `A-analysis` | `A-analysis` | PASS | Top finding «سیکل‌هایت منظم‌اند (نوسان ۳ روز). نفخ در ۴ از ۶ سیکل…» |
| Analysis hub — TTC | `TT-analysis` | `TT-analysis` | PASS | 5 trying cycles, 3 confirmed by BBT, ASRM referral copy |
| Analysis hub — teen | `TN-analysis` | `TN-analysis` | PASS | No mood/sleep chip, no upsell CTA (correlations shows a lock with no button) |
| Analysis hub — menopause | `MN-analysis` | `MN-analysis` | WARN (B-5) | Asks a menopause user to «۳ سیکل دیگر ثبت کن» |
| Analysis hub — pregnancy (free) | `PG-analysis` | `PG-analysis` | PASS | IOM weight band, BP vs 140/90, glucose + trimester symptoms Plus-locked |
| Analysis hub — postpartum | `PP-analysis` | `PP-analysis` | PASS (cycle hub by design, B-N3-08) | |
| `/analysis/cycle` | `A-analysis_cycle`, `TT-analysis_cycle`, `TN-analysis_cycle` | same | PASS | FIGO bands; a 58-day outlier is marked «لحاظ‌نشده» |
| `/analysis/period` | `A-analysis_period` | `A-analysis_period` | PASS | |
| `/analysis/symptoms` | `A-…`, `AF-…`, `MN-analysis_symptoms` | same | PASS | |
| `/analysis/correlations` — Plus | `A-analysis_correlations` | same | PASS | sleep×mood over 81 days, «نه رابطه علت و معلول» |
| `/analysis/correlations` — free / teen lock | `AF-analysis_correlations`, `TN-analysis_correlations` | same | PASS | |
| `/analysis/body` | `A-analysis_body`, `MN-analysis_body` | same | PASS | 7-day moving average (the seeded weights really are flat) |
| `/analysis/labs` | `A-analysis_labs` | same | PASS | Empty state until B-N6-06 |
| `/analysis/fertility` | `TT-analysis_fertility` | same | PASS / B-7 | BBT chart, coverline, window, LH/intercourse rows |
| `/analysis/pregnancy-weight` | `PG-analysis_pregnancy-weight` | same | PASS / B-6 | +5.4 kg, within the IOM band, baseline = first-trimester weight |
| `/analysis/monthly/1405-06?calendar=jalali` | `A-…`, `AF-…` | same | PASS | Jalali month title + stepper, deltas against Mordad |
| `/analysis/monthly/1405-07?calendar=jalali` (running month) | `A-analysis_monthly_1405-07…` | same | PASS | «تا امروز، ۱۰ مهر» subtitle |
| Plus locks for free users (voice, correlations, monthly PDF, pregnancy glucose/symptoms) | `VF-*`, `AF-*`, `PG-analysis` | same | PASS | |
| Regression: home | `R-home`, `T-home`, `MN-home` | same | PASS | MN 400 `/messages/daily` = known O-1 |
| Regression: calendar | `R-calendar`, `T-calendar` | same | PASS | |
| Regression: Plus paywall / plans / checkout (free user) | `AF-plus`, `R-plus_plans`, `AF-plus_checkout_plan_2` | same | PASS | 237,000 + 10% VAT = 260,700 T |
| Regression: Me hub (masked number) | `R-profile` | `R-profile` | PASS | |
| Regression: signup → OTP → name → gender → goal «باردارم» → week basis → conditions → health → ready → pregnancy home (fresh number) | `N-01…N-10` | `N-signup` | PASS | Lands on `/fa/pregnancy` week 23 (22w0d entered), EDD ۱۶ بهمن ۱۴۰۵ |

**Summary: 41 checks: 35 PASS, 6 PASS-with-finding / WARN, 0 FAIL.** No 5xx, no console errors or uncaught exceptions on any captured page, and dark mode was readable on every screen. Voice works end to end headless with the fake mic and the fake AI provider.

## Bugs / findings

| ID | Severity | Finding | Repro | Suspected file |
|---|---|---|---|---|
| B-1 | **High** | A bleeding log on a day whose *previous* day has no bleeding log always starts a **new period**, even inside an existing confirmed period or in the past. The new `cycle_histories` row is unconfirmed, open, and its `cycle_length` is measured against the *latest* period, so it can be negative. It also cuts the latest period's end and rewinds `user_profiles.last_period_start` to the back-dated day. One skipped day in the middle of a period is enough. The v2 sheet makes back-dated logging easy (date strip, voice), so this will happen to real users. | U37: `POST /cycle/period {start_date:2026-06-01,end_date:2026-06-05}` then `PUT /logs/days/2026-06-01`, `…-02`, `…-04` with `{"categories":{"bleeding":{"flow":"medium"}}}` → new row `2026-06-04, end NULL, cycle_length −120, is_confirmed 0`; U37 LMP is now `2026-06-04`. Seeding U31 produced rows with `cycle_length −78 / −23`, and the confirmed 09-23..09-27 period was cut to 09-23. | `backend-go/internal/healthlog/cyclehistory.go` `checkAndUpdatePeriodStart` (checks only yesterday's log, `GetLatestCycleHistory` instead of the period around/before `logDate`, no in-period check, LMP update for past dates — Laravel-parity port) |
| B-2 | Medium | After a late user logs bleeding in the log sheet, the app disagrees with itself. The calendar day and `/cycle/today` `calculation` say «روز ۱ سیکل». The home hero (`cycle_view`) still says «تأخیر پریود ۹ روز · روز ۳۸ سیکل», because the auto-created period row is `is_confirmed=0` and the resolver anchors only on confirmed or estimated rows. `/home/cycle-overview` also stays on the old prediction. | U37 (period 9 days late) → `/fa/log` → پریود «متوسط» → save → `/fa/calendar` day 10 vs `/fa/home` (`CF-calendar-day-10.dark`, `CF-home-after-log.dark`) | `backend-go/internal/cycle/resolver/resolver.go` `anchorFor` vs `internal/cycle/legacy`; or have the log sheet ask «پریودت شروع شد؟» and confirm the start |
| B-3 | Medium | LH / BBT logged in the v2 log sheet (`measurements.lh_test` / `bbt` in `health_log_entries`) never reach `fertility_logs`. `/fertility/today` and `/fertility/days/{date}` show `lh:null` and the chance doesn't change, while the TTC analysis hub does show the LH. Sync works only fertility → logs (seeded BBT shows up in `/logs/days`). | U36 (TTC) → `/fa/log` → «تست تخمک‌گذاری (LH)» → «مثبت» → save → `GET /fertility/today` → `lh.value:null` | `backend-go/internal/healthlog` (v2 `PUT /logs/days` → legacy sync covers `daily_health_logs` only) / `internal/fertility` |
| B-4 | Medium | The postpartum home is the cycle home: «تأخیر پریود ۱۰ روز · روز ۳۹ سیکل · پریودم شروع شد». It is wrong and insensitive for someone who just gave birth. | U37 → mode postpartum → `/fa/home` (`PP-home.light/dark`) | `frontend/src/screens/home/ui/HomePage.tsx` (no postpartum branch); product: postpartum home is planned later, but at least hide late-period copy |
| B-5 | Low | The menopause analysis hub tells the user «برای دیدن الگوها، ۳ سیکل دیگر ثبت کن» and «دست‌کم ۲ سیکل کامل لازم است». Cycle-based empty states for a mode without cycle tracking. | U38 → `/fa/analysis` (`MN-analysis`) | `frontend/src/screens/analysis/model/hub.ts` (menopause → cycle hub) + `backend-go/internal/analysis` top-finding phrases |
| B-6 | Low | The pregnancy weight card mixes week conventions: the chip says «هفته ۲۳» but the body says «+۵٫۴ کیلو تا هفته ۲۲». Same 22w0d day; `current.week` is completed weeks + 1 of the last weigh-in. | N1 → `/fa/analysis` / `/fa/analysis/pregnancy-weight` | `frontend/src/screens/analysis-pregnancy` (label of `current.week`) / `backend-go/internal/analysis/pregnancy.go` |
| B-7 | Low | `/analysis/fertility` subtitle «سیکل ۵ · ۱ مهر تا امروز» renders with broken bidi: the `·` jumps to the line start and the digits look like «۱۰ ۵». | U36 → `/fa/analysis/fertility` (`TT-analysis_fertility.dark`) | `frontend/src/screens/analysis-ttc` (subtitle; wrap the number in `<bdi>` or use one formatted string) |
| B-8 | Low | `POST /logs/voice` suggestion labels use Latin digits in `fa`: «وزن · 58.5 کیلوگرم», «گرگرفتگی · 3 بار». The UI builds its own chip labels for log items, but canvas targets show these labels. | `RITME-FAKE:weight` / `menopause` upload (above) | `backend-go/internal/voicelog/labels.go` |
| B-9 | Low | Teen log sheet shows the «ثبت با صدا · پلاس» badge and lock. Teens have no Plus path (`/fa/plus` redirects teens), so this is a dead-end upsell. | U37 as teen → `/fa/log` (`TN-log`) | `frontend/src/features/log-day` (voice tab: hide for teen) |
| O-1 | Info | `GET /logs/custom-items` returns soft-deleted items with `deleted_at` (probably on purpose, to label old entries). The customize UI hides them correctly. | | — |
| O-2 | Info | The pregnancy sheet's «همه موارد» still lists «پریود و لکه‌بینی», «وزن و دمای پایه» (category label, no BBT param in pregnancy), «رابطه و میل جنسی». The category labels aren't mode-specific. | N1 → `/fa/pregnancy/log` | `backend-go/internal/healthlog/taxonomy` labels |
| O-3 | Info | `send-otp` is throttled per IP at 5/min (shared with verify?). The smoke needs ≥30 s between logins. | | — |

## Not covered

- Real-device MediaRecorder (Safari mp4, permission denied, unsupported): headless only used Chrome's fake device. Voice quota exhaustion (6/min, 60/h) wasn't hit.
- Monthly PDF («ساخت PDF برای پزشک», B-N6-04) and labs data (B-N6-06): placeholders only.
- Admin-web, locale `en`, Android (never).
- Log sheet drag-to-reorder by pointer: only the ↑ button was used.

## Test data left on stage

- U31/U36/U37/U38 now have ~6 months of seeded periods and logs (U36 + fertility days). Custom items «قهوه» / «قهوه تلخ» / «چای سبز تلخ» are soft-deleted, and preferences are back to default.
- U37 has the B-1 repro rows (period 06-01..06-05 plus the spurious 06-04 row; LMP rewound to 06-04), an inferred open period from 10-02, and a lochia log on 10-03. Its mode is back to `cycle`.
- New user `0990•••••72` (pregnancy 22w0d, height 161, weight 60, 18 weekly weight/BP logs).
- Voice: 4 fake transcriptions on U31, 1 on U38 (nothing stored server-side beyond committed log rows).
