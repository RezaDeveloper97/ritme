<!-- Read-only frontend sweep (2026-10-21), second pass over frontend/src for Engine v6. Input to M9 frontend tasks. -->

# Engine v6 → frontend impact

This was a read-only investigation. Nothing in the repo was created, edited or deleted, and no builds or tests were run.

**Path shorthand:** `FE/` = `frontend/src/`, `MSG/` = `frontend/messages/`, `BE/` = `backend-go/`, `SPEC` = `docs/engine-v6/SPEC.md`. All paths are relative to the repo root. I re-checked the line numbers against the current working tree on `stage`.

## Main risks
1. **The whole Home can fail to load.**
   - The zod objects don't reject unknown keys (they just drop them), but some fields are required and can't be null:
     - `predictions.*` dates (FE/entities/cycle/api/schema.ts:154-158)
     - `metrics.effective_*` (:259-260)
     - the three value layers `profile_values` / `calculated_values` / `effective_values` (:299-301)
     - the old per-day `calculation.estimated_ovulation_day` (:63)
   - The Go backend sends an empty date as `null` (BE/internal/platform/civildate/civildate.go:153-158).
   - `.parse` throws (FE/entities/cycle/api/queries.ts:43,66,127). The query then errors, `loadFailed` is set, and Home shows its error card (FE/screens/home/ui/HomePage.tsx:936,1088-1095).
   - Most likely trigger: hormonal contraception, where v6 computes no ovulation and no fertile window (SPEC:588).
2. **All new v6 fields are silently dropped**: `future_cycles`, `clinical_flags`, `suggest_pregnancy_test`, `anchors.ovulation_source`, `metrics.cycle_regularity`. Nothing errors, and nothing shows them.
3. **Wrong source priority.** `deriveCycleSchedule` uses the calendar-only `predictions` block before the engine's `anchors` (FE/entities/cycle/model/schedule.ts:94-103).
   - `predictions` is always `Predict(lastStart,…,lastStart)` (BE/internal/cycle/view/builder.go:127; BE/internal/cycle/view/services.go:30-53).
   - Once v6 moves the next period or ovulation based on BBT/LH (SPEC:322-324, 336-340), every screen would still show the plain calendar date.
4. **A late period is shown as if a new cycle had started.**
   - When `main_phase=unknown`, Home falls back to the old engine's phase.
   - The prediction rows roll forward by one cycle length (L).
   - The old month data paints late days as a predicted period.
   - The "my period started" prompt has no upper limit.
   - Every lateness level gets the same "a few days late is usually normal" text.
5. **Fertility is hidden only for teen and menopause.** Under hormonal contraception (forbidden, SPEC:570) these places would still show or compute a fertile window or ovulation:
   - Home
   - Calendar
   - the onboarding ReadyStep
   - the phase sheet
   - the partner's (companion) bar
   - the fertile-window reminder
6. **No translation keys exist** for any v6 banner, warning, ovulation source or regularity value.

## A) Schema / type inventory
**General notes:**
- No schema uses `.strict()` or `.passthrough()`, so unknown keys are dropped.
- Enum-like strings are cast with `as`. Unknown values never throw, but they are never checked either.

| Block | Schema (FE/entities/cycle/api/schema.ts) | Throws if missing/null | Has a default | Enum handling | v6 gap |
|---|---|---|---|---|---|
| today/date wrapper | :335-347 | — | `calculation` and `cycle_view` can be null | — | — |
| `calculation` (old engine, per day) | :54-96. Turned into null only when `cycle_day==null` (:107-113) | calculation_date, cycle_day, phase, estimated_ovulation_day, cycle_length_used (:59-64) | flags, probability, tips. Tips are parsed one by one (:73-79) | phase is a free string | null ovulation day with a non-null cycle_day → throws |
| `cycle_view` | :276-332 | `date` and the 3 value layers (:299-301, no default) | every other field is nullable with a default | main_phase, fertility_level, confidence, data_quality, resolution_source are casts (:310-322) | The SPEC:500-528 example has no value layers, `predictions` or `daily_card`. Missing value layers → throws |
| `anchors` | :234-254 | — | all have defaults; sources are free strings, default `unknown` | — | no `ovulation_source` (SPEC:271,518) |
| `metrics` | :257-269 | effective_cycle_length, effective_period_length (:259-260) | cycle_variability | — | no `cycle_regularity` (SPEC:444,520-521) |
| `predictions`, renamed to `forecast` | :152-174 | **all 5 dates, non-null** (:154-158) | source, confidence, reasons | — | null ovulation or window under contraception → throws |
| `data_quality_details` | :215-231 | — | all | regularity_status defaults to the old `not_enough_data` | — |
| `daily_card` | :124-150 | title, data_status | — | fertility_level is `z.enum(very_low…very_high).catch('low')` (:131-133) | The backend already sends `unknown` (BE/internal/cycle/view/daily_card.go:85-91), and v6 adds `peak`/`unknown`. Both silently become `low`. Not visible today: only `subtitle` is read (HomePage.tsx:744) |
| month wrapper | :349-378 | each day as in `calculation`; one bad day fails the whole array | month_summary is optional and never read | — | The backend still runs the old engine for every day, future months included (BE/internal/cycle/service/service.go:165-182) |
| TS types | FE/entities/cycle/model/types.ts: ConfidenceLevel :130, MainPhase :133-139, CycleFertilityLevel :142, DataQualityLevel :145, ResolutionSource :148-154, CycleAnchors :161-169, CycleEngineMetrics :172-177, CycleForecast :201-211 (non-null strings), CycleView :244-276 | compile-time only | | | Adding required fields breaks the full test object in FE/entities/cycle/model/schedule.test.ts:37-88 |

**Other cycle-related parsers:**
- **Home cycle overview** (`/home/cycle-overview`, FE/entities/cycle/api/overview.ts:24-53):
  - `regularity` is a free string, and the backend sends the old enum (BE/internal/home/cycle_overview.go:44,57).
  - Next period and PMS come from the engine (:63-67).
- **Unused sections hooks.** FE/entities/cycle/api/sections.ts:82-84 uses a strict `z.enum`, but nothing uses its hooks.
- **Companion.** FE/entities/companion/api/companion-side.ts:71-91 has every field optional or with `.catch`. It has no "fertility hidden" flag.
- **Analysis hub.** FE/entities/analysis/api/schema.ts:85-86 parses regularity with `.catch('not_enough_data')`, so v6 `insufficient_data` falls back safely. Its `cycle_status` values match v6 `clinical_flags.frequency`.
- **/cycle history.** FE/screens/cycle/model/schema.ts:25,41 maps an unknown regularity to `not_enough_data`.
- **Analysis TTC.** FE/screens/analysis-ttc/api/ttc.ts:42 accepts `bbt`/`lh`/`estimate` (BE/api/openapi.yaml:583). v6 uses `bbt_confirmed`/`lh_positive`/`calendar`/`unknown`.
- **Period history.** FE/features/log-period/api/mutations.ts:122-126 has no source and no skipped flag.
- **OpenAPI.** `CycleView` (BE/api/openapi.yaml:30813-30945) has required non-null `Predictions` dates and none of the v6 fields.

## B) Consumers
| file:line | layer | fields | renders | v6 impact |
|---|---|---|---|---|
| FE/entities/cycle/model/schedule.ts:65-122 | entity | anchors start/end; forecast next/ovulation first (:94-103); lengths metrics → effective_values → calc (:79-92) | the shared schedule used by Home, Calendar and onboarding | **wrong order**; makes up ovulation as next−14 (:103) |
| schedule.ts:141-160, 232-247 | entity | old-engine day phase + schedule rolled forward whole cycles | calendar grid, ring and week-strip dots | **wrong**: unlimited client roll instead of future_cycles; late days read from the old engine |
| FE/screens/home/ui/HomePage.tsx:175-201 | screen | 3 × `/cycle/month` + schedule | ring and week markers | same as above |
| HomePage.tsx:652-687 | screen | overview next_period/pms_window, schedule ovulation/window | prediction rows, TTC card (:1110-1118) | **wrong when late**: a past due date shows «در جریان» (ongoing), then jumps ahead by L. Window and ovulation shown under contraception. Only `teen` hides them (:1170-1172), and teen never reaches CycleHome (:1308-1329) |
| HomePage.tsx:706-730 | screen | old-engine day; cycle_view main_phase / days_to_period / cycle_day; effective_values cycle_length | ring numbers, phase card | **wrong**: `unknown` (more than 14 days late, SPEC:387) falls back to the old phase (:723, :937-939) |
| HomePage.tsx:738-742 | screen | fertility_level | chance pill | OK (`unknown` is hidden) |
| HomePage.tsx:778 + FE/screens/home/model/cycle-data.ts:11-13 | model | current_period_start, cycle_day | "no data" state | **gap**: a `predicted_reference` start counts as real data, and there's no "when was your last period?" banner |
| HomePage.tsx:827-837 → FE/screens/home/model/hero-state.ts:41-60 | model | days_late, days_to_period, end_is_confirmed, cycle_day, period length | button inside the ring + end-missing alert (:1078-1086) | **wrong** vs SPEC:599-602 (see D11) |
| HomePage.tsx:840, 901-915 | screen | days_late | `nb.lateText` | **wrong**: one text for every lateness level (MSG/fa/home.json:260), no pregnancy-test button |
| HomePage.tsx:845-929 | screen | days_to_ovulation, main_phase | TTC ring, chance text, LH tip | OK (handles null; v6 sends null, SPEC:497) |
| HomePage.tsx:943-957 + FE/screens/home/ui/PredictionsCard.tsx:25-29, 83, 115-119 | screen | overview regularity | regularity pill | **gap**: v6 values aren't in the map, so the pill silently disappears |
| PredictionsCard.tsx:74-79 | screen | — | ovulation row always says «تخمینی» (estimated) | **gap**: SPEC:606 wants the source shown |
| HomePage.tsx:959-985 | screen | profile, calculated and effective values | cycle-length sync suggestion | **breaks** if the value layers are removed |
| FE/screens/calendar/ui/CalendarPage.tsx:136, 574-580 | screen | life mode | hide fertility + legend | **gap**: nothing for contraception |
| CalendarPage.tsx:164, 227-250 | screen | profile period length, period history | open period painted as "logged" | **wrong**: uses the profile length and ignores whether the end is confirmed (SPEC:275-282) |
| CalendarPage.tsx:198-225, 284-293, 395-397, 596-606 + FE/screens/calendar/model/useCalcMap.ts:26-62 | screen | old-engine days + schedule; day sheet cycle day from the old engine (:599) | grid colours, day sheet | as in the schedule rows; a month parse error hides the grid |
| FE/screens/phase-details/model/tabs.ts:37-54 (ui/PhaseDetailsSheet.tsx:41-68) | screen | main_phase, anchors, metrics | «روز X تا Y» (day range) | **wrong** fallback: ovulation = L−14 (:44) |
| FE/screens/onboarding-flow/ui/ReadyStep.tsx:43-65 | screen | schedule | next period + fertile window | **wrong** under contraception. Onboarding collects `contraceptive_pill`/`iud` (FE/screens/onboarding-flow/model/state.ts:54) |
| FE/screens/checkup-self-exam/ui/SelfExamPage.tsx:58-68 | screen | old-engine day first | exam-window chip | minor: rolled day when late |
| FE/features/log-day/model/use-heading.ts:33-34 | feature | cycle_view.cycle_day | log-sheet heading | OK |
| FE/screens/fertility-insights/ui/FertilityInsightsPage.tsx:325-340 | screen | metrics.effective_period_length | history strip | OK |
| FE/widgets/fertility-tiles/model/ttc.ts:12-47 | widget | main_phase, level | TTC tiles | OK |
| FE/screens/companion-home/ui/CompanionHomePage.tsx:64-148 + model/phase-bar.ts:16-39 | screen | companion day, length, phase | partner's bar | **wrong** maths (see C); no flag to hide fertility |
| FE/screens/cycle-settings/ui/CycleSettingsPage.tsx:47,195; FE/screens/notification-settings/model/settings.ts:13,29 | screen | — | `fertile_window` reminder | **gap**: not hidden under contraception |

**Parsed but never read outside FE/entities/cycle (checked by grep):** `warnings`, `confidence_reasons`, `confidence`, `data_quality(_details)`, `resolution_source`, `is_predicted`, `requires_user_input`, `data_status`, `current_period_*_source`, `cycle_variability`, `month_summary`, and the forecast's `fertile_window_*`.

## C) Cycle maths done in the browser
| file:line | computes | rule | copies engine? | differs from v6? | recommendation |
|---|---|---|---|---|---|
| FE/entities/cycle/model/schedule.ts:94-103 | next period, ovulation | forecast → anchors → start+L; next−14 | yes | **yes**: wrong order; makes up ovulation when fertility is hidden | use anchors only; no fallback when the source is unknown or fertility is hidden |
| schedule.ts:108-133 | shown fertile window, PMS | max(O−5, period end+1)..O; next−3..−1 | yes | no (SPEC:349-357, 418, 606) | keep; ask the backend for per-cycle window fields |
| schedule.ts:141-160 `cycleScheduleFor` | later cycles | shift by n·L, no limit | yes (old `rollForward`, BE/internal/cycle/legacy/engine.go:201-210) | **yes**: v6 sends only k=1..6, with k≥2 at most medium confidence (SPEC:328-331) | look up `future_cycles` |
| schedule.ts:168-170 | countdown | max(0, next−date) | yes | equal when the view is present | keep only as a loading fallback |
| schedule.ts:185-199, 232-247 | marker per day | period from the old engine / first Lp days; window from the schedule | yes | **yes**: late days shown as period | replace (F4) |
| schedule.ts:173-177, 205-215 | progress %, window as cycle days | — | — | — | not used → delete |
| FE/entities/cycle/model/predictions.ts:43-48 | phase cleanup | unknown → luteal | — | **yes**: `unknown` is a real v6 state | map to null |
| predictions.ts:58-93 | reads old-engine days with the new rules | drops O+1; PMS 7 days → 3 | yes | no | keep only while the old month data is used |
| predictions.ts:143-166 | day offsets | L−day+1 etc. from the old engine | yes | **yes** when late | stop using it as the phase / cycle-day fallback (HomePage.tsx:710,723,728,937-939) |
| FE/entities/cycle/model/phase.ts:23-52 | day info | window O−5..O+1 (:40) | yes | **yes** | not used → delete |
| highlights.ts:19-28; markers.ts:80-118; fertilityStyle.ts:13-28; ui/DailyStatusCard.tsx (all in FE/entities/cycle/) | various | old engine | — | DailyStatusCard would show `unknown` as "low" | only exported from FE/entities/cycle/index.ts:2-31 → delete |
| HomePage.tsx:652-661 `slotFor` | which occurrence a row shows | roll a past row forward by L | yes | **yes** when late | don't roll the period row while days_late>0 |
| FE/screens/home/model/hero-state.ts:39-60 | prompt rules | near period ≤2 days; any late; end prompt from day 1; end missing when day > Lp | yes | **yes** (SPEC:599-602) | replace with a pure §2.43 mapper |
| FE/screens/calendar/ui/CalendarPage.tsx:227-228 | end of open period | start + profile length − 1 | yes (§2.11) | **yes** | use anchors.current_period_end + is_confirmed |
| FE/screens/phase-details/model/tabs.ts:37-54 | day range per phase | ovulation fallback L−14 (:44); fertile from Lp+1 | yes | **yes**: off by one, since v6 ovulation is cycle day L−13 (SPEC:340,343); fertile start comes from current_period_end (SPEC:355) | use L−13 or null; use the period end |
| FE/screens/companion-home/model/phase-bar.ts:16-39 | partner bar | period fixed at 5; ovulation L−14; fertile O−4..O | yes | **yes**: off by one, and 5 days instead of 6 | align with v6 |
| FE/screens/cycle/model/bars.ts:71-79 | typical bands | fertile O−5..O+1; PMS last 5 days | yes | **yes** | O−5..O; PMS last 3 days |
| FE/screens/analysis-cycle/model/cycle.ts:14-19 | day layout | ovulation L−13; max(Lp+1, O−5)..O | yes | no | keep |
| FE/screens/fertility-insights/model/view.ts:166-184 | history strip | O−5..O | yes | no | keep |
| FE/screens/checkup-self-exam/model/view.ts:41-45 | days until exam window | modulo L | partly | minor | feed it the v6 cycle_day |

## D) Impact of each v6 change
1. **Fertile window O−5..O** (SPEC:349-350; change list SPEC:19).
   - Home and Calendar already end the window on ovulation day (schedule.ts:108-121; predictions.ts:58-81).
   - Still wrong: bars.ts:76, phase-bar.ts:27-30, and phase.ts:40 (not used).
   - The forecast's `fertile_window_end` is still O+1 (BE/internal/cycle/view/services.go:50-51), but the frontend never reads it.
2. **Ovulation source and BBT/LH priority** (SPEC:271, 322-324, 336-345).
   - `ovulation_source` is dropped on parse.
   - Wrong source priority (schedule.ts:94-103).
   - The ovulation row always says «تخمینی» (PredictionsCard.tsx:78).
   - Translation keys needed: `home.nb.predictions.ovulationSource.{bbt_confirmed,lh_positive,calendar}`.
   - TTC analysis has its own `bbt/lh/estimate` labels (MSG/fa/analysis.json:559-561, 620-624). Renaming them has to be coordinated with BE/api/openapi.yaml:583.
3. **`future_cycles`** (SPEC:328-331, 523; must never be stored, :572).
   - Dropped on parse.
   - Future days are painted from two different sources: periods from the old engine's month data, and window/PMS from the browser's own roll forward. These disagree once BBT/LH moves the next period.
   - Storage is fine: no query-cache persister exists, and the service worker never caches `/api/*` (frontend/public/sw.js:18,151).
4. **`clinical_flags`** (SPEC:447-452, 522).
   - Dropped on parse.
   - §2.43 doesn't put them on Today, and Analysis already gets FIGO data from its own endpoint (MSG/fa/analysis.json:291-292).
   - Parse them as nullable for later use.
5. **`metrics.cycle_regularity`** (SPEC:442-445, 606).
   - Dropped on parse.
   - The pill reads the overview's old enum, so v6 values make it disappear.
   - Translation keys needed: `home.nb.predictions.regularity.{regular,irregular,insufficient_data}`. Only the old keys exist (MSG/fa/home.json:297-300, MSG/en/home.json:297-300).
6. **Lateness levels and `suggest_pregnancy_test`** (SPEC:384-390).
   - The flag is dropped on parse.
   - Home shows "late" for any days_late>0, and the start prompt has no upper limit (hero-state.ts:52,56).
   - `lateText` (MSG/fa/home.json:260) is shown even at 8+ or 15+ days late.
   - When `unknown`, Home falls back to the old phase.
   - There is no late banner and no "log a test" button.
   - A `pregnancy_test` measurement already exists (FE/features/log-day/ui/panels/MeasurePanel.tsx:33; MSG/fa/log-taxonomy.json:421). The only target is `/fertility/log` (HomePage.tsx:895); I found no direct link to the measurement panel.
   - I found no "switch to pregnancy mode" prompt after a positive test (SPEC:591). I searched features/log-day, screens/home and widgets.
   - Translation keys needed: `home.nb.banner.late.*`.
7. **Hormonal contraception** (SPEC:238, 570, 588).
   - This is where a parse error is most likely (see A).
   - Fertility content appears on Home (HomePage.tsx:676-682, 801-806, 1110-1118, 1167-1175), the Calendar (:136), ReadyStep (:53-62), the phase sheet, the companion bar and the reminder.
   - The frontend has no signal for it: `warnings` is never read and there's no flag.
   - Missing copy: «خون‌ریزی قطع مصرف» (withdrawal bleed) and the Today disclaimer «پیش‌بینی روش پیشگیری نیست» ("predictions are not a contraceptive method", SPEC:608). The disclaimer exists only on the fertility screens (MSG/fa/fertility.json:171,192).
   - Existing code to reuse: `withoutFertility` / `NO_FERTILITY_LEGEND` (FE/widgets/cycle-calendar/model/tones.ts:16-21) and `hideFertility` in PredictionsCard (:53-54, :81).
8. **New warnings and confidence reasons** (SPEC:474, 490).
   - They parse fine (string arrays, schema.ts:291,297) but are never read.
   - No translation keys exist. Only the §2.43 banners and the §2.42 explanation lines need text; raw codes should never be shown.
9. **`requires_user_input`, stale anchor, `predicted_reference` / `onboarding_declared` sources** (SPEC:269-270, 492, 573-582, 604).
   - Parsed but not read.
   - `needsPeriodData` treats a `predicted_reference` start as real data.
   - `periodOpen = !is_confirmed` (HomePage.tsx:832) might show «پریودم تموم شد» ("my period ended") on a period that was only projected. This is likely but I haven't verified it.
   - The "when was your last period?" banner is missing. PeriodDateEditor already exists (HomePage.tsx:1215).
10. **Period caps** (SPEC:284-297).
    - The end-missing alert uses the client rule cycleDay>Lp, and only while the end prompt is showing (hero-state.ts:43-48). Tapping «هنوز ادامه دارد» ("still going") hides the banner too.
    - The end prompt shows from day 1 with no upper limit.
11. **§2.43 Today mapping** (SPEC:595-608):

| Element | SPEC | Now | Gap |
|---|---|---|---|
| start button | :599 period not open, next−3..next+14 | near period (≤2 days) or any lateness (hero-state.ts:39,52-56) | −2 instead of −3; no +14 limit |
| end button | :600 end unconfirmed and days_since in [Lp−1, 12] | any open period (:43) | no lower or upper limit |
| edit button | :601 | ✅ HomePage.tsx:1070-1075 | — |
| end-missing banner | :602 cap warnings | client rule (:48) | wrong source |
| late banner + test button | :603 | none | missing |
| last-period banner | :604 | none | missing |
| at most one banner, in table order | :608 | nothing chooses between banners | missing |
| ring text length limits | :605 | not checked | not verified |
| predictions card | :606 | period, PMS and window ✅; ovulation without source; old regularity values | partly done |
| dismissals kept locally for the same day | :608 | ✅ FE/screens/home/model/prompt-dismiss.ts:3-21 | — |
| disclaimer | :608 | missing | missing |

12. **Skipped cycle «پریود نداشتم»** ("I had no period", SPEC:559). There is no UI and no API client for it (log-period only has start/end/update/delete/history). It needs a backend endpoint first.
13. **§2.40 "21–45 is not normal"**: compliant. The only 21–45 mention is the calculation-filter text (MSG/fa/analysis.json:274), which matches SPEC:242-248. The /cycle history "in range" check uses median±spread (BE/internal/cycle/insights/history.go:60-62).

**Translation keys to add** in MSG/{fa,en}/home.json, then copy to BE/resources/translations/{fa,en}/ (see frontend/CLAUDE.md:325-332). Write the fa copy in the M8 style.
- `nb.banner.{late,lastPeriod}.*`
- a hard-cap version of `nb.endMissing`
- `nb.predictions.ovulationSource.*`
- `nb.predictions.regularity.{regular,irregular,insufficient_data}`
- `nb.predictions.withdrawalBleed`
- `nb.disclaimer`
- postpartum / 45+ explanation lines
- skipped-cycle text, once the backend supports it

## E) Tests that lock in current behaviour
- **FE/entities/cycle/model/schedule.test.ts**
  - :37-88 builds a full `CycleView` object, with more anchors/metrics objects at :114-122, :309-318, :418-427. These stop compiling when required fields are added.
  - :93-107 has forecast and anchors agreeing. Add a case where they disagree.
  - :185-202 and :252-258 check unlimited rolling ("rolls into the next cycle once the predicted start is reached").
  - :217-225 tests code that isn't used.
  - :257-281 is a generator of old-engine days (ovulation+1, PMS from L−6). :303-460 tests `cycleDayMarkerAt`.
- **FE/entities/cycle/model/predictions.test.ts**: :42-46 checks unknown → luteal; :88-158 checks offsets and the O+1 → luteal reading.
- **Tests for unused code**: FE/entities/cycle/model/phase.test.ts:16-47 (O+1), highlights.test.ts:21-52, markers.test.ts:24-84.
- **FE/entities/cycle/api/schema.test.ts:11-60** covers daily tips only. There is no parse test for `cycle_view`, forecast or month.
- **FE/screens/home/model/hero-state.test.ts**: :23-28 (near ≤2 days), :29-32 (late with no limit), :33-40 (end prompt on day 3 and end-missing above Lp, both hidden after dismissing), :63-70.
- **FE/screens/home/model/cycle-data.test.ts:16-30** has no `predicted_reference` case.
- **Other maths tests**: FE/screens/phase-details/model/tabs.test.ts:27-29 (L−14), FE/screens/companion-home/model/phase-bar.test.ts:6-26, FE/screens/cycle/model/bars.test.ts:47-51, FE/widgets/cycle-calendar/model/tones.test.ts:5-40.
- **Already match v6** (keep as regression tests): FE/screens/analysis-cycle/model/cycle.test.ts:25-26, FE/screens/fertility-insights/model/view.test.ts:141-152, FE/widgets/fertility-tiles/model/ttc.test.ts:6-20.
- There are no e2e tests for the cycle screens.

## F) Proposed frontend tasks (web only; no `tasks/M9` exists yet)
**F1: Make cycle parsing safe for v6** (entity only; do this first)
- **Scope:**
  - Allow null for `predictions` dates (or `.catch(null)` on the block), for `metrics.effective_*` and for the value layers. Consumers fall back to `metrics`.
  - Allow a null ovulation day in the old-engine data, and parse month days one by one.
  - Parse `futureCycles`, `clinicalFlags`, `suggestPregnancyTest`, `anchors.ovulationSource` and `metrics.cycleRegularity`.
  - Add `peak` and `unknown` to the daily-card level.
  - In `deriveCycleSchedule`, use anchors first and the forecast only for an old backend; ovulation can be null.
  - Add an `isFertilitySuppressed(view)` helper.
- **Paths:** FE/entities/cycle/api/schema.ts, model/types.ts, model/schedule.ts, index.ts, and their tests.
- **Backend:** nothing needs to merge first (still works with v1.1). Agree on: whether `predictions`, the value layers and `daily_card` stay in v6; nullable dates; the OpenAPI update.
- **Tests:** vitest parse cases for the current payload, the SPEC:500-528 example, a contraception payload with nulls, a payload with no value layers, and a month with one bad day. Plus "anchors win over forecast" and "no window when fertility is hidden". Then typecheck and lint.

**F2: Today screen §2.43 + lateness levels**
- **Scope:** a pure `todayUi()` model with:
  - the start/end/edit button windows
  - one banner chosen in table order: end-missing → late (with a test button when `suggest_pregnancy_test`) → last period
  - no old-phase fallback when `unknown`
  - no rolled period row while late
  - ovulation source and the `cycle_regularity` pill on the predictions card
  - the disclaimer and new late text
- **Paths:** FE/screens/home/model/hero-state.ts (+ new today-ui.ts), HomePage.tsx (:652-687, :723, :827-915, :1038-1086, :1166-1175), PredictionsCard.tsx, MSG/{fa,en}/home.json plus the backend copy.
- **Backend:** the v6 warnings, `requires_user_input`, `suggest_pregnancy_test`, `cycle_regularity`, `ovulation_source`. The button target is `/fertility/log`.
- **Tests:** table-driven cases from SPEC §6.1 (SPEC:903-915):
  - days_late 7 / 8 / 15
  - open period at days Lp−2 / Lp−1 / 12 / 13
  - start window at next−4 / −3 / +14 / +15
  - several warnings → only one banner
  - `unknown`

  Rewrite hero-state.test.ts. Visual check in light and dark.

**F3: Hide fertility under hormonal contraception + restricted-mode text**
- **Scope:** use the F1 helper on:
  - Home rows, TTC card and ring dots (reuse `hideFertility`)
  - Calendar (`withoutFertility` / `NO_FERTILITY_LEGEND`)
  - ReadyStep
  - the phase sheet
  - the companion bar
  - the `fertile_window` reminder

  Add «خون‌ریزی قطع مصرف» and the postpartum / 45+ lines. The `teen` branches inside CycleHome can never run (HomePage.tsx:1308-1329); remove or merge them.
- **Backend:** a clear flag in `cycle_view` and `/companion/shared`, and a profile input for hormonal contraception. Whether an IUD counts as hormonal is a product decision (state.ts:54). The reminder must also be skipped on the server.
- **Tests:** vitest for the helper, the schedule, PredictionsCard rows and `withoutFertility`; visual check.

**F4: Calendar and ring from `future_cycles`; remove client rolling and unused code**
- **Scope:**
  - Replace `cycleScheduleFor` and the forward branch of `cycleDayMarkerAt` with a `future_cycles` day map (k≤6). Late days are not painted as a new period.
  - Take the open-period end from `anchors.current_period_end` + `is_confirmed` (CalendarPage.tsx:164, 227-250).
  - Delete phase.ts, highlights.ts, `markerIntensityByDate`, fertilityStyle.ts, DailyStatusCard, `cycleProgressPercent`, `fertileWindowDays`, `calcToPhase` and the unused sections hooks.
- **Backend:** answers to Q1–Q3 below. Ideally a v6 per-day month view to replace BE/internal/cycle/service/service.go:165-182.
- **Tests:** the SPEC:523 example, the k=6 boundary, "10 days late → no rolled period", confirmed vs assumed end. Rewrite schedule.test.ts:185-258 and :303-460.

**F5: Align the remaining browser maths with v6** (ovulation = cycle day L−13, window O−5..O, PMS 3 days)
- **Scope:** bars.ts:71-79, phase-bar.ts:16-39, tabs.ts:44 (+ fertile start from the period end), predictions.ts:43-48.
- **Backend:** none.
- **Tests:** update bars.test.ts:47-51, phase-bar.test.ts:6-26, tabs.test.ts:27-29 and predictions.test.ts:42-46. Add a shared 21/28/35-day test table (SPEC §2.36, :532) run against all four functions.

**Order:** F1 first; then F2, F3 and F5 in parallel; F4 once the backend contract is clear.

**Open questions for the backend:**
1. Does v6 keep `predictions`, the value layers and `daily_card`? None of them are in the SPEC:500-528 example.
2. Does the `ovulation` in a `future_cycles` entry belong to cycle k or to cycle k−1? In the example it equals period_start−14. Also: the SPEC says k runs 1..6, but the example starts at 2. Please add period_end, the window, PMS and confidence to each entry.
3. Can there be a clear fertility-suppression flag (also in the companion payload)?
4. Will there be a v6 per-day month view?
5. Will the TTC `ovulation_source` values be renamed?
6. Which enum should the overview `regularity` use?
7. What phase does a `predicted_reference` period day resolve to?
8. Is there a skipped-cycle endpoint?
