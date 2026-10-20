# M9 Engine v6 — frontend consumers gap analysis (02b)

Read-only analysis of `frontend/src` against `docs/engine-v6/SPEC.md` §2.34–§2.43. All paths are relative to `frontend/src/` unless noted.

## 1. Summary

- **One boundary, two engines.** `entities/cycle/api/schema.ts` parses `/cycle/today` and `/cycle/date/{d}` as `{calculation (legacy), cycle_view (v1.1)}`. `/cycle/month` returns legacy `calculation[]` only. Every screen mixes the two: the engine's `cycle_view` fields win when they exist, and the legacy `calculation` plus local math fill in when they don't.
- **The v6 fields are not modelled yet.** None of `future_cycles`, `clinical_flags`, `suggest_pregnancy_test`, `metrics.cycle_regularity` or `anchors.ovulation_source` is in zod or the types. Zod objects strip unknown keys, so adding them on the backend is harmless today. The UI just never shows them.
- **Parse-crash risks under v6** (an envelope parse failure blanks home, calendar, onboarding and phase sheet):
  - `forecastSchema` requires every date as a string (schema.ts:152-162). Go's `civildate.Date` marshals a zero date as `null` (backend-go/internal/platform/civildate/civildate.go:154). If v6 nulls `predictions.fertile_window_*` or `estimated_ovulation_date` under hormonal contraception or for unknown ovulation, the whole `cycle_view` throws.
  - `engineMetricsSchema` requires `effective_cycle_length` and `effective_period_length` as numbers (schema.ts:257-269).
  - `profile_values`, `calculated_values` and `effective_values` are required with no default (schema.ts:299-301).
  - `daily_card.title` and `data_status` are required (schema.ts:124-128).
- **Client math duplicates the engine in 8 places.** `deriveCycleSchedule` / `cycleScheduleFor` / `cycleDayMarkerAt`, `deriveCyclePredictions`, `calcInPmsWindow`, `analysis-cycle dayLayout`, `companion phaseBar`, `phase-details phaseDayRange`, `fertility-insights historyStrip` and `cycle/model/bars typicalBands`. The worst case is `deriveCycleSchedule` (entities/cycle/model/schedule.ts:99-103): **it invents an ovulation date** (`calc.estimatedOvulationDay`, else `nextPeriod − 14`) when the engine gives none. So the home, calendar and onboarding would still paint a fertile window when v6 suppresses it. `typicalBands` still draws the old O−5..O+1 window.
- **Hormonal contraception: the fertile window is never hidden** on any surface. Only teen and menopause hide it (home `hideFertility={teen}`, calendar `hideFertility` for teen/menopause, analysis variant). This breaks §2.40 today.
- **§2.43 Today screen is partly built.** `screens/home/model/hero-state.ts` has start/end prompts and the per-day local dismissals. The windows don't match the spec:
  - start prompt: 0..2 days before, or late with no upper bound. Spec: −3..+14.
  - end prompt: from period day 1. Spec: from L−1 up to the hard cap.
  - «پایان پریود ثبت نشده» is computed locally. Spec: driven by warnings.
  - Missing entirely: the late-period banner at ≥ 8 days with the pregnancy-test button, the «آخرین پریودت کی بود؟» banner, the ovulation source, the max-one-banner rule and the contraception disclaimer.
  - `cycle_view.warnings` and `requires_user_input` are parsed but **read by nothing**.
- **Regularity enum mismatch.** The home pill maps `relatively_regular | irregular_possible | not_enough_data` (`screens/home/ui/PredictionsCard.tsx:25-29`). v6 sends `regular | irregular | insufficient_data`, so the pill would silently disappear.
- **Cache refresh gaps.** Period, profile and cycle-settings writes don't refresh `fertilityKeys`, `analysisKeys` or (partly) `messageKeys`. Contraception save/stop never refreshes `cycleKeys`, which matters in v6 because contraception changes `cycle_view`.
- **Tests.** There is no `cycleViewSchema` test at all. The existing `schedule`, `predictions`, `markers`, `hero-state`, `phase-bar`, `bars` and `analysis-cycle` tests lock in the pre-v6 math.

## 2. Types / zod inventory (`entities/cycle/api/schema.ts`)

The style throughout is non-strict `z.object`, so unknown keys are stripped. Enums are mostly parsed as `z.string()` and cast. No `.passthrough()` and no `.strict()`.

| Schema (line) | Field → domain | Zod | Tolerant? |
|---|---|---|---|
| `cycleCalculationSchema` (54) | calculation_date→calculationDate | string | required |
| | cycle_day→cycleDay | number | required (an empty sentinel is collapsed to null by `isEmptyCalculation`, 107) |
| | phase | string | required |
| | subphase | string.nullable.default(null) | ✓ |
| | estimated_ovulation_day | number | **required** (v6 null ovulation in the legacy calc would throw) |
| | cycle_length_used→cycleLength | number | required |
| | is_fertile_window / is_pms_window / is_period_tomorrow | boolean.default(false) | ✓ |
| | final_probability→fertilityPercent | number.default(0) | ✓ |
| | cycle_variability | string.nullable.default(null) | ✓ |
| | daily_tips | array(unknown)→per-item safeParse | ✓ |
| `dailyCardSchema` (124) | title | string | **required** |
| | subtitle | string.default('') | ✓ |
| | data_status | string | **required** |
| | fertility_level | enum(very_low..very_high)**.catch('low')** | ✓ (but `unknown` becomes 'low', which is wrong under contraception) |
| | fertility_label, badges, primary_action, secondary_actions | defaults | ✓ |
| `forecastSchema` (152) | next_period_start, next_period_end, estimated_ovulation_date, fertile_window_start, fertile_window_end | **string (non-null)** | ✗ **null throws the whole view** |
| | source / confidence / confidence_reasons | defaults | ✓ |
| `profileValuesSchema` / `calculatedValuesSchema` / `effectiveValuesSchema` (181-213) | cycle_length, period_duration, based_on_cycles, source, period_duration_source | nullable/default | ✓ inside, but the **blocks themselves are required** in `cycleViewSchema` (299-301) |
| `dataQualityDetailsSchema` (215) | confidence, confidence_reasons, regularity_status (default 'not_enough_data'), is_irregular_possible, missing_period_end | defaults | ✓ |
| `anchorsSchema` (234) | current_period_start, current_period_start_source, current_period_end, current_period_end_source, current_period_end_is_confirmed, predicted_next_period_start, estimated_ovulation_date | nullable/default | ✓ — **no `ovulation_source`** |
| `engineMetricsSchema` (257) | effective_cycle_length, effective_period_length | **number (required)** | ✗ |
| | cycle_variability | number.nullable.default(null) | ✓ — **no `cycle_regularity`** |
| `cycleViewSchema` (276) | date | string | required |
| | cycle_day, phase, subphase, main_phase, fertility_level, data_status, days_to_ovulation, days_to_period, days_late | nullable.default(null) | ✓ (main_phase/fertility_level are cast, so `unknown` passes through) |
| | anchors, metrics, data_quality_details, predictions, daily_card | nullable.default(null) | ✓ at block level |
| | confidence, data_quality, resolution_source | string.nullable | ✓ |
| | confidence_reasons, warnings | array.default([]) | ✓ |
| | is_predicted, requires_user_input | boolean.default(false) | ✓ |
| | — | — | **missing: `future_cycles`, `clinical_flags`, `suggest_pregnancy_test`** |
| `cycleCalculationEnvelopeSchema` (335) | calculation, cycle_view, calculation_status, is_recalculating | defaults | ✓ |
| `cycleMonthEnvelopeSchema` (364) | calculations[], month_summary{fertile_days, period_days, pms_days} | defaults | ✓ |
| `cycleStatusSchema` (381) | status, status_label, is_processing, version | defaults | ✓ |
| `api/overview.ts` `overviewSchema` (26) | logging{streak_days, logged_today}, next_period, pms_window, cycle_length{median, source, based_on_cycles, spread_days, **regularity: string**} | all required inside | the query errors on Laravel and home falls back |
| `api/sections.ts` my_cycles / cycle_summary (49, 93) | strict inner shapes (`normal_range`, `normal_ranges`, status enum) | required | **hooks are exported but unused** (no consumer) |

Types live in `entities/cycle/model/types.ts`: `MainPhase` (133, includes `unknown`), `CycleFertilityLevel` (142, includes `unknown`), `ResolutionSource` (148), `CycleAnchors` (161), `CycleEngineMetrics` (172), `CycleView` (244).

## 3. Consumers

| File:line | Source | Fields read |
|---|---|---|
| screens/home/ui/HomePage.tsx:601-639 | useCycleToday | `calculation`, `cycleView` → `deriveCycleSchedule` |
| HomePage.tsx:187-200 (`useDayMarks`) | useCycleMonth ×3 | calc[] → `cycleDayMarkerAt` (week strip and ring dots) |
| HomePage.tsx:665-694 | useCycleOverview | nextPeriod, pmsWindow, cycleLength{median, spread, regularity, basedOnCycles}, streakDays; falls back to schedule |
| HomePage.tsx:706-731 | useCycleForDate (selected day) | calc→`deriveCyclePredictions`; view.mainPhase, daysToPeriod, cycleDay, effectiveValues.cycleLength/periodDuration, dailyCard.subtitle, fertilityLevel |
| HomePage.tsx:778 / model/cycle-data.ts:11 | — | `needsPeriodData` = schedule === null |
| HomePage.tsx:827-838 / model/hero-state.ts:41 | view | daysLate, daysToPeriod, anchors.currentPeriodEndIsConfirmed, cycleDay, periodLength |
| HomePage.tsx:846-893 | view | daysToOvulation (TTC ring), fertility label |
| HomePage.tsx:928-933 | view.mainPhase==='fertile' + fertility today | LH tip |
| HomePage.tsx:943-993 | overview / view.effectiveValues, calculatedValues, profileValues | cycle-length summary, `CycleValuesCard` sync nudge |
| HomePage.tsx:1079 | hero.endMissing | «پایان پریود ثبت نشده» alert |
| HomePage.tsx:1167-1189 / PredictionsCard.tsx:37,132 | schedule/overview slots | period, PMS, fertile window, ovulation ("estimated"), regularity pill, PMS insight |
| HomePage.tsx:1196 (Recommendations) | calc.dailyTips | tips |
| screens/calendar/ui/CalendarPage.tsx:185, model/useCalcMap.ts:37 | fetchCycleMonth | calc[] (phase, flags) |
| CalendarPage.tsx:207-224 | useCycleToday | cycleView → `deriveCycleSchedule` → `cycleDayMarkerAt`; `hideFertility` only for teen/menopause (136) |
| CalendarPage.tsx:397 | calc | `markerPhase` → day card phase pill |
| widgets/cycle-calendar/model/tones.ts | markers | month/year tones |
| screens/phase-details/ui/PhaseDetailsSheet.tsx:41-70; model/tabs.ts:38-58 | useCycleToday | view.subphase (content), mainPhase, anchors.currentPeriodStart, estimatedOvulationDate, metrics.* → `phaseDayRange` |
| screens/fertility-insights/ui/FertilityInsightsPage.tsx:325-339; model/view.ts:166 | useCycleToday + /fertility/insights | anchors.currentPeriodStart, metrics.effectivePeriodLength → `historyStrip` |
| screens/onboarding-flow/ui/ReadyStep.tsx:47-63 | useCycleToday | schedule.nextPeriodStart, fertileStart/End (no contraception check) |
| screens/checkup-self-exam/ui/SelfExamPage.tsx:58-61 | useCycleToday | calc.cycleDay/cycleLength ?? view.cycleDay / metrics.effectiveCycleLength |
| features/log-day/model/use-heading.ts:33-34 | useCycleForDate | view.cycleDay |
| screens/companion-home/ui/CompanionHomePage.tsx:67; model/phase-bar.ts:20 | companion API (not cycle_view) | cycleDay, cycleLength → textbook split |
| screens/cycle/ui/CyclePage.tsx:38-127 | `/cycle/history` (cycleKeys.all+'history') | regularity verdict regular/irregular/not_enough_data |
| screens/cycle/ui/CycleSymptomsPage.tsx:182; model/bars.ts:72 | symptom-pattern API | `typicalBands` (fertile O−5..**O+1**, PMS last **5** days) |
| screens/analysis-cycle/ui/AnalysisCyclePage.tsx:40-181; model/cycle.ts:14 | analysis reports | `dayLayout` local; FIGO normal ranges from backend; hideFertility only teen/menopause |
| screens/analysis/ui/HubCards.tsx:101-129, AnalysisHub.tsx:75 | analysis summary | hideFertility by variant |
| widgets/fertility-tiles/ui/FertilityChanceCard.tsx | props from home | fertility level/window/ovulation/next period |
| entities/cycle/ui/DailyStatusCard.tsx | — | **unused** (no importer outside the entity) |
| useMyCyclesSection / useCycleSummarySection | — | **unused** |
| widgets/teen-home | teen API (`next_period_week`) | does not read cycle_view |

## 4. Duplicated client math

| Helper | Computes | Divergence from v6 |
|---|---|---|
| `entities/cycle/model/schedule.ts:65 deriveCycleSchedule` | cycleStart (anchor, else back-count from calc), length, periodLength, next start, **ovulation fallback chain** (forecast → anchors → `calc.estimatedOvulationDay` → next−14, l.99-103), fertile window `max(O−5, end+1)..O`, PMS next−3..−1 | Fertile end = O already matches v6. **But it fabricates ovulation when v6 sends null** (contraception, unknown ovulation, days_late>14). It ignores `ovulation_source`. It uses `metrics.effective_*` then `effectiveValues`. |
| `schedule.ts:141 cycleScheduleFor` | rolls the schedule ±N×cycleLength | Uses a fixed length shift. v6 `future_cycles[k]` gives per-cycle period_start/ovulation, which can differ (e.g. after a skipped cycle; spec §2.39-8). Also rolls past a late period into a "next cycle" while the engine says `period_expected` / late. |
| `schedule.ts:185 scheduleDayMarker`, `:232 cycleDayMarkerAt` | calendar/ring/week marker: period from legacy calc, window/ovulation/PMS from schedule for current+future | Paints fertile/ovulation even under contraception. Future cycles should come from `future_cycles`. History still uses the legacy `/cycle/month` calc. |
| `schedule.ts:168 daysUntilNextPeriod`, `:173 cycleProgressPercent` | countdown and progress | Should come from `days_to_period` (home already prefers it, HomePage.tsx:724). |
| `entities/cycle/model/predictions.ts:143 deriveCyclePredictions` | from legacy calc: days until period, ovulation, PMS, fertile; fertilityPercent | Legacy engine. `daysUntilOvulation` is always a number (v6 may null it). Fertile lead is fixed at 5. Used as the fallback phase on home (723, 938) when main_phase is `unknown`/null, so the UI may show a phase the engine refused to give. |
| `predictions.ts:77 calcInFertileWindow`, `:89 calcInPmsWindow`, `:102 calcToPhase`, `:115 cycleDayMarker`, `:130 markerPhase` | re-reads legacy flags by the §19 rule (drops O+1; PMS narrowed to 3 days) | Workaround for the legacy engine. No contraception awareness. |
| `screens/home/model/hero-state.ts:41 heroState` | prompt windows: near = 0..2 days (`NEAR_PERIOD_DAYS=2`), late = any daysLate>0, end prompt any day of an open period, endMissing = cycleDay > periodLength | Spec §2.43: start −3..+14, end from L−1 to the hard cap, endMissing from warnings `*_cap_exceeded`. |
| `screens/phase-details/model/tabs.ts:38 phaseDayRange` | phase day ranges from anchors/metrics, ovulation fallback `cycleLength − 14` | Fabricates ovulation. Ignores `main_phase=unknown` and contraception (shows a fertile range). |
| `screens/analysis-cycle/model/cycle.ts:14 dayLayout` | ovulation = L−14+1, fertile O−5..O | Display-only per past cycle. Off by one vs the engine (O = L−14 in spec §2.35: "30 − 14 = 16"). Should come from the backend. |
| `screens/companion-home/model/phase-bar.ts:20 phaseBar` | textbook split: period 5, O = L−14, fertile O−4..O | 5-day window (O−4..O) vs v6 6-day (O−5..O). Fixed period 5. No contraception. |
| `screens/fertility-insights/model/view.ts:166 historyStrip` | fertile = O−5..O−1 plus ovulation O, PMS last 3 | Consistent with v6. Fine as a display reading of backend rows. |
| `screens/cycle/model/bars.ts:72 typicalBands` | fertile **O−5..O+1**, PMS **last 5 days** | **Conflicts with v6** (O−5..O, PMS 3 days). |
| `screens/home/ui/HomePage.tsx:651 slotFor` | rolls past events forward a cycle | Should use `future_cycles`. |

## 5. Impact matrix per v6 change

| v6 change | What breaks or shows wrong today | Required change |
|---|---|---|
| Fertile window ends on O (O−5..O) | Legacy `/cycle/month` still flags O+1, already patched by `calcInFertileWindow`. `typicalBands` (bars.ts:76) still draws O+1. `analysis-cycle dayLayout` O off by one. `phaseBar` is a 5-day window. | Fix `typicalBands` and `phaseBar`. Get the analysis layout from the backend (or align with O = L−14). Once `/cycle/month` serves v6 per-day `main_phase`, delete the §19 shims. |
| `future_cycles[{k, period_start, ovulation}]` | Ignored (stripped). Calendar, ring and prediction rollover use a fixed `cycleLength` shift (`cycleScheduleFor`, `slotFor`). | Add an optional `future_cycles` array (`.catch([])`). `cycleScheduleFor` / `cycleDayMarkerAt` / `slotFor` should prefer it for k≥2 and fall back to the shift. Never persist it (localStorage included; §2.40). |
| `clinical_flags{frequency, duration, absent}` | Not modelled. Nothing shows clinical frequency/duration/absent. | Optional schema (each `.catch(null)`). Render only as a calm "talk to a doctor" hint on the cycle/analysis page (copy from the backend/admin). Never as "normal 21–45". |
| `suggest_pregnancy_test` | Not modelled. There is no pregnancy-test button anywhere on home. | Optional boolean `.default(false)`. The late banner shows «ثبت تست بارداری» (route to the log sheet test entry / pregnancy-test flow). |
| `metrics.cycle_regularity` (regular / irregular / insufficient_data) | Not parsed. The home pill reads `overview.cycleLength.regularity` with the old enum (PredictionsCard.tsx:25), so a v6 value means the **pill silently hidden**. `/cycle/history` uses regular/irregular/not_enough_data. | Parse `metrics.cycle_regularity.catch(null)`. Map both the old and new enums to one UI union (`regular` / `irregular` / `insufficient`). Add `home.nb.predictions.regularity.{regular,irregular,insufficient_data}` in fa/en. |
| `anchors.ovulation_source` (bbt_confirmed / lh_positive / calendar / unknown) | Not parsed. The PredictionsCard ovulation row always says «تخمینی» (`t('estimated')`). | Optional `.catch('unknown')`. Word the row by source («تأییدشده با دما» / «با تست LH» / «تخمینی»). Hide the row when `unknown` and the date is null. |
| `days_to_ovulation` may be null | Home already guards it (`ttcRing` requires != null, 846). `deriveCyclePredictions` / schedule still compute an ovulation that the engine withheld. | Treat a null engine ovulation as "no window": `deriveCycleSchedule` must not fall back to `calc.estimatedOvulationDay` / `next−14` when `cycle_view` is present (only for a Laravel-only payload). Return `ovulation: null` and an empty window. |
| `main_phase` may be `unknown` with anchors still present | `phaseFromMainPhase('unknown')` returns null, so home falls back to the **legacy calc phase** (723, 938). `phaseDayRange` returns null (OK). The phase sheet title falls back (OK). | When `cycle_view` exists and main_phase is `unknown`, show a neutral "not enough data" phase. Don't fall back to the legacy calc. Countdown/anchors can still render. |
| Contraception (§2.40/§2.42: no fertile window or fertility_level when hormonal) | **Never hidden.** Home PredictionsCard window/ovulation rows, ring/week fertile dots, calendar legend + markers, FertilityChanceCard, phase sheet fertile range, onboarding ReadyStep window, analysis cycle/hub fertile bands. `dailyCardSchema.fertility_level.catch('low')` turns `unknown` into 'low'. | Derive one `fertilitySuppressed` flag from the engine: `warnings ∋ hormonal_contraception_active` or `fertility_level === 'unknown'` with a null ovulation. Optionally also check the life-stage `trackContraception` + method type for Laravel prod. Extend the existing teen/menopause `hideFertility` plumbing to it (home 1172, calendar 136, analysis 40, ReadyStep, phase sheet). Swap the next-period wording for «خون‌ریزی قطع مصرف» per §2.42. Change the `.catch('low')` to `'unknown'`-tolerant. |
| Never show 21–45 as "normal" (§2.40) | `messages/{fa,en}/analysis.json:274` «خارج از بازهٔ قابل‌اتکا (۲۱ تا ۴۵ روز)». That says "reliable", not "normal", but sits next to FIGO "normal" copy (analysis.json:33-34, 258). The backend `home/support.go:18-19` digest uses 21/45. `sections.ts` `normal_ranges` (unused). `home.json:260` lateText says «تأخیر چندروزه معمولاً طبیعی است». | Audit: normal ranges must come only from the backend FIGO/teen `{min,max}` (verify the backend sends 24–38 for adults and the teen range for <18). Reword analysis.json:274 so it doesn't read as a norm. Delete the unused `useCycleSummarySection` / `normal_ranges` or align it. |

## 6. §2.43 gap table (Today screen = `screens/home/ui/HomePage.tsx` CycleHome)

| Element (spec) | Today | Gap | Source |
|---|---|---|---|
| «پریودم شروع شد» + «هنوز نه»: no open period and target ∈ [predicted_next−3, predicted_next+14] | `heroState`: near (days_to_period 0..2) or late (days_late>0, no cap); today only | Window should be −3..+14 (`NEAR_PERIOD_DAYS` 2→3, cap late at 14) | Backend: `predicted_next_period_start`, `days_to_period`, `days_late`, open-period anchors. Client: window check (pure). Optionally the backend emits `today_actions` later. |
| «پریودم تموم شد» + «هنوز ادامه دارد»: end not confirmed and days_since ∈ [L−1, hard_cap] | from day 1 of an open period | Add the lower bound L−1 (`effective_period_length`) and the upper bound hard_cap (backend constant, or derive from the `*_hard_cap_exceeded` warning) | Backend: `current_period_end_is_confirmed`, `cycle_day`, `effective_period_length`, warnings. Client: rule. |
| «ویرایش پریود» otherwise | ✓ (ring edit button, `PeriodDateEditor`) | — | client |
| Banner «پایان پریود ثبت نشده»: warnings ∋ warning_cap_exceeded / hard_cap_exceeded | `hero.endMissing` = cycleDay > periodLength (HomePage.tsx:1079) | Drive from `warnings` | backend warnings |
| Banner «پریود دیر کرده»: predicted_period_overdue (days_late ≥ 8) + test button if suggest_pregnancy_test | ring/phase-card "late" text from days_late>0; no banner, no test button | New banner + `suggest_pregnancy_test` | backend |
| Banner «آخرین پریودت کی بود؟»: requires_user_input with stale_onboarding_anchor / cycle_unresolved_after_expected_period | absent (`warnings` and `requiresUserInput` unread) | New banner opening `PeriodDateEditor` (calendar picker) | backend warnings |
| Max one banner, in table order | n/a (one banner exists) | Pure `pickTodayBanner(view)` selector + test | client |
| Ring text limits (label ≤14, unit ≤10, sub ≤18 chars) | Not enforced. `nb.hero.late` / `ring.daysLeft` / `ring.usually` may exceed | Copy audit fa/en + unit test on message lengths | client (i18n) |
| Prediction card: period next..next+L−1 | ✓ (overview or schedule + periodDays) | Prefer `anchors.predicted_next_period_start` + `metrics.effective_period_length` over the legacy forecast | backend |
| PMS next−3..−1 | ✓ (`PMS_WINDOW_DAYS=3`, overview) | — | backend/client |
| Fertile = display window | ✓ from client schedule | Hide under contraception. Don't invent ovulation. | backend anchors |
| Ovulation with source | always «تخمینی» | Read `ovulation_source` | backend |
| Cycle length + regularity badge | overview median ± spread + old-enum pill | Map v6 `cycle_regularity` | backend |
| «هنوز نه / هنوز ادامه دارد» stored locally for that day | ✓ `model/prompt-dismiss.ts` (localStorage per day, guarded) | — | client |
| Fixed disclaimer «پیش‌بینی روش پیشگیری نیست» | absent on home (only on fertility-insights / BBT) | Add under PredictionsCard (all non-teen modes) | client copy |

The rules are pure selectors (`todayActions(view, today, dismissals)`), so they stay client-side. Every number they read must come from `cycle_view`. Nothing should be re-derived from the legacy `calculation` when `cycle_view` is present.

## 7. Query invalidation

The key families: `cycleKeys.all = ['cycle']` covers today, status, date, month, overview, home-section, history and symptom-pattern. The others are `fertilityKeys.all`, `analysisKeys.all`, `messageKeys.all`, `userKeys.all`, `contraceptionKeys.all`, `lifeStageKeys.all`, `careKeys.all`, `['period-status']`, `['period-history']`, `['cycle-settings']` and `articleKeys`.

| Mutation (file:line) | Invalidates today | Missing for v6 |
|---|---|---|
| Period start/end/range (features/log-period/api/mutations.ts:55-66, 86-117, 283-290) | cycle, message, user, period-history, articles (+ period-status set directly) | **fertilityKeys.all** (window, insights, chance), **analysisKeys.all** (cycle/period reports, regularity), careKeys (before-period reminders) |
| Period edit/delete/reconcile (147-153, 162-274) | same + period-status | same |
| Day log save (features/log-day/api/save.ts:53-58) | log ranges, healthLog, cycle, fertility, message, wellbeing | analysisKeys.all |
| Fertility day log (features/log-fertility-day/api/mutations.ts:51-55) | fertility, cycle, message, healthLog | analysisKeys.all (`/analysis/fertility`). LH/BBT now change `ovulation_source` → cycle is already covered ✓ |
| Profile update (features/edit-profile/api/mutations.ts:59-63) | user.profile, user.current, cycle | messageKeys, fertilityKeys, analysisKeys |
| Cycle settings (screens/cycle-settings/api/settings.ts:22) | cycle, user.profile, notification-settings | messageKeys, fertilityKeys, analysisKeys |
| Contraception save (screens/contraception-setup/ui/ContraceptionSetupPage.tsx:142-145) | lifeStage, care, notification-settings, cycle-settings (+ overview set) | **cycleKeys.all**, fertilityKeys, messageKeys (v6 suppresses fertility from `hormonal_contraception`) |
| Contraception stop (screens/mode/ui/ModePage.tsx:186) / pill log (entities/contraception/api/queries.ts:45) | lifeStage only / overview set | cycleKeys.all, fertilityKeys, messageKeys on stop. Pill log is fine. |
| Mode change (ModePage.tsx:127-130) | `invalidateQueries()` (everything) ✓ | — |
| Recalculate (entities/cycle/api/queries.ts:151) | cycle ✓ | fertility, analysis |

Recommendation: add one shared `invalidateCycleDependents(queryClient)` helper. FSD forbids sibling-feature imports, so it should live in a lower layer or be duplicated as an agreed list. It would cover cycle + fertility + analysis + message. Use it from every mutation above.

## 8. Tests

Existing (relevant):
- `entities/cycle/model/{schedule,predictions,markers,phase,highlights}.test.ts`
- `entities/cycle/api/schema.test.ts` (daily tips only)
- `screens/home/model/{hero-state,cycle-data,ttc-hint}.test.ts`
- `screens/companion-home/model/phase-bar.test.ts`
- `screens/cycle/model/bars.test.ts`
- `screens/analysis-cycle/model/cycle.test.ts`
- `screens/phase-details/model/tabs.test.ts`
- `screens/fertility-insights/model/view.test.ts`
- `screens/calendar/model/months.test.ts`
- `widgets/cycle-calendar/model/tones.test.ts`
- `widgets/fertility-tiles/model/ttc.test.ts`
- `features/log-period/model/selection.test.ts`

Needed:
1. `cycleViewSchema` contract tests:
   - the spec §2.35 example payload (v6) parses with every new field;
   - a Laravel-shaped payload (no v6 fields) parses;
   - null `predictions.*` dates, null `metrics`, `main_phase: 'unknown'`, `fertility_level: 'unknown'`;
   - unknown enum values degrade gracefully.
2. `deriveCycleSchedule`: a v6 null ovulation means no window and no invented ovulation; `future_cycles` is preferred in `cycleScheduleFor` / `cycleDayMarkerAt`.
3. `todayActions` / `pickTodayBanner` selectors: the −3/+14 boundaries, L−1/hard_cap, banner priority, the days_late 7 vs 8 boundary, the pregnancy-test flag.
4. `fertilitySuppressed` + marker tests: no fertile/ovulation under contraception.
5. Regularity enum mapping (old + new).
6. `typicalBands` and `phaseBar` re-recorded to O−5..O and PMS 3.
7. Ring copy length test (fa/en ≤14/10/18).
8. Invalidation unit test: each mutation's onSuccess calls the dependents list (mock QueryClient).

## 9. Proposed frontend tasks (M9)

Verify for every task: `cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test`.

**T-M9-F1 — v6 cycle_view boundary (tolerant zod + types)**
- Add optional `future_cycles` (`array(...).catch([])`), `clinical_flags` (`.nullable().catch(null)`), `suggest_pregnancy_test` (`.default(false)`), `metrics.cycle_regularity` (`.catch(null)`), `anchors.ovulation_source` (`.catch('unknown')`).
- Make the `forecastSchema` dates nullable, and `metrics.effective_*` nullable with defaults. Default the `profile_values` / `calculated_values` / `effective_values` blocks. Change `dailyCard.fertility_level` from `.catch('low')` to accepting `unknown`.
- Contract tests: the §2.35 payload, a Laravel payload, and null/unknown edge cases.
- touches: `entities/cycle/api/schema.ts`, `entities/cycle/model/types.ts`, `entities/cycle/api/schema.test.ts`, `entities/cycle/index.ts`
- deps: none (safe before the backend). The backend v6 output task supplies the fixtures.

**T-M9-F2 — Stop fabricating engine output (schedule = engine first)**
- `deriveCycleSchedule`: when `cycle_view` exists, use the engine's ovulation only; null means `ovulation: null` and no window. The legacy fallback applies only when `cycle_view` is null (Laravel).
- `cycleScheduleFor` / `cycleDayMarkerAt` / home `slotFor`: prefer `future_cycles[k]` and fall back to the length shift.
- Home: `main_phase: 'unknown'` shows a neutral phase, with no legacy-phase fallback while `cycle_view` exists.
- Fix `phaseDayRange` (no L−14 invention) and `typicalBands` (O−5..O, PMS 3). Align `phaseBar` (O−5..O) and `analysis-cycle dayLayout` (O = L−14) with the spec.
- touches: `entities/cycle/model/schedule.ts` (+test), `screens/home/ui/HomePage.tsx`, `screens/home/model/hero-state.ts`, `screens/phase-details/model/tabs.ts`, `screens/cycle/model/bars.ts`, `screens/companion-home/model/phase-bar.ts`, `screens/analysis-cycle/model/cycle.ts` (+tests)
- deps: F1; backend future_cycles / null ovulation.

**T-M9-F3 — Hide fertility under hormonal contraception (§2.40/§2.42)**
- `isFertilitySuppressed(view)` in `entities/cycle/model`: warnings ∋ `hormonal_contraception_active`, or `fertility_level==='unknown'` with a null ovulation.
- Feed it into the existing `hideFertility` paths: home PredictionsCard, ring/week markers, FertilityChanceCard/TTC ring, calendar markers + legend, phase sheet fertile range, onboarding ReadyStep, analysis cycle/hub. Use «خون‌ریزی قطع مصرف» wording instead of the next-period row.
- touches: `entities/cycle/model/*`, `screens/home/**`, `screens/calendar/ui/CalendarPage.tsx`, `screens/phase-details/**`, `screens/onboarding-flow/ui/ReadyStep.tsx`, `screens/analysis-cycle/**`, `screens/analysis/ui/*`, `messages/{fa,en}/home.json`, `messages/{fa,en}/calendar.json`
- deps: F1; backend contraception suppression + warning.

**T-M9-F4 — Today screen §2.43 mapping (prompts + banners + disclaimer)**
- Pure `todayActions(view, todayIso, dismissals)`:
  - start prompt for predicted_next−3..+14 with no open period;
  - end prompt for unconfirmed end with days_since ∈ [L−1, hard_cap];
  - edit otherwise.
- Pure `pickTodayBanner(view)` in priority order:
  1. end missing (warnings `*_cap_exceeded`);
  2. late (`predicted_period_overdue`, days_late ≥ 8) with a «ثبت تست بارداری» button if `suggest_pregnancy_test`;
  3. «آخرین پریودت کی بود؟» (requires_user_input + `stale_onboarding_anchor` / `cycle_unresolved_after_expected_period`), which opens `PeriodDateEditor`.
- Max one banner. Keep the localStorage dismissals.
- Add the fixed disclaimer «پیش‌بینی روش پیشگیری نیست» under PredictionsCard.
- Ring copy audit to ≤14/10/18 chars (fa/en), with a test.
- touches: `screens/home/model/hero-state.ts` (→ today-actions) + tests, `screens/home/ui/HomePage.tsx`, new `screens/home/ui/TodayBanner.tsx`, `messages/{fa,en}/home.json`
- deps: F1; backend warnings / suggest_pregnancy_test / days_late v6 semantics.

**T-M9-F5 — Prediction card v6: ovulation source + regularity badge**
- Ovulation row wording by `anchors.ovulation_source`.
- Regularity pill maps the v6 `metrics.cycle_regularity` (and the old overview enum) to one union. Add fa/en keys.
- Period row from `predicted_next_period_start` + `effective_period_length`.
- touches: `screens/home/ui/PredictionsCard.tsx`, `screens/home/ui/HomePage.tsx`, `entities/cycle/api/overview.ts` (if the overview adopts v6 regularity), `messages/{fa,en}/home.json`
- deps: F1; backend regularity + ovulation_source.

**T-M9-F6 — Calendar on v6 (future cycles + contraception + period_expected)**
- Current/future markers come from `future_cycles`. Late period: no rolled-forward period paint past `predicted_next` while `days_late > 0`.
- History stays on `/cycle/month`. If the backend moves `/cycle/month` to v6 per-day `main_phase`, delete the §19 legacy shims (`calcInFertileWindow` / `calcInPmsWindow` O+1 / 7-day workarounds), keeping a Laravel fallback.
- touches: `screens/calendar/**`, `widgets/cycle-calendar/**`, `entities/cycle/model/{schedule,predictions}.ts` + tests
- deps: F2, F3; backend `/cycle/month` v6 (optional).

**T-M9-F7 — Clinical flags + no-"normal 21–45" copy audit**
- Show `clinical_flags` (frequency / duration / absent ≠ normal) as a calm doctor hint on `/cycle` and `/analysis/cycle` (backend/admin copy). Teen range comes from the backend.
- Reword `analysis.json:274` (21–45 "reliable range"). Verify the backend sends FIGO / teen `{min,max}` and not 21–45. Delete the unused `useCycleSummarySection` / `useMyCyclesSection` / `DailyStatusCard`, or wire them correctly.
- touches: `screens/cycle/**`, `screens/analysis-cycle/**`, `entities/cycle/api/sections.ts`, `entities/cycle/ui/DailyStatusCard.tsx`, `messages/{fa,en}/analysis.json`, `messages/{fa,en}/cycle.json`
- deps: F1; backend clinical_flags.

**T-M9-F8 — Cycle-dependent invalidation**
- One agreed dependents list (cycle + fertility + analysis + message). Apply it to period start/end/edit/delete/reconcile, day-log save, fertility log, profile update, cycle-settings, contraception save/stop, and recalculate.
- Unit tests with a mocked QueryClient.
- touches: `features/log-period/api/mutations.ts`, `features/log-day/api/save.ts`, `features/log-fertility-day/api/mutations.ts`, `features/edit-profile/api/mutations.ts`, `screens/cycle-settings/api/settings.ts`, `screens/contraception-setup/ui/ContraceptionSetupPage.tsx`, `screens/mode/ui/ModePage.tsx`, `entities/cycle/api/queries.ts`
- deps: none.

Order: F1 → (F8 in parallel) → F2 → F3 / F5 → F4 → F6 → F7. F1 and F8 are safe on production (Laravel). All others must keep a Laravel fallback until the T-M2-26 cutover.
