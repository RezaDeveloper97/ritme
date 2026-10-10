# COND — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png`. Screens: 390 px, fa (plus one en), headless Chrome over CDP against
the worktree Go API :8301 + Next dev :3123 (`ritme_dev`, goose 00047, not migrated). Test user **`09120002302`**
(created for CB-COND-02: onboarding name نیلا / female / goal cycle; enrolled in `endo` via the API; heavy flow logged
on 2026-10-03 and 2026-10-04 so `GET /messages/nudges` returns `heavy_bleeding`). Colours are checked against the token
map (`docs/canvas-build/README.md` §4), not the board hex.

## CB-COND-02

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Cond_Hub` | `/programs` | [hub](cond/CB-COND-02/fa_programs.light.png) · [join sheet](cond/CB-COND-02/fa_programs.sheet-join.light.png) · [en](cond/CB-COND-02/en_programs.light.png) | [hub](cond/CB-COND-02/fa_programs.dark.png) · [active sheet](cond/CB-COND-02/fa_programs.sheet-active.dark.png) | ✔ | Same order: back header «برنامه‌های مراقبت» (→ `/services`), 24/800 «برای شرایطی که داری» + lead, four program cards in API order (endo / PMDD / heavy bleeding / PCOS) with the tone disc (flame·period, brain·brand, drop·period, sprout·data), title + subtitle from catalog `condition_programs`, the «ثبت می‌کنی: …» line from meta `logs`; an enrolled card has the `--period` 1.5px ring and the «فعال» pill (tone data), the rest a chevron; «این برنامه‌ها تشخیص نمی‌دهند…» `InfoNote` from `condition_alerts/not_a_diagnosis`; bottom nav with «خدمات» active (`/programs` added to the nav hubs). Additions (not on the board): this cycle's nudge (`GET /messages/nudges`) as a soft period-tint card above the list with its in-app action link; a card opens a half sheet — not enrolled: «شروع برنامه» (POST `/conditions/enrolments`); enrolled: «فعال از <Jalali date>», open the diary (`/programs/pain`, `/programs/pmdd`, `/programs/bleeding`; PCOS → `/analysis/cycle`) and «ترک برنامه» (DELETE, entries kept). Verified through the UI: join PMDD → card turns active, sheet links `/fa/programs/pmdd`; leave → inactive again. Services: the three condition tiles (`pain_endometriosis`, `pmdd`, `heavy_bleeding`) that had no admin href now link `/programs` ([services](cond/CB-COND-02/fa_services.programs.light.png)); pelvic floor stays «به‌زودی», contraception keeps its href. |

## CB-COND-05

Worktree Go API :8331 + Next dev :3126 against `cond05_dev` (a copy of the `ritme_dev` dump, goose 00047, not migrated —
the test stack had been reset). Test users **`09120002305`** (onboarding نیلا / female / cycle; enrolled in
`heavy_bleeding`; PBAC yesterday light 1 + heavy 2 = 41, today light 1 · medium 2 · heavy 3 · clots none = 71 → period
day 2, score 112 — the board's numbers) and **`09120002306`** (same onboarding, not enrolled).

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Cond_Bleed` | `/programs/bleeding` | [chart](cond/CB-COND-05/fa_bleeding.light.png) · [en](cond/CB-COND-05/en_bleeding.light.png) · [not enrolled](cond/CB-COND-05/fa_bleeding.join.light.png) | [chart](cond/CB-COND-05/fa_bleeding.dark.png) · [edited](cond/CB-COND-05/fa_bleeding.edited.dark.png) | ✔ | Same order: back header «جدول خونریزی» + «روز ۲ پریود» (`period.day`; «امروز» without a period) → `/programs`; «نوارهای امروز» + hint; one card with three rows — pad pictogram (CSS, `--period` fill at 26 / 55 / 100 %), soak label, «هر عدد ۱ / ۵ / ۲۰ امتیاز», `NumberStepper` 0–50; «لخته» card with نداشتم / کوچک (اندازه سکه) / بزرگ chips (multi style, tone period, one at a time; tapping the active one clears it) and the «نشت از نوار به لباس» switch; «امتیاز امروز» / «امتیاز این پریود» tiles in Lalezar `--period-deep`; the ≥ `alert_threshold` card (`--period-soft` + `--period` border, warning disc) with the admin `condition_alerts/pbac_over_100` copy, own fa/en fallback while editing; sticky «ذخیره» footer, no bottom nav. The tiles and the alert recompute live while editing (period = saved period score − saved day score + live day score; [edited](cond/CB-COND-05/fa_bleeding.edited.dark.png): +1 light pad, large clot, flooding → ۸۲ / ۱۲۳). Save is enabled only when dirty; `clots` is sent only when changed (the API gates it on the mode's clot param). Verified through the UI: + then «ذخیره» → PUT saved (GET shows light 2, score 72) and the screen returns to `/fa/programs` (the board's flow). Addition: not enrolled → a join card («شروع برنامه», POST `/conditions/enrolments`) since PUT needs the enrolment. The board shows «کوچک» selected with a total of 71, which doesn't add up (small clot = +1); the seed uses «نداشتم». |
## CB-COND-03

Worktree Go API :8311 on a fresh goose-migrated `cond03_dev` (00054) + Next dev :3124. Test user **`09120002303`**
(profile via `POST /profile`, enrolled in `endo` via the API after the «join» shot; day 2026-10-08 saved with score 7,
pelvis + ovary, cramping + stabbing, dyschezia + bloating, ibuprofen 08:00 «کمی»). The dark shot was taken after a save through the UI with «لگن» toggled off (so only ovary is picked) — that save round-tripped through `PUT` as expected.

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Cond_Endo` | `/programs/pain` | [diary](cond/CB-COND-03/fa_programs_pain.light.png) · [not enrolled](cond/CB-COND-03/fa_programs_pain.join.light.png) · [en](cond/CB-COND-03/en_programs_pain.light.png) | [diary](cond/CB-COND-03/fa_programs_pain.dark.png) | ✔ | Same order: back header «دفترچه درد» / «اندومتریوز · امروز» (→ `/programs`), «کجا درد داری؟» + hint over a 3-column grid of square location chips (period tone), «چقدر شدید؟» `NumericScale` 0–10 solid (danger tone; wraps 0–5 / 6–10 so every cell stays ≥ 44px — the board has one row), «چه جور دردی؟» chips (catalog `pain_types`, period tone), «همراه با» chips (catalog `pain_associated`, brand tone), «اثر روی روزت و دارو» card with the missed-work `Switch` and the painkiller row (pill disc in data tone, name · time summary, effect chips کمک کرد / کمی / کمک نکرد in data tone), sticky «ذخیره» footer → `PUT /conditions/pain/{today}` then back to `/programs`. No bottom nav (form). Differences: the locations are the log taxonomy's `pain.location` options of her mode (abdomen, pelvis, ovary, back, head, breast) — the board's finer regions (above navel, lower abdomen L/M/R, thighs) are not in bloom's taxonomy (CB-COND-01 open item); chips rather than bloom's body-map figure, to match the board. Additions: painkiller name + time inputs (the board shows only the result line), «کمک نکرد» (API value `no`), the `condition_alerts/pain_scale` InfoNote (0–10 meaning + see a doctor the same day for severe/sudden pain, fever, fainting), a «join the program» card when not enrolled (save disabled until joined), client-side check of the score/location pairing. Relief options stay as the log sheet left them (not sent). |
## CB-COND-04

Screens were taken against a **mock of the Go API** on :8321 (shapes from the `conditions` contract goldens) and Next dev on
:3125. The docker test MariaDB stopped answering, and a container restart had already wiped `ritme_dev`. Earlier, the real
worktree API ran the flow end to end on a private `cond04_dev` copy of the dump: enrol, then PUT days, then the chart
returns `pattern: luteal` with 2 complete cycles.

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Cond_PMDD` | `/programs/pmdd` | [fa](cond/CB-COND-04/fa_pmdd.light.png) · [en](cond/CB-COND-04/en_pmdd.light.png) | [fa](cond/CB-COND-04/fa_pmdd.dark.png) | ✔ | Same order: back header «پرسشنامه خلق روزانه» / «PMDD · روز N سیکل» (→ `/programs`), card with the «۱ یعنی اصلاً، ۶ یعنی خیلی شدید» hint and six `NumericScale` 1–6 (soft, brand); questions come from catalog `pmdd_items` and are prefilled from today's `GET /conditions/pmdd/{date}`; «این سیکل» chart card (daily mean line over the current cycle, period band + last-10-days band, day 1 at the reading start, axis پریود / وسط سیکل / ۱۰ روز آخر) with the pattern sentence from `condition_alerts` (`pmdd_pattern_*` + `pmdd_needs_two_cycles`); the crisis note (`UrgentCard` note from `pmdd_crisis`, tel: 1480 / 123, fallback copy if the alert is missing); sticky «ذخیره» footer (PUT today). No bottom nav (form). Deviations: the board paints a single pink band. Here the last-10-days band is a brand tint and the period band is `--period-soft`, so the two can be told apart, with a small legend. Also added: a «not a diagnosis» InfoNote and a progress line «N از ۶ مورد امروز». If she isn't enrolled, a join card («شروع برنامه», POST `/conditions/enrolments`) replaces the form and the chart still shows. |

## CB-COND-07 — epic QA

Real stack, no mocks: worktree Go API :8371 on `cond07_dev` (a copy of a goose-00055 dev DB, `RUN_MIGRATIONS=false`) +
Next dev :3130 with `NEXT_PUBLIC_API_BASE_URL=http://localhost:8371/api/v1`, headless Chrome over CDP, 390 px, fa.
Test users: **`09120002307`** (onboarded نیلا / female / cycle, profile with last period 2026-10-01; enrolled in `endo`,
`pmdd` and `heavy_bleeding` through the API; 3 days 10-08 → 10-10 logged in each: pain score 5/6/7 pelvis + ovary,
PMDD six items, PBAC 26 / 32 / 83; two earlier 28-day cycles back-filled with PMDD scores low then high in the last
10 days so the chart has 2 complete cycles) and **`09120002308`** (same onboarding, not enrolled, heavy flow logged on
3 days). This replaces CB-COND-04's mock-based PMDD shots: the PMDD screen now runs on the real API.

### Journey

1. Fresh user finishes onboarding and the profile → `/programs` lists the four programs, none active.
2. Enrol in each program through `POST /conditions/enrolments` (the hub's sheet does the same call; checked in
   CB-COND-02) → `GET /conditions` returns all three `enrolled` (200 each).
3. Log three days in each: `PUT /conditions/pain/{date}` (200, «دفترچه درد ذخیره شد»), `PUT /conditions/pmdd/{date}`
   (200), `PUT /conditions/pbac/{date}` (200, scores 26 / 32 / 83).
4. Summaries: the pain diary shows today's saved score, locations, types, associated and the painkiller; the PMDD
   screen shows today's six answers (6 of 6), the cycle chart and the pattern sentence «علائم در ۱۰ روز آخر سیکل بالا
   می‌رود و با شروع پریود کم می‌شود» (`GET /conditions/pmdd/chart`: `complete_cycles` 2, `pattern` luteal); the bleeding
   screen shows «روز ۳ پریود», today 83, period 141 and the ≥ 100 warning.
5. Nudges: they fire only for users **not** enrolled in the program. `…308` (3 heavy-flow days, not enrolled) gets
   `heavy_bleeding` from `GET /messages/nudges` and the hub shows the card with «باز کردن جدول خونریزی»; `…307`
   (enrolled) gets `[]`, by design (CB-COND-06).

### Boards

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Cond_Hub` | `/programs` | [enrolled](cond/CB-COND-07/fa_programs.light.png) · [with nudge](cond/CB-COND-07/fa_programs.nudge.light.png) | [enrolled](cond/CB-COND-07/fa_programs.dark.png) · [with nudge](cond/CB-COND-07/fa_programs.nudge.dark.png) | ✔ | Matches CB-COND-02: four program cards, enrolled ones with the ring and «فعال» pill; the nudge card sits above the list in the period tint, readable in dark. |
| `nbl_Cond_Endo` | `/programs/pain` | [fa](cond/CB-COND-07/fa_pain.light.png) | [fa](cond/CB-COND-07/fa_pain.dark.png) | ✔ | Saved day round-trips from the API (score 7, pelvis + ovary, cramping, bloating, Ibuprofen 08:00 «کمی»). |
| `nbl_Cond_PMDD` | `/programs/pmdd` | [fa](cond/CB-COND-07/fa_pmdd.light.png) | [fa](cond/CB-COND-07/fa_pmdd.dark.png) | ✔ | Now on the real API: prefilled answers, chart with the period band (right, RTL) and last-10-days band (left), legend, pattern sentence, crisis note. With only 3 logged days in the open cycle the daily line is a short stroke; the older cycles feed the pattern. |
| `nbl_Cond_Bleed` | `/programs/bleeding` | [fa](cond/CB-COND-07/fa_bleeding.light.png) | [fa](cond/CB-COND-07/fa_bleeding.dark.png) | ✔ | Real PBAC numbers (83 today / 141 period) and the ≥ 100 warning; save is disabled until something changes. |

No drift fixed: nothing needed changing in the canvas-owned condition slices.

### CB-COND-02 open item — home nudge slot

Bloom's home has **no** condition-nudge slot (its only nudge is the profile-vs-recent-cycles sync nudge in
`HomePage.tsx`, shown in [home](cond/CB-COND-07/fa_home.nonudge.light.png) for a user with heavy flow: nothing about
bleeding). So the condition nudge is visible only on `/programs`. Follow-up task **CB-COND-07b** (home condition nudge
card) is queued in `roadmap/E05-cond/`.

## CB-COND-07b

Worktree Go API :8401 + Next dev :3133 against `cond07b_dev` (copy of the CB-COND-07 DB). User **`09120002308`** (not
enrolled, heavy flow on 3 days this cycle, so `GET /messages/nudges` returns `heavy_bleeding`). 390 px.

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Cond_Hub` (nudge card) | `/home` (cycle mode) | [fa](cond/CB-COND-07b/fa_home.light.png) · [en](cond/CB-COND-07b/en_home.light.png) · [enrolled: card gone](cond/CB-COND-07b/fa_home.enrolled.light.png) | [fa](cond/CB-COND-07b/fa_home.dark.png) · [en](cond/CB-COND-07b/en_home.dark.png) | ✔ | Same card as the hub (`ConditionNudgeCard`, now in `entities/condition/ui`), placed right under «دسترسی سریع», hidden for teens. Shown only when `/messages/nudges` has a hit; the action link opens `/programs/bleeding`. After enrolling (POST `/conditions/enrolments`) the nudge list is empty and the card is gone (`homeNudges` also drops hits for enrolled programs). Unit test `home-nudges.test.ts`. No new message keys (reuses `conditions.nudge`); `conditions` added to the `home` and `services` scopes. |
