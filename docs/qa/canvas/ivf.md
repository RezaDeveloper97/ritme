# IVF — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png` (`roadmap/bin/shot-board.sh`). Screens: 390 px, headless Chrome over
CDP (`bloom/bin/shot.mjs --token`) against a local Go API (:8121) + Next dev (:3101). Test user **`09900002021`**
(created for CB-IVF-02): life stage `ttc` + «IVF/IUI» on, cycle 1 (antagonist) started 1405-07-04, stimulation from
1405-07-05 (→ «روز ۷ تحریک» on 1405-07-11), two medicines (suppression 08:00, FSH 150 IU 20:00), next scan tomorrow
09:00, beta 1405-07-28; linked partner companion **`09900002022`** (meds + appointments view) so the reminder toggle
shows. Colours are checked against the token map (`docs/canvas-build/README.md` §4), not the board hex.

## CB-IVF-02

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_IVF_Home` | `/ivf` (`/home` redirects for ttc + «IVF/IUI») | [home](ivf/CB-IVF-02/fa_ivf.home.light.png) · [dose logged](ivf/CB-IVF-02/fa_ivf.logged.light.png) · [no cycle](ivf/CB-IVF-02/fa_ivf.start.light.png) · [en](ivf/CB-IVF-02/en_ivf.home.light.png) | [home](ivf/CB-IVF-02/fa_ivf.home.dark.png) · [no cycle](ivf/CB-IVF-02/fa_ivf.start.dark.png) · [en](ivf/CB-IVF-02/en_ivf.home.dark.png) | ✔ | Same hierarchy and copy: eyebrow «درمان ناباروری» + Lalezar «IVF · سیکل اول» + bloom egg disc; «مرحله فعلی» card with the outlined «روز ۷ تحریک» chip and CB-CORE-02 `StepTimeline` (6 stages, done = turquoise check, current = brand ring, titles/hints from catalog `ivf_stages`, a known future date such as the beta day under its step); «تزریق‌های امروز» rows (syringe disc turquoise when done / brand to do, «۰۸:۰۰ · زیرجلدی · ۱۵۰ واحد», «انجام شد» turquoise pill ↔ «ثبت» brand outline — tap logs / undoes via `POST`/`DELETE /ivf/meds/{id}/doses`); «نوبت بعدی» row (calendar bloom disc, «فردا ۰۹:۰۰», chevron → `/reminders/appointment/{id}`); companion card «همدمت هم در جریان باشد» + switch (`PUT /ivf/cycles/current {notify_companion}`), hidden without a linked companion. Nav: امروز · درمان (syringe) · + · خدمات · من. Deliberate differences: «برنامه» link and the «ثبت نتیجه سونو» / «دو هفته انتظار» pills are hidden until CB-IVF-03/04/05 add `/ivf/meds`, `/ivf/scan`, `/ivf/tww` (`IVF_SCREENS_READY` in `screens/ivf/model/home.ts`) — no 404 links; «درمان» opens `/ivf#ivf-doses` until then (`NAV_READY.ivfMeds`). Medicine titles are the names the user typed (the board shows class names). No board for the no-cycle / switch-off states: minimal `EmptyState` (start a cycle → `POST /ivf/cycles` at «آماده‌سازی»; switch off → link to `/profile/mode`). |
| `IA_Nav` (IVF row) | `/home` with «IVF/IUI» off | [ttc nav](ivf/CB-IVF-02/fa_home.flag-off.light.png) | — | ✔ | Flag off → the plain TTC nav (امروز `/home` · باروری · + · خدمات · من) and the cycle home; flag on → `/home` hands over to `/ivf`. Nav shown on `/ivf` (tab root, `app-nav.ts`). |

## CB-IVF-03

Local Go API :8151 + Next dev :3105. Test user `09900002021` (cycle re-created by CB-IVF-04's seed) got three medicines
via `POST /ivf/meds` on 1405-07-12 (2026-10-04): FSH 150 IU 20:00 from 1405-07-05 with 3 pens, «آنتاگونیست» 08:00 from
1405-07-09 with 5 prefilled syringes, hCG trigger 1405-07-14 22:30; today's 08:00 dose logged from the screen with
«ران · راست» (request body `{"date":"2026-10-04","slot":"08:00","site":"thigh_right"}`).

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_IVF_Meds` ([board](boards/nbl_IVF_Meds.png)) | `/ivf/meds` (stage tab «درمان») | [schedule](ivf/CB-IVF-03/fa-meds.light.png) · [dose logged w/ site](ivf/CB-IVF-03/fa-meds-logged.light.png) · [en](ivf/CB-IVF-03/en-meds.light.png) | [schedule](ivf/CB-IVF-03/fa-meds.dark.png) · [en](ivf/CB-IVF-03/en-meds.dark.png) | ✔ | Same hierarchy and copy: back header «برنامه تزریق» + «سیکل اول · تحریک»; trigger card (bloom tint, clock disc, title/body from catalog `ivf_guidance.trigger_timing`, exact «۱۴ مهر ساعت ۲۲:۳۰» line, «انجام شد» once taken); «امروز» / «فردا» lists («FSH · ۱۵۰ واحد», «۲۰:۰۰»; today rows log/undo, a taken injection shows its site); «محل تزریق را عوض کن» picker + rotation line «دفعه قبل: … پیشنهاد امروز: …» + the site hint from catalog; «موجودی دارو» rows with «کم است» (danger soft) / «کافی» (data) pills and «۳ قلم باقی مانده · کافی تا چهارشنبه»; dashed «افزودن دارو از روی نسخه». Deliberate differences: **8 sites** (task scope + catalog `ivf_injection_sites`) instead of the board's 4, labelled «شکم · راست بالا» etc. from the catalog; the API's least-recently-used `sites.suggested` is pre-selected, «دفعه قبل» gets a history tag, and the chosen site is sent with the next injection log (non-injected routes send none); stock rows open the edit form (chevron); a medicine without tracked stock shows «موجودی ثبت نشده» and no pill. Nav: «درمان» now → `/ivf/meds` (`NAV_READY.ivfMeds`), the IVF home's «برنامه» link is on (`IVF_SCREENS_READY.meds`). |
| — (no board: «افزودن دارو از روی نسخه») | `/ivf/meds/new` | [preset picked](ivf/CB-IVF-03/fa-med-new.light.png) · [en](ivf/CB-IVF-03/en-med-new.light.png) | [fa](ivf/CB-IVF-03/fa-med-new.dark.png) · [en](ivf/CB-IVF-03/en-med-new.dark.png) | ✔ | Minimal form (DECISIONS #8), no nav: prescription presets (catalog `ivf_med_presets`, classes only) fill name/type/route/unit/times/stock unit; type + route + unit chips; amount; daily times (1–4, wheel sheet) or, for the trigger, exact date + time; start / optional end (calendar sheet, Jalali in fa); optional stock tracking (unit chips, steppers for count and doses per unit); notes; «not a prescription» note. |
| — (no board: edit) | `/ivf/meds/{id}` | [fa](ivf/CB-IVF-03/fa-med-edit.light.png) · [en](ivf/CB-IVF-03/en-med-edit.light.png) | [fa](ivf/CB-IVF-03/fa-med-edit.dark.png) · [en](ivf/CB-IVF-03/en-med-edit.dark.png) | ✔ | Same form pre-filled; the stock starts at the units left today, so saving untouched keeps the count (`PUT` semantics of CB-IVF-01); «حذف دارو» → confirm sheet → `DELETE /ivf/meds/{id}`. |
## CB-IVF-04

Local stack for these shots: Go API :8161 + Next dev :3106 (fresh `ritme_dev` — the test stack had been reset, so
`09900002021` was re-created: ttc + «IVF/IUI», cycle 1 antagonist started 2026-09-26 / 1405-07-04, stimulation from
2026-09-27; scans on 2026-09-29, 10-01, 10-03 = stim days 3/5/7; today 1405-07-12 = stim day 8, no scan).

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_IVF_Scan` | `/ivf/scan[?date=Y-m-d]` | [saved day](ivf/CB-IVF-04/fa_ivf_scan_date_2026-10-03.saved.light.png) · [new day](ivf/CB-IVF-04/fa_ivf_scan.new.light.png) · [after save](ivf/CB-IVF-04/fa_ivf_scan.after-save.light.png) · [en](ivf/CB-IVF-04/en_ivf_scan_date_2026-10-03.saved.light.png) | [saved day](ivf/CB-IVF-04/fa_ivf_scan_date_2026-10-03.saved.dark.png) · [new day](ivf/CB-IVF-04/fa_ivf_scan.new.dark.png) · [en](ivf/CB-IVF-04/en_ivf_scan_date_2026-10-03.saved.dark.png) | ✔ | Same hierarchy and copy: back header «ثبت نتیجه سونو» + «روز ۷ تحریک · ۱۱ مهر»; «تعداد فولیکول‌ها بر اساس اندازه» + hint; one card with «تخمدان راست» / «تخمدان چپ» columns of compact CB-CORE-02 `NumberStepper`s for the four bins (کمتر از ۱۰ میلی‌متر · ۱۰ تا ۱۴ · ۱۵ تا ۱۷ · ۱۸ و بیشتر, 0–60); card with «ضخامت آندومتر … میلی‌متر» and «استرادیول (E2)» («مقدار آزمایش» placeholder); «روند رشد» card with two series per scan day (10–14 mm brand, ≥ 15 mm bloom — the board's purple/pink pair) + legend; the «تفسیر … با پزشکت است» note; sticky «ذخیره» (`PUT /ivf/scans/{date}`, enabled once something changed). Deliberate differences: the chart runs oldest → newest left to right (the app-wide `widgets/charts` rule), so روز ۳ sits on the left where the board's RTL drawing has it on the right; endometrium/E2 are editable fields (the board shows values only), E2 has a small unit toggle (pg/mL ↔ pmol/L, API `e2_unit`); a «حذف نتیجه این روز» text button (`DELETE /ivf/scans/{date}`) shows only when that day has a saved scan. No board for no-cycle / load-error states: minimal `EmptyState`. |
| `nbl_IVF_Home` (sono pill) | `/ivf` | [scan action](ivf/CB-IVF-04/fa_ivf.scan-action.light.png) | — | ✔ | `IVF_SCREENS_READY.scan` flipped: the «ثبت نتیجه سونو» pill now shows and opens `/ivf/scan`. |

## CB-IVF-05

Local Go API :8171 + Next dev :3107. Dedicated test user **`09900002051`** (ttc + «IVF/IUI» on; a cycle per run:
stimulation from 2026-09-12, retrieval 2026-09-25, transfer 2026-09-30 = ۸ مهر, beta 2026-10-13 = ۲۱ مهر → «۹ روز»
on 1405-07-12; «پروژسترون» luteal support 400 mg 08:00 + 20:00 with 08:00 logged → «۱ از ۲»; mood «امیدوار»). Each
outcome screenshot closed that run's cycle (hence «سیکل ۷» on the home); the user is left with an open TWW cycle.
`09900002021` was not touched.

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_IVF_TWW` ([board](boards/nbl_IVF_TWW.png)) | `/ivf/tww` | [fa](ivf/CB-IVF-05/fa_ivf_tww.light.png) · [en](ivf/CB-IVF-05/en_ivf_tww.light.png) · [no cycle](ivf/CB-IVF-05/fa_ivf_tww.nocycle.light.png) | [fa](ivf/CB-IVF-05/fa_ivf_tww.dark.png) · [en](ivf/CB-IVF-05/en_ivf_tww.dark.png) · [no cycle](ivf/CB-IVF-05/fa_ivf_tww.nocycle.dark.png) | ✔ | Same hierarchy and copy: back header «دو هفته انتظار» + «انتقال جنین: ۸ مهر»; thick brand `ProgressRing` (fill = days since transfer ÷ transfer→beta span) with Lalezar «۹ روز» + «تا تست خون (بتا)» (beta day → «امروز», past → «روز آزمایش رسید», no beta date → «روز N بعد از انتقال» / clinic note); «امروز حالت چطور است؟» card with آرام · امیدوار · نگران · خسته single-select chips (`PUT /ivf/tww/{today}`, optimistic, tap again clears) and the catalog `ivf_guidance.tww_feelings` line; «داروهای این دوره» rows (`luteal_support`, pill disc, times + dose, «۱ از ۲» rose pill → turquoise when all taken); the OHSS danger note from catalog `ivf_danger_signs` with a `tel:115` pill; result buttons «نتیجه مثبت بود» (heart, turquoise outline) / «نتیجه منفی بود» (chat). Deliberate additions: «ثبت علائم امروز» (opens the global log sheet — the API has no TWW symptom field), the catalog `early_test` note (task scope), the second danger row (fever after a procedure), a quiet «سیکل لغو شد» link (API outcome `cancelled`). Single-select mood (API stores one mood/day; the board tints two chips). No tab bar (back-header, sensitive screen). |
| — (no board: confirm) | `/ivf/tww` sheet | [negative](ivf/CB-IVF-05/fa_ivf_tww.confirm.light.png) | [negative](ivf/CB-IVF-05/fa_ivf_tww.confirm.dark.png) | ✔ | Half sheet before `POST /ivf/cycles/current/outcome`: says the cycle closes and (non-positive) its medicine reminders turn off — «پیش از قطع هر دارو با پزشکت هماهنگ کن». |
| — (no board: outcome) | `/ivf/tww` after saving | [negative](ivf/CB-IVF-05/fa_ivf_tww.negative.light.png) · [positive](ivf/CB-IVF-05/fa_ivf_tww.positive.light.png) · [en negative](ivf/CB-IVF-05/en_ivf_tww.negative.light.png) | [negative](ivf/CB-IVF-05/fa_ivf_tww.negative.dark.png) · [positive](ivf/CB-IVF-05/fa_ivf_tww.positive.dark.png) · [cancelled](ivf/CB-IVF-05/fa_ivf_tww.cancelled.dark.png) | ✔ | Calm on every result (DECISIONS #8 minimal; no confetti). Negative: neutral sprout disc, «متأسفیم که این نتیجه را گرفتی», reminders-off note, «اگر بخواهی» → «مراقبت و همراهی» → **`/loss`** (CB-LOSS-02's flow, board link `→Loss_Start`; no second loss flow) and «برگشت به درمان» → `/ivf`. Positive: «شروع پیگیری بارداری» → bloom's `/pregnancy/setup` (the activation path), «بعداً» → `/ivf`, «keep support medicines as your doctor says». Cancelled: only «برگشت به درمان». Rows follow the API's `next_steps`. |
| `nbl_IVF_Home` (TWW button) | `/ivf` | [home](ivf/CB-IVF-05/fa_ivf.light.png) | [home](ivf/CB-IVF-05/fa_ivf.dark.png) | ✔ | `IVF_SCREENS_READY.tww` on: the «دو هفته انتظار» pill links to `/ivf/tww` («ثبت نتیجه سونو» waits for CB-IVF-04). |

## CB-IVF-06 — epic QA

Worktree Go API :8201 + Next dev :3111, 390 px, fa. Board states via `bloom/bin/shot.mjs --token`, the journey through
the Playwright MCP browser (same build) plus API checks with `curl`. Screens in [`ivf/CB-IVF-06/`](ivf/CB-IVF-06/).

**Test data (local `ritme_dev`, through the API/UI):** the IVF catalog groups (`ivf_stages` 6, `ivf_protocols` 6,
`ivf_injection_sites` 8, `ivf_med_presets` 9, `ivf_guidance` 5, `ivf_danger_signs` 2) were already present from goose
`00027_ivf` — nothing had to be seeded by hand. New woman **`09900002061`** «نگار» (onboarding steps name / female /
goal ttc / cycle 2026-09-20, 5 d / 28 d) and new partner **`09900002062`** «آرش» (male, onboarding done). Companion
link #7 (partner, grants `meds: view`, `appointments: view`) accepted. Cycles: #1 (antagonist, closed **negative**),
#2 (closed **positive**), #3 (closed **cancelled**); four medicines on cycle 1 (FSH 150 IU 20:00 with 1 pen, antagonist
0.25 mg 08:00 with 5 syringes, hCG trigger 1405-07-14 22:30, vaginal progesterone 400 mg 08:00 + 20:00); scans on
2026-09-29 / 10-01 (API) and 10-04 (UI). Stage moves (prep → stim → tww) and retrieval / transfer / beta dates went
through `PUT /ivf/cycles/current`, since no screen edits them (known CB-IVF-02 open item). She ends with «IVF/IUI» off.
Earlier users `09900002021` / `09900002051` were not touched.

### Journey

| # | Step | Result |
|---|---|---|
| 1 | ttc user → `/profile/mode` → «درمان ناباروری (IVF/IUI) دارم» switch on | ✔ `life_stage.ivf_iui: true`, mode stays `ttc` |
| 2 | `/home` | ✔ redirects to `/ivf`; no cycle → «شروع سیکل درمان» empty state ([shot](ivf/CB-IVF-06/j02-home-redirect-start.light.png)) |
| 3 | «شروع سیکل درمان» | ✔ cycle 1 at «آماده‌سازی», «روز ۱ آماده‌سازی», 6-step timeline, «برای امروز دارویی در برنامه نیست» |
| 4 | `/ivf/meds/new`: presets «هورمون تحریک (FSH)» (+ amount, stock on) and «آنتاگونیست» (+ amount) → «ذخیره» | ✔ presets fill name / type / route / unit / time; both saved and back on `/ivf/meds` ([form](ivf/CB-IVF-06/j04-med-preset-fsh.light.png)) |
| 5 | `/ivf/meds`: log 08:00 with the suggested site, then pick «ران · راست» and log 20:00 | ✔ `ivf_dose_logs` site `abdomen_upper_right` then `thigh_right`; suggestion rotates to the least-recently-used `abdomen_upper_left`, «دفعه قبل» tag on the last site ([shot](ivf/CB-IVF-06/j05-meds-logged-rotation.light.png)) |
| 6 | IVF home «ثبت نتیجه سونو» → `/ivf/scan` → steppers + endometrium 9.5 + E2 1450 → «ذخیره» | ✔ `PUT /ivf/scans/2026-10-04` (stim day 8); chart روز ۳ · ۵ · ۸ with both series ([shot](ivf/CB-IVF-06/j06-scan-saved.light.png)) |
| 7 | Companion toggle with only a **pending** invite | ✔ card hidden on `/ivf`; `PUT /ivf/cycles/current {notify_companion:true}` → 422 «برای این کار اول یک همدم وصل کن.» |
| 8 | Partner accepts → `/ivf` | ✔ «همدمت هم در جریان باشد» card appears; switch on → `notify_companion: true`, `companion {linked, notify, shares_meds, shares_appointments}` all true ([shot](ivf/CB-IVF-06/j08-home-companion-on.light.png)) |
| 9 | Partner `/home` (→ `/companion`) | ✔ «داروهای نگار» (IVF meds incl. «تزریق تریگر (hCG)» و ۲ مورد دیگر) and «نوبت‌های نگار» (سونوگرافی فولیکول · دوشنبه ۰۹:۰۰), both «فقط دیدنی»; cycle/symptoms/pregnancy «not shared» line ([shot](ivf/CB-IVF-06/fa_home.partner.light.png)). The partner's `/home` fires three 409s (`messages/mode`, `messages/daily`, `home/cycle-overview`) before redirecting — bloom's male-account path, not IVF. |
| 10 | «درمان» tab | ✔ → `/ivf/meds` (nav: امروز `/ivf` · درمان `/ivf/meds` · + · خدمات · من) |
| 11 | Cycle → TWW (transfer ۸ مهر, beta ۲۱ مهر) → IVF home «دو هفته انتظار» → `/ivf/tww`, mood «نگران» | ✔ «۹ روز تا تست خون (بتا)», `GET /ivf/tww` `today.mood = worried`, progesterone «۱ از ۲», OHSS + fever danger rows with `tel:115` |
| 12 | «نتیجه منفی بود» → confirm «ثبت» | ✔ calm screen «متأسفیم که این نتیجه را گرفتی», reminders-off note; cycle closed (`GET /ivf` no open cycle), all four medicine reminders `is_active: false` ([shot](ivf/CB-IVF-06/j10-outcome-negative.light.png)) |
| 13 | «مراقبت و همراهی» | ✔ `/loss` (CB-LOSS-02 flow, «متأسفیم که این را تجربه کردی») ([shot](ivf/CB-IVF-06/j11-loss-from-negative.light.png)) |
| 14 | Cycle 2 at TWW → «نتیجه مثبت بود» → confirm → «شروع پیگیری بارداری» | ✔ «نتیجه مثبت ثبت شد» (no confetti) → `/pregnancy/setup` step 1 of 4 ([shot](ivf/CB-IVF-06/j13-pregnancy-setup-from-positive.light.png)); the expected 404 `GET /pregnancy/profile` (no profile yet) |
| 15 | Cycle 3 → `/ivf/tww` «سیکل لغو شد» → confirm → «برگشت به درمان» | ✔ «این سیکل متوقف شد», only «برگشت به درمان» → `/ivf` (start-cycle empty state) |
| 16 | `/profile/mode` switch off → `/home` | ✔ `ivf_iui: false`; `/home` stays, TTC nav امروز `/home` · باروری `/calendar` · + · خدمات · من ([shot](ivf/CB-IVF-06/j16-switch-off-ttc-nav.light.png)) |

### Fidelity (390 px, light + dark)

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_IVF_Home` ([board](boards/nbl_IVF_Home.png)) | `/ivf` | [stim day 8](ivf/CB-IVF-06/fa_ivf.light.png) | [stim day 8](ivf/CB-IVF-06/fa_ivf.dark.png) | ✔ | Same order and copy as the board: header, «مرحله فعلی» + «روز ۸ تحریک» chip, 6-step timeline, «تزریق‌های امروز» + «برنامه», «نوبت بعدی», companion card, «ثبت نتیجه سونو» / «دو هفته انتظار». **Fixed drift:** the hairline between today's dose rows bent up at both ends (shared `.nb-row` radius on a `border-top`) → `.ivf-screen .nb-list-rows > * + * { border-radius: 0 }` in the CB-IVF-02 CSS block (covers home, meds, TWW). **Fixed:** doses printed «۰.۲۵» → «۰٫۲۵» (`formatDecimal`). The board's appointment prep line («ناشتا نیاز نیست») shows only when the appointment has prep. |
| `nbl_IVF_Meds` ([board](boards/nbl_IVF_Meds.png)) | `/ivf/meds` | [schedule + trigger](ivf/CB-IVF-06/fa_ivf_meds.light.png) | [schedule + trigger](ivf/CB-IVF-06/fa_ivf_meds.dark.png) | ✔ | Trigger card (catalog copy + «تزریق تریگر (hCG) · ۱۴ مهر ساعت ۲۲:۳۰»), امروز / فردا, 8-site picker (deliberate vs the board's 4), inventory «کم است» / «کافی», dashed add. Same row-hairline + decimal fixes. **Fixed:** a stock at 0 printed «۰ قلم باقی مانده · کافی تا یکشنبه» (runs-out = today) — `runsOutLabel` now drops «کافی تا …» when no doses are left, the «کم است» pill says it (unit test added). |
| `nbl_IVF_Scan` ([board](boards/nbl_IVF_Scan.png)) | `/ivf/scan?date=2026-10-04` | [saved day](ivf/CB-IVF-06/fa_ivf_scan_date_2026-10-04.light.png) | [saved day](ivf/CB-IVF-06/fa_ivf_scan_date_2026-10-04.dark.png) | ✔ | Unchanged since CB-IVF-04: per-ovary steppers, endometrium «۹٫۵» / E2 fields, growth chart (oldest → newest left to right, the app-wide chart rule), doctor note, sticky «ذخیره», «حذف نتیجه این روز». |
| `nbl_IVF_TWW` ([board](boards/nbl_IVF_TWW.png)) | `/ivf/tww` | [day 4 after transfer](ivf/CB-IVF-06/fa_ivf_tww.light.png) | [day 4 after transfer](ivf/CB-IVF-06/fa_ivf_tww.dark.png) | ✔ | Ring «۹ روز», mood chips, meds «۱ از ۲», danger note, result buttons — as CB-IVF-05 (extra early-test note, fever row, `tel:115`, «سیکل لغو شد» kept). The selected mood uses the shared `Chip` pressed state (solid brand) where the board tints the chip soft — the design-system chip, not IVF-specific; left as is. Outcome screens: CB-IVF-05 shots + journey 12–15. |

**Verdicts:** 4 boards ✔, no ✘. Code changes (IVF files only): `globals.css` CB-IVF-02 block (+1 rule),
`screens/ivf/ui/IvfHomePage.tsx`, `screens/ivf-meds/ui/{IvfMedsPage,IvfMedFormPage}.tsx`,
`screens/ivf-tww/ui/IvfTwwPage.tsx` (dose → `formatDecimal`), `screens/ivf-meds/model/schedule.ts` (+ test).

**Follow-ups (not fixed here):**
- **CB-IVF-06b (proposal) — stage + dates editor:** no screen moves the stage (prep → stim → … → tww) or sets
  retrieval / transfer / beta / next-scan; the journey needed `PUT /ivf/cycles/current`. Without it a real user never
  reaches the TWW countdown or the trigger → retrieval steps.
- Stimulation meds stay active after the stage moves on (no `is_active` in the form; CB-IVF-03 open item) — TWW users
  still see FSH in «تزریق‌های امروز».
- The partner's companion meds subtitle prints «۰.۲۵ میلی‌گرم» (care backend formatting, Persian decimal) — bloom/care.
- `/loss` from an IVF negative offers pregnancy-loss kinds only (no «IVF ناموفق / انتقال ناموفق» option) — LOSS
  content, [needs clinical review].

## CB-IVF-06b — cycle setup, stage + date editor, med active switch

Worktree Go API :8211 + Next dev :3114, 390 px, fa. Journey through the Playwright MCP browser (dev build), board
states via `bloom/bin/shot.mjs --token`, API/DB checks with `curl` / `mariadb`. Screens in
[`ivf/CB-IVF-06b/`](ivf/CB-IVF-06b/).

**Test data (local `ritme_dev`):** woman **`09900002061`** (CB-IVF-06's user, ttc, «IVF/IUI» off) → through the UI:
cycle #12 (cycle 4, antagonist), two meds (FSH 20:00, antagonist 08:00, both now paused), one home dose log, the cycle
ends at «انتظار دو هفته‌ای» (retrieval ۸ مهر, transfer ۱۱ مهر, beta ۲۳ مهر). New woman **`09900002071`** «سارا»
(onboarding via API: female / ttc / cycle 2026-09-22; «IVF/IUI» on, no cycle) for the start + setup states.

### Journey (UI only)

| # | Step | Result |
|---|---|---|
| 1 | ttc user → `/profile/mode` → «درمان ناباروری (IVF/IUI) دارم» on → `/home` | ✔ `ivf_iui: true`; `/home` → `/ivf` start card |
| 2 | «شروع سیکل درمان» | ✔ now a link → `/ivf/cycle/new` (was a one-tap default POST) |
| 3 | Setup: protocol «آنتاگونیست» (catalog hint shown), «تحریک تخمک‌گذاری», start ۲ مهر, stimulation start ۵ مهر → «شروع سیکل» | ✔ `POST /ivf/cycles {protocol, stage: stim, started_on, stim_started_on}` → `/ivf`, «روز ۸ تحریک», prep done / stim current ([form](ivf/CB-IVF-06b/j03-setup-filled.light.png) — shot before the bottom note changed to `setup.later`, [home](ivf/CB-IVF-06b/j04-home-after-setup.light.png)) |
| 4 | Two meds from presets (`/ivf/meds/new`), home «ثبت» on the 08:00 antagonist | ✔ `ivf_dose_logs.site = abdomen_upper_left` = the schedule's `sites.suggested` (the CB-IVF-06 row from the home had `site NULL`); suggestion rotates to `thigh_left` |
| 5 | Timeline «ویرایش» → `/ivf/cycle` → next scan ۱۳ مهر 09:00 → «ذخیره» | ✔ `PUT {next_scan_at: "2026-10-05 09:00"}` only; care appointment «سونوگرافی فولیکول» created, home «نوبت بعدی» shows it ([editor](ivf/CB-IVF-06b/j06-editor-open.light.png)) |
| 6 | Stage «تخمک‌کشی» + retrieval ۱۴ مهر → «ذخیره» | ✔ `PUT {stage, retrieval_at}`, then the «داروهای تحریک متوقف شوند؟» sheet lists FSH + antagonist ([shot](ivf/CB-IVF-06b/j08-stop-stim-meds.light.png)) |
| 7 | «متوقف کن» | ✔ both `PUT /care/medications/{reminder_id} {is_active:false}`; `GET /ivf/meds` `is_active: false`, today's doses empty, inventory pills «متوقف» |
| 8 | `/ivf/meds/17` «در برنامه» switch on → off | ✔ the reminder flips each time (saved at once, not with the form) ([shot](ivf/CB-IVF-06b/j09-med-form-paused.light.png)) |
| 9 | Stage «انتظار دو هفته‌ای», clear next scan, retrieval ۸ مهر, transfer ۱۱ مهر | ✔ «برای دیدن شمارش معکوس…» note while beta is empty ([shot](ivf/CB-IVF-06b/j10-editor-tww-no-beta.light.png)) |
| 10 | Beta ۱۰ مهر (before the transfer) | ✔ client check «تست بتا نمی‌تواند پیش از انتقال باشد.», «ذخیره» disabled ([shot](ivf/CB-IVF-06b/j11-editor-beta-order-error.light.png)) |
| 11 | Beta ۲۳ مهر → «ذخیره» | ✔ stage `tww`, `stage_day 2`, `days_to_beta 11`; scan appointment deleted, retrieval / transfer / beta (08:00) appointments; home «روز ۲ انتظار», «تست بارداری · ۲۳ مهر» ([shot](ivf/CB-IVF-06b/j12-home-tww.light.png)) |
| 12 | Home «دو هفته انتظار» → `/ivf/tww` | ✔ ring «۱۱ روز تا تست خون (بتا)», «انتقال جنین: ۱۱ مهر» ([shot](ivf/CB-IVF-06b/j13-tww-countdown.light.png)) |

### Fidelity (390 px, light + dark)

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_IVF_Home` (entry) | `/ivf` (TWW) | [home](ivf/CB-IVF-06b/fa_ivf.light.png) | [home](ivf/CB-IVF-06b/fa_ivf.dark.png) | ✔ | Board order unchanged; one addition under the StepTimeline: «ویرایش» pill (44 px, pencil, `--brand-strong`) → `/ivf/cycle`. |
| `nbl_IVF_Home` (empty) | `/ivf` (no cycle) | [start](ivf/CB-IVF-06b/fa_ivf_start.light.png) | [start](ivf/CB-IVF-06b/fa_ivf_start.dark.png) | ✔ | Same card; the CTA opens the setup instead of starting with defaults. |
| — (no board; minimal per DECISIONS #8, `ivfm-` form look) | `/ivf/cycle/new` | [setup](ivf/CB-IVF-06b/fa_ivf_cycle_new.light.png) | [setup](ivf/CB-IVF-06b/fa_ivf_cycle_new.dark.png) | ✔ | Protocol chips (catalog `ivf_protocols`, optional, second tap clears; bundled names as fallback), «الان کجای درمان هستی؟» prep / stim, start (+ stim start) dates, note on adding dates later. |
| — (no board) | `/ivf/cycle` | [editor](ivf/CB-IVF-06b/fa_ivf_cycle.light.png) | [editor](ivf/CB-IVF-06b/fa_ivf_cycle.dark.png) | ✔ | Six stage chips (catalog titles), protocol, dates card: start / stim start, next scan, retrieval, transfer (date + time), beta; clear buttons; same order checks as the API. |
| `nbl_IVF_Meds` | `/ivf/meds` | [paused meds](ivf/CB-IVF-06b/fa_ivf_meds.light.png) | [paused meds](ivf/CB-IVF-06b/fa_ivf_meds.dark.png) | ✔ | Inventory rows of paused medicines show a neutral «متوقف» pill instead of «کم است» / «کافی». |
| — (med form) | `/ivf/meds/17` | [form](ivf/CB-IVF-06b/fa_ivf_meds_17.light.png) | [form](ivf/CB-IVF-06b/fa_ivf_meds_17.dark.png) | ✔ | New first card «در برنامه» switch + hint (edit only). |
| `nbl_IVF_TWW` | `/ivf/tww` | [countdown](ivf/CB-IVF-06b/fa_ivf_tww.light.png) | [countdown](ivf/CB-IVF-06b/fa_ivf_tww.dark.png) | ✔ | Unchanged screen, now reached from the UI. |

**Verdicts:** all ✔, no ✘.

**Open / TODO (ask user):**
- The stop prompt covers `stimulation` + `suppression` medicines (not trigger, not luteal support) and only on a move
  from prep/stim to retrieval or later — [needs clinical review].
- The home empty-state body still says the cycle «از آماده‌سازی شروع می‌شود» (CB-IVF-02 copy); setup can now start at
  stimulation.
- Pausing a medicine refreshes the IVF reads only; a `/care` screen already open refetches on its own stale time.
