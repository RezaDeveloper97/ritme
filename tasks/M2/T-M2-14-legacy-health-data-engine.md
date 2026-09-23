---
id: T-M2-14
title: Legacy HealthDataEngine library (calculation, probability, text flags, tips)
milestone: M2
type: backend
status: todo
depends_on: [T-M2-05, T-M2-13]
parallel_group: M2-F
touches: [backend-go/internal/cycle/legacy, backend-go/internal/cycle/recommendation]
skills: []
verify: cd backend-go && go test -count=1 ./internal/cycle/legacy/... ./internal/cycle/recommendation/...
---

# T-M2-14 — Legacy HealthDataEngine library

## Why
`HealthDataEngine::calculateForDate` (1,132 lines, **no unit tests**) feeds `calculation` in `/cycle/today|date`, the
whole month view, home and the message system. It disagrees with the v1.1 resolver by design; both must be ported
exactly. Read domain-inventory §4.1 "Legacy engine" and the PHP source end to end before writing code.

## Scope
1. `cycle/recommendation` = `RecommendationRepository` + `Recommendation::appliesTo/toTip` (DB reader behind an
   interface; request-scoped memo).
2. `cycle/legacy`: anchor search over **all** history rows + roll-forward + LMP extrapolation, ovulation day
   `max(ECL−13,7)`, bleed length guard, phase/subphase (12 states), fertile window, PMS, period-tomorrow window,
   luteal spotting, probability model (base × age × cycleScore × symptomScore, caps, `round(x,4)`,
   `final_probability = round(p*100,2)`), bilingual `text_flags` with PHP float interpolation, `daily_tips`
   (DB recommendations if the table has **any** row, else the hardcoded tips at `HealthDataEngine.php:780-965` —
   embed them), calendar mode `CALENDAR_FIELDS`, and `localizeCalculation` (string when the locale key exists, whole
   `{en,fa}` object otherwise).
3. Implement the interface `cycle/view` expects from the legacy engine (T-M2-13).
4. Before porting, write characterization tests from the `cycle-sweep` goldens (T-M2-05): for each persona/day,
   the `calculation` object (and month `calculations[]`) is the expected output.

## Out of scope
HTTP layer and cache (T-M2-15). Messages (T-M2-17).

## Acceptance
- Characterization tests over the full sweep (all personas × 60 days × fa/en + 3 months full/calendar) green with
  **zero** allow-list entries.
- Probability/rounding edge cases have explicit unit tests with values computed by `php -r` against the PHP class.
- No DB/Fiber imports except through the recommendation repository interface.
