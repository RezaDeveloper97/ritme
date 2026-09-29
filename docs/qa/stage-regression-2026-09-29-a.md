# Stage regression 2026-09-29, part A: cycle, fertility/TTC, home, calendar

Target: `https://stage.ritmeapp.ir`, `stage` @ `e3aea8d` (goose v8, Go only). Today on stage: 2026-09-29 = 7 مهر 1405.
Harness: headless Chrome over CDP (port 9301, its own profile), 390 × 844 @2x, full-page captures, fa (all screens)
plus an en spot check, light and dark (`ritme_theme`). Gate: one Basic-auth page load minted the `ritme_stage` cookie,
then the header was dropped. OTPs were read from the stage MariaDB (`ritme-stage-mysql-1`). Every CDP call had a
20 s timeout, every run a perl `alarm`. No app code changed, nothing committed, production not touched.

## Test users (all deleted afterwards)

| Mobile | Onboarding (UI) | Seeded over the API | State on 2026-09-29 |
|---|---|---|---|
| `09900001101` | signup → OTP → name … conditions, intention **trying** | periods 06-26, 07-24, 08-21, 09-18 (5 days each) via `POST /cycle/period`; BBT cd 4–12 (36.30–36.42), LH negative cd 11 via `PUT /fertility/days` | TTC, **cycle day 12**, in the window, ovulation in 3 days (the `v19_Main` state) |
| `09900001103` | same, intention **trying**, last period 23 شهریور picked on the wheel | periods 06-22, 07-20, 08-17, 09-14; BBT cd 4–15 with a shift at cd 13, LH faint cd 11 / positive cd 12; two earlier cycles with shifts on cd 15 and 16. **Today's log filled through the UI**: LH «مثبت», 36.52, «شفاف و کشسان», «بدون محافظت», «درد تخمدان» → `PUT /fertility/days/2026-09-29` 200 | TTC, **cycle day 16**, post-shift (the Log / BBT / Insights state) |
| `09900001102` | same, intention **avoiding** | same periods as 1103 | non-TTC, cycle day 16 |

The first user's wheel was left untouched, so onboarding saved today (2026-09-29) as the last period, which is the
intended behaviour of an untouched picker. That period was deleted and replaced by the seeded ones. The other two picked
the date on the wheel.

## Checklist

| # | Check | Result |
|---|---|---|
| 1 | Signup → OTP (from the stage DB) → 9 onboarding steps → `/fa/home`, trying and avoiding | ✔ (3 users, `auth/send-otp`, `auth/verify-otp`, `POST /profile` 200) |
| 2 | `user_goal` follows the intention: trying → `ttc`; avoiding → `non_ttc` (`/messages/daily`) | ✔ |
| 3 | TTC home, cycle day 12: ring centre «تخمک‌گذاری تا ۳ روز», «روز ۱۲ سیکل · شانس بارداری متوسط», ink «ثبت امروز» | ✔ |
| 4 | TTC home: phase pills (پریود / فولیکولار / پنجره باروری (active) / لوتئال) | ✔ (cd 16: «لوتئال» active) |
| 5 | TTC home: chance card (donut, level, window start / ovulation / next period) | ✔ values. ✘ copy, see B-2 and B-6 |
| 6 | TTC home: «امروز تست LH بزن» only in the window with no LH today | ✔ (cd 12 shown; cd 16 and LH logged: hidden) |
| 7 | TTC home: tiles (LH amber / BBT teal / intercourse rose, values after a UI save: «مثبت · ۳۶٫۵۲° · بدون محافظت») | ✔ |
| 8 | Non-TTC (avoiding) home unchanged: phase card «فاز لوتئال» + «شانس بارداری کم» pill, no tiles, no chance card, no pills | ✔ layout. ✘ copy, see B-2 and B-3 |
| 9 | «یادآورهای امروز» placement: after the hero block (TTC: after chance card → tiles → LH tip), before «چیزهایی که در پیشه» | ✔ both homes |
| 10 | No late pop-in of tiles: tiles render in the same frame as the timeline (5 loads with the TTC hint, 5 without) | ✔ (see Timings) |
| 11 | Calendar month: window days amber, ovulation turquoise, predicted period hollow red, PMS violet; day sheet | ✔ window. ✘ PMS length (B-4), ✘ day-sheet chance (B-1) |
| 12 | Fertility Log (focus LH from the tile, highlight, chips, BBT field, save, back to home) | ✔ |
| 13 | BBT screen (tabs, hero, chart with coverline, window band, shift at cd 13, stat cards, tip + insights link, reminder button) | ✔ |
| 14 | Insights (confidence pill, window headline, window-weeks calendar, disclaimer, evidence copy per T-M5-12, history strips, «چطور تخمین را دقیق‌تر کنم؟» pill) | ✔ (low: B-8) |
| 15 | **One fertile window everywhere** (below) | ✔ |
| 16 | `fertility_level` word identical on home and Log (today) | ✔ «متوسط»/«متوسط» (cd 12), «کم»/«کم» (cd 16 TTC and avoiding) |
| 17 | Every `/api/v1` response `X-Backend: go` | ✔ 479/479 in the browser + every curl call |
| 18 | No 5xx | ✔ (477 × 200, 2 × 404, see B-5) |
| 19 | No console errors / exceptions | ✘ only the 404 of B-5 (avoiding user's calendar). No JS exceptions anywhere |
| 20 | No raw i18n keys | ✔ (all 38 captures) |
| 21 | No Latin digits in fa | ✔ (text-node scan on every fa capture, chart axes included) |
| 22 | No horizontal overflow at 390 px | ✔ (all captures, fa + en) |
| 23 | `<html data-theme>` matches the theme | ✔ |
| 24 | en spot check (home, calendar, log, BBT, insights light; avoiding home dark) | ✔ no keys, no overflow; «Window starts / 27 September» wraps onto two lines (cosmetic) |

### Fertile window across surfaces

Rule §19: `max(O − 5, period end + 1)` … `O`, where `O` = predicted next start − 14.

| Surface | 1101 (start 09-18, cd 12) | 1103 / 1102 (start 09-14, cd 16) |
|---|---|---|
| Calendar grid (aria labels, current cycle) | ۵–۹ مهر window, ۱۰ مهر ovulation (09-27 … 10-02) | ۱–۵ مهر window, ۶ مهر ovulation (09-23 … 09-28) |
| Calendar, next cycle | ۳–۷ آبان + ۸ آبان | ۲۹ مهر–۳ آبان + ۴ آبان (10-21 … 10-26) |
| Home ring / week strip | amber cd 10–14, turquoise cd 15; strip ۵–۹ amber, ۱۰ turquoise | amber cd 10–14, turquoise cd 15; strip ۴–۵ amber, ۶ turquoise |
| Home timeline row + bar | «۵ مهر تا ۱۰ مهر — در جریانه», ovulation «۱۰ مهر» | «۲۹ مهر تا ۴ آبان», ovulation «۴ آبان» (next cycle, as designed) |
| Home chance card | «شروع پنجره ۵ مهر · تخمک‌گذاری ۱۰ مهر» | «۲۹ مهر · ۴ آبان» |
| Insights | «تخمینی ۵ مهر تا ۱۰ مهر، تخمک‌گذاری حدود ۱۰ مهر» | «تخمینی ۲۹ مهر تا ۴ آبان، تخمک‌گذاری حدود ۴ آبان» |
| BBT (`fertile_window`, chart band) | cd 10–15 | cd 10–15 (all 3 cycles) |

All identical. Only the length of the window's neighbour, PMS, disagrees (B-4).

## Timings (stage, from this Mac, TTC user 1101)

Home: navigation → first frame with the 3 tile discs (`.fert-disc`), measured by a MutationObserver injected at
document start. One warm-up load first. `/profile` = the XHR's Resource Timing duration.

| Load | with TTC hint: tiles / timeline | `/profile` | without hint (`ritme_home_ttc` removed): tiles / timeline | `/profile` |
|---|---|---|---|---|
| 1 | 343 / 343 ms | 151 ms | 407 / 407 ms | 164 ms |
| 2 | 474 / 460 ms | 191 ms | 471 / 471 ms | 119 ms |
| 3 | 786 / 786 ms | 149 ms | 564 / 564 ms | 261 ms |
| 4 | 504 / 504 ms | 211 ms | 354 / 354 ms | 203 ms |
| 5 | 430 / 430 ms | 180 ms | 377 / 367 ms | 149 ms |

`GET /api/v1/profile` over curl (5×): 262–278 ms total. `/fa/profile` page, navigation → name visible: 209–430 ms
(XHR 88–131 ms). The ~5 s seen on 2026-09-28 did not reproduce: tiles never came later than the timeline.

## Bugs

| # | Sev | Where | What | Repro |
|---|---|---|---|---|
| B-1 | **medium** | `frontend/src/screens/calendar/ui/CalendarPage.tsx:132-136` (`chanceKey`), used at `:423` | The calendar day sheet's «احتمال باردار شدن» is a local 3-step map of the marker (fertile → «متوسط», ovulation → «زیاد»), not the engine's v1.1 `fertility_level`. The Log / `/fertility/days` for the same day says «زیاد» (O−2, O−1) and «خیلی زیاد» (O). The same word disagrees between the calendar and the Log. Today matched only because cd 12 = O−3 = medium. | User 1103: calendar → tap ۵ مهر → «متوسط» (`ttc-calendar-daysheet-past-cd14-light.png`); `/fa/fertility/log?date=2026-09-27` → «زیاد» (`ttc-log-past-cd14-light.png`). User 1101: ۹ مهر sheet «متوسط» vs `GET /fertility/days/2026-10-01` `high`. |
| B-2 | **medium** | `backend-go/internal/cycle/view/daily_card.go:241-243` (`post_ovulation` → `fertileNote()`), shown by `frontend/src/screens/home/ui/HomePage.tsx:800` | After ovulation (`subphase post_ovulation`, level low), the home phase card (avoiding) and the TTC chance card show «در این بازه احتمال باروری بر اساس پیش‌بینی چرخه بالاتر است…» right under «کم». The card's title (not rendered on the home) says «احتمالاً از پنجره باروری عبور کرده‌ای». A 1:1 Laravel port, so a parity or contract question. | Users 1102/1103 home (`avoid-home-cd16-*.png`, `ttc-home-cd16-*.png`); en: "Low · Fertility is estimated higher here…". |
| B-3 | **medium** | `backend-go/internal/messages/manager/manager.go:152` (copies the legacy calculation's `IsFertileWindow`/phase) | `/messages/daily` for cycle day 16 returns `context_info.phase = ovulation`, `is_fertile_window = true`, while the cycle view says `luteal / post_ovulation / low` and every surface ends the window on cd 15. So the home «توصیه‌های امروز» shows «باروری: روزهای اوج باروری. اگر قصد بارداری دارید، بهترین زمان است.» and the primary message «در اوج انرژی…» on a low-chance day, including to the **avoiding** user. It is the last place still on the legacy O + 1 window after T-M5-13. | User 1102 home, bottom (`avoid-home-cd16-light.png`); `GET /messages/daily` → `context_info`. |
| B-4 | low | `frontend/src/entities/cycle/model/schedule.ts:244` (calendar PMS = the per-day `is_pms_window`) vs `frontend/src/entities/cycle/model/predictions.ts:10` (`PMS_WINDOW_DAYS = 4`, home timeline and insights strips) | PMS is 7 days in the calendar (۱۳–۱۹ مهر) but 4 days on the home timeline («۱۶ مهر تا ۱۹ مهر») and in the insights strips (4 violet dots). task.md §25.2 says `pms_possible` = 1–3 days before the period. Three different lengths. | Users 1102/1103: calendar vs home «دوره PMS»; 1101: calendar ۱۷–۲۳ مهر vs home «۲۰ مهر تا ۲۳ مهر». |
| B-5 | low | `frontend/src/entities/health-log/api/queries.ts:44` (`GET /health-logs/{date}`, maps 404 → `null`), used by the calendar day sheet | For a day with no health log, Go returns 404 (Laravel parity), and Chrome logs "Failed to load resource: 404" on every calendar open of a user who has not logged today. It is the only console error of the run. | User 1102 → `/fa/calendar` (today is selected by default). |
| B-6 | low | `frontend/messages/fa/fertility.json:53` (`home.chanceCard.inWindow`), en likewise | Fixed copy «رابطه در امروز و ۲ روز آینده بیشترین شانس را دارد» while the ring says «تخمک‌گذاری تا ۳ روز» (ovulation = today + 3). It is only right when ovulation is exactly 2 days away (the artboard's state). | User 1101 home (`ttc-home-cd12-*.png`). |
| B-7 | low | `frontend/src/screens/fertility-log/ui/FertilityLogPage.tsx:140` (`chance.today`) | The Log for a past date still says «شانس بارداری امروز». `chanceCard.titleDay` («شانس بارداری {date}») exists. | `/fa/fertility/log?date=2026-09-27` (`ttc-log-past-cd14-light.png`). |
| B-8 | low | `frontend/src/screens/fertility-insights/ui/FertilityInsightsPage.tsx` (window-weeks caption) | When the grid starts in the previous month, the caption names only one month: «مهر ۱۴۰۵» over a first row of ۲۸–۳۱ شهریور. It correctly reads «مهر ۱۴۰۵ · آبان ۱۴۰۵» when the window itself spans two months. | User 1101 insights (`ttc-insights-cd12-*.png`). |
| B-9 | low (design) | `FertilityBbtPage.tsx:189`, `FertilityInsightsPage.tsx:120` and the Log root: `.view` has no background, so the white `.app-shell` shows | Light mode: the Log / BBT / Insights canvas is white `#FFF`. The artboards (and home/calendar/checkups) use the lavender canvas `rgb(242,236,255)` with the violet glow. Dark mode is fine. | Any fertility screen, light (`ttc-bbt-*-light.png` vs `docs/fertility-ttc/screenshots/audit/bbt-light-design.png`). |
| B-10 | low (colour) | home timeline «تخمک‌گذاری» row + bar tick | The ovulation drop and «N روز دیگه» pill are **green**, while the ring, week strip, chance card, calendar and insights use **turquoise** for ovulation (§10.2: turquoise = data/ovulation). | `ttc-home-cd12-light.png`, «چیزهایی که در پیشه». |

Not bugs, noted: BBT «۳ روز جاافتاده» counts cycle days 1–3 (the period, before the first reading) as gaps. That is
engine-consistent, though the artboard shows «بدون جاافتادگی» for readings from day 1. `/fertility/today` is also
fetched for non-TTC homes (harmless, 200).

## Remaining deviations from the design PNGs (`docs/fertility-ttc/screenshots/audit/*-design.png`)

Numbers differ because the data differs. Colour follows §10.2 (gradient CTAs are correct).

- **Home (`v19_Main`)**: matches (ring caption, pills, chance card, tiles, LH tip). The header has settings, bell and
  theme icons where the artboard has profile and bell, which is app-wide. The chance card copy is B-6.
- **Log (`v19_TTC_Log`)**: matches (single card, BBT field with steppers at the end, chips, note outside, «C°»).
  Canvas white (B-9).
- **BBT (`v19_TTC_BBT`)**: matches. The hero keeps «°C» as drawn. Legend «خط مبنا» without «· میانگین ۶ روز اول» is
  kept on purpose (audit #21). Canvas white (B-9).
- **Insights (`v19_TTC_Insights`)**: matches. The calendar shows only the window weeks (audit #19), the month caption
  is B-8, and the history label is «روز N» on one line (the artboard wraps it, a design quirk). Canvas white (B-9).
- **Calendar / day sheet**: no artboard in this set. The window colours follow the legend.

## Screenshots (`docs/qa/screenshots/a/`, 585 px wide, pngquant)

TTC cycle day 12 (1101), light + dark: `ttc-home-cd12-*`, `ttc-calendar-cd12-*`, `ttc-calendar-daysheet-cd12-*`
(۹ مهر), `ttc-log-cd12-*`, `ttc-bbt-cd12-*`, `ttc-insights-cd12-*`.
TTC cycle day 16 (1103), light + dark: `ttc-home-cd16-*`, `ttc-calendar-cd16-*`, `ttc-calendar-next-cd16-*` (آبان),
`ttc-log-cd16-*`, `ttc-bbt-cd16-*`, `ttc-insights-cd16-*`. UI flow (light): `ttc-log-focus-lh-cd16-light`,
`ttc-log-filled-cd16-light`. B-1 repro: `ttc-calendar-daysheet-past-cd14-light`, `ttc-log-past-cd14-light`.
Avoiding (1102), light + dark: `avoid-home-cd16-*`, `avoid-calendar-cd16-*`.
en: `en-ttc-home-cd12-light`, `en-ttc-calendar-cd12-light`, `en-ttc-log-cd12-light`, `en-ttc-bbt-cd12-light`,
`en-ttc-insights-cd12-light`, `en-avoid-home-cd16-dark`.

## Cleanup

`DELETE /api/v1/account` for 1101/1102/1103 (200 ×3). Then on the stage DB only: the 3 `otp_verifications` rows were
deleted. No orphaned `oauth_access_tokens`, and 0 leftover `fertility_logs` / `daily_health_logs` / `cycle_histories`
rows. The Chrome on port 9301 was stopped and the scratch folder (gate creds, cookie jar, tokens, profile) removed.
