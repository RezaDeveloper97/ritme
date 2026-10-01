# CONTRA — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png`. Screens: 390 px, fa, headless Chrome over CDP against the local Go API
(test user with a combined pill, 21+7 pack started 7 days earlier, days 1–7 logged). Colours are checked against the token
map (`docs/canvas-build/README.md` §4), not the board hex.

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Contra_Setup` | `/contraception/setup` | [light](contra/CB-CONTRA-02/contraception-setup-light.png) | [dark](contra/CB-CONTRA-02/contraception-setup-dark.png) | ✔ | Same order: back header «روش پیشگیری», 24/900 question + lead, 2×4 method tiles (icon over label, selected = `--brand-soft` + `--brand` border), details card (pack-type chips, «شروع بسته فعلی» Jalali date, «ساعت یادآور»), sticky full-width «ذخیره». No bottom nav. Hormonal-IUD shield is `warm`, not the board rose (red is menstruation-only, CLAUDE.md §10.2). Non-pill methods swap the card for their dates (IUD insertion + lifetime, injection, implant) — no board for those states. |
| `nbl_Contra_Pill` | `/contraception` | [light](contra/CB-CONTRA-02/contraception-light.png) | [dark](contra/CB-CONTRA-02/contraception-dark.png) | ~ | Header «قرص پیشگیری» + «بسته ۱ · روز ۸» (real pack number, board shows 4), today card (pill disc, «قرص امروز · ۲۱:۰۰», Lalezar «روز ۸ از ۲۱», «خوردم», «یک قرص را جا انداختم» → `/contraception/missed`), «بسته فعلی» 4×7 grid (taken = `--data-soft`/`--data-deep` tick, today = `--brand` ring, break = dashed), streak (`--data-deep`) + next-pack (`--brand`) tiles, refill row. Deliberate differences: a settings button at the header end opens setup (the board's `[→Contra_Setup]` link has no visible control); legend reads «دایره تیک‌دار: خورده شد» instead of «دایره سبز» (taken is turquoise in the tokens and colour alone mustn't carry meaning); the refill row is tappable (chevron) to set packs left. In en, «22 October» wraps to two lines in the next-pack tile. |
| `nbl_Contra_Missed` | `/contraception/missed` | [light](contra/CB-CONTRA-03/missed-light.png) · [1 pill](contra/CB-CONTRA-03/missed-one-light.png) · [en](contra/CB-CONTRA-03/missed-en-light.png) | [dark](contra/CB-CONTRA-03/missed-dark.png) | ✔ | CB-CONTRA-03. Same order and hierarchy: back header «قرص جا افتاده» + method subtitle, «چند قرص جا افتاده؟» card with the two count chips (selected = `--brand-soft` + `--brand` border), «چه کار کنم؟» numbered steps (`--brand-soft` discs), danger card (bloom `UrgentCard`, `--danger-soft`), general-guidance note. Shots with «۲ قرص یا بیشتر» picked as on the board. All copy is the catalog group `missed_pill_rules` (chip = item title, steps = `meta.steps`, note = item body, danger card = `severity: urgent`), so wording follows the seed: the note reads «… در صورت شک با پزشک یا داروساز صحبت کن.» (the board's progestogen sentence lives in the `progestin_note` item, shown instead of chips to a progestogen-only pill user). Default chip = 1 pill. Non-pill method → empty state back to `/contraception`. No bottom nav. |
| `nbl_Contra_Other` | `/contraception/other` | [IUD](contra/CB-CONTRA-03/other-iud-light.png) · [IUD done](contra/CB-CONTRA-03/other-iud-done-light.png) · [injection](contra/CB-CONTRA-03/other-injection-light.png) · [implant](contra/CB-CONTRA-03/other-implant-light.png) · [pill → empty](contra/CB-CONTRA-03/other-pill-light.png) | [IUD done](contra/CB-CONTRA-03/other-iud-done-dark.png) · [IUD en](contra/CB-CONTRA-03/other-iud-en-dark.png) · [injection](contra/CB-CONTRA-03/other-injection-dark.png) · [implant set](contra/CB-CONTRA-03/other-implant-set-dark.png) · [date sheet](contra/CB-CONTRA-03/other-implant-sheet-dark.png) | ✔ | CB-CONTRA-03. Header «یادآورهای روش پیشگیری»; IUD card = string-check row + switch (`PUT /reminders/{id}` is_active), «تاریخ تعویض» row with the Jalali replacement-year tag («گذاشتن ۱۴۰۵ + N سال عمر» fills the board's `[سال گذاشتن] + طول عمر` placeholder), «ویزیت کنترل» row with turquoise «انجام شد» → `followup_done: true` (the visit reminder is removed server-side, row then shows a success «✓ انجام شد» pill); IUD danger note (`UrgentCard variant=note`). Injection card: «تزریق بعدی», Lalezar «۱۸ روز دیگر», «۲۷ مهر · معمولاً هر ۱۲ هفته». Implant row with «تنظیم کن» / «تغییر» → calendar sheet → `replace_on`. Deliberate differences: the board stacks all three methods, real data shows only the user's method (one method per user); injection adds «امروز تزریق کردم» (within 21 days of the due date) as its done state; injection/implant get a neutral info note instead of the IUD-only warning; hormonal-IUD shield is `warm`, copper `bloom` (no red). Pill/condom/other → «این روش یادآور جداگانه‌ای ندارد» empty state. Exercised on the test user (copper + hormonal IUD, injection, implant) then restored to the combined pill 21+7 from 2026-09-24 (pill log days 1–7 intact). |

## CB-CONTRA-04 — epic QA

Local run 2026-10-01: Go API :8020 (`ritme_dev`) + Next dev :3000, headless Chrome over CDP at 390 px, test user
`09120000427`. Shots in `contra/CB-CONTRA-04/`.

### Journey (UI-driven, DB/API checked after each step)

| # | Step | Result |
|---|---|---|
| 0 | Start: combined pill 21+7 from 2026-09-24, days 1–7 logged; `/profile/mode` switch on + «روش و بسته قرص» row | ✔ |
| 1 | `/profile/mode` switch **off** | ✔ `DELETE /contraception/method`: method `null`, `track_contraception=0`, pref `{"pill":false}`, refill reminder + `contraception_reminders` row gone, 7 pill logs kept; manage row hidden |
| 2 | Switch **on** → `/contraception/setup` | ✔ navigates to setup, nothing saved yet |
| 3 | Pick «قرص ترکیبی», chip «۲۱ قرص + ۷ روز استراحت», start date sheet → «۲ مهر», «ذخیره» | ✔ lands on `/contraception` «بسته ۱ · روز ۸»; streak «۷ روز» (kept log counts); pref `{"pill":true}` 21:00; flag on |
| 4 | «خوردم» → «امروز خوردی» + «برگرداندن» | ✔ row `2026-10-01 taken`, streak ۸; undo → row deleted, streak ۷ |
| 5 | «یک قرص را جا انداختم» → `/contraception/missed` → header back | ✔ guide for «قرص ترکیبی» (catalog copy); back → `/contraception` |
| 6 | Header settings → setup → «آمپول سه‌ماهه», last injection «۲ مهر», save | ✔ `/contraception` shows the non-pill summary; «یادآورهای روش» → `/contraception/other` «تزریق بعدی · ۷۷ روز دیگر · ۲۶ آذر»; pref `{"pill":false}` |
| 7 | `/profile/mode` switch off | ✔ method/reminders gone (`/care/appointments` empty), 7 pill logs kept; `/contraception` shows «هنوز روشی ثبت نکرده‌ای» + «انتخاب روش» |
| 8 | Restore (API `PUT /contraception/method`, same body as the original) | ✔ combined 21+7 from 2026-09-24, packs_left 1, reminder 21:00, pill logs 09-24…09-30, refill reminder 2026-11-14 (new reminder id) |

The IUD and implant states were set through the same `PUT` for the fidelity shots, then step 8 was re-run.

### Reminders — what fires vs what is only scheduled

There is **no push sender** in backend-go (no cron/worker, no web-push) and the frontend schedules no local
notifications, so **nothing fires** today. What is verifiable:

| Reminder | Stored as | Verified |
|---|---|---|
| Daily pill | `notification_preferences` `categories.pill` + `schedule.pill.time` | ✔ `true` / 21:00 for a pill method, `false` after switching to injection and after switch-off. `internal/reminders.Plan` (run in-process with a `go test -overlay` probe, no repo file) emits `pill at 21:00 send=true` for the pill prefs and `send=false reason=category_off` once off. |
| Next injection | care `reminders` row `type=appointment`, «آمپول سه‌ماهه بعدی» 2026-12-17 09:00 (= injected_on + 12 w) | ✔ in `GET /api/v1/reminders` and `GET /api/v1/care/appointments` (`remind_at` 2026-12-16 09:00, `days_until` 77); removed on method change/stop. Not part of `reminders.Plan` (cycle categories only) — a future sender reads it from `reminders`. |
| Pill refill, IUD string check / 6-week visit / replacement, implant replacement | care `reminders` rows | ✔ created with the expected dates (refill 2026-11-14, string check monthly from 2026-10-01, visit 2026-10-13, replacement 2036-09-01, implant 2029-03-01) |

Gaps for the future sender (already in CB-CONTRA-01 open items, re-confirmed): `Plan` still emits the pill on
break days (2026-10-16 → `send=true`) and defers a pill time inside quiet hours to 08:00 (23:30 → `quiet_hours`,
defer 08:00).

### Final verdicts

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Contra_Setup` | `/contraception/setup` | [light](contra/CB-CONTRA-04/setup-light.png) | [dark](contra/CB-CONTRA-04/setup-dark.png) | ~ | Structure, order, copy and states match. Method icons differ from the board for injection (board syringe → `calendar`) and implant (board rod → `hand`): the shared icon set has no syringe/implant glyph — proposed follow-up CB-CONTRA-04b (shared/ui/Icon is outside this epic's files). |
| `nbl_Contra_Pill` | `/contraception` | [light](contra/CB-CONTRA-04/contraception-light.png) · [en](contra/CB-CONTRA-04/contraception-en-light.png) | [dark](contra/CB-CONTRA-04/contraception-dark.png) | ✔ | **Fixed**: long next-pack dates («22 October», «۳۰ اردیبهشت», «30 September») wrapped to two lines in the half-width tile — labels over 9 characters now step down to 22 px (same 33 px line box), `ctr-stat-num.is-long`. Deliberate differences from CB-CONTRA-02 stand (header settings button, «دایره تیک‌دار» legend, tappable refill row). |
| `nbl_Contra_Missed` | `/contraception/missed` | [light](contra/CB-CONTRA-04/missed-light.png) | [dark](contra/CB-CONTRA-04/missed-dark.png) | ✔ | «۲ قرص یا بیشتر» picked as on the board; steps, danger card and note from the catalog. No change. |
| `nbl_Contra_Other` | `/contraception/other` | [IUD](contra/CB-CONTRA-04/other-iud-light.png) · [injection](contra/CB-CONTRA-04/other-injection-light.png) · [implant](contra/CB-CONTRA-04/other-implant-light.png) | [IUD](contra/CB-CONTRA-04/other-iud-dark.png) · [injection](contra/CB-CONTRA-04/other-injection-dark.png) · [implant](contra/CB-CONTRA-04/other-implant-dark.png) | ✔ | All three methods now shot in both themes (implant light was missing). Same icon note as Setup for the injection/implant rows. No change. |

No ✘ open.
