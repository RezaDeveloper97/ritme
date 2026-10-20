---
id: T-M9-18
title: Engine v6 quality layer: regularity, clinical flags, data quality, confidence
milestone: M9
type: backend
status: todo
depends_on: [T-M9-17]
parallel_group: M9-B
touches: [backend-go/internal/cycle/engine]
skills: []
verify: cd backend-go && go vet ./internal/cycle/engine/... && go test -race ./internal/cycle/engine/...
---

# T-M9-18 — Engine v6 quality layer: regularity, clinical flags, data quality, confidence
**Model:** opus (health logic).

## Why
`cycle_regularity` and `clinical_flags` do not exist in the engine (a second FIGO implementation lives in the
analysis tab); confidence misses the predicted-reference, hormonal, postpartum, age ≥ 45 and BBT rules; reasons cover
12 of 14 (§2.27–§2.29, §2.32–§2.33, gap 01).

## Scope
1. `cycle_regularity` (§2.27, E-08, E-09): real complete cycles whose start lies in the 12 calendar months before
   `ref = min(target, today)`; at least 3; skipped cycles excluded; outliers included; range = max − min; threshold by
   age at `ref`: 26–41 → `regularity.adult`, 18–25 and 42–45 → `regularity.young`/`regularity.late`, < 18 → young band,
   > 45 → late band; no birth date → adult band. Result regular | irregular | insufficient_data.
2. `clinical_flags` (E-06, E-07, E-08): frequency from the median of the last ≤ 3 real complete cycle lengths
   (outliers included, skipped excluded), fallback declared, else null; FIGO bands, teen (< 18 or `TeenBandOnly`) uses
   `cycle.teen.frequency`; duration = prolonged when the median of the last ≤ 3 closed real periods > FIGO max, null
   without a closed real period; absent per E-07 at `ref` (`long_absence_no_period` when true).
3. `data_quality` (§2.28, order insufficient → poor → good → partial) with E-02 (contradiction → poor), E-20 (manual
   lengths never good) and E-23 (good also needs history-sourced period length).
4. `confidence` (§2.29, order unknown → low → high → medium) including every low condition (predicted_reference
   start, hormonal, postpartum with < 2 real periods after delivery, `main_phase = unknown`, late > 7, poor, past warning
   cap, < 2 valid cycles, variability > 5 **and** source ≠ bbt_confirmed (E-19), default/onboarding resolution), the
   high conditions, the medium caps (age ≥ 45 → `perimenopause_age_range`; future k ≥ 2 is applied in T-M9-20).
5. All 14 confidence reasons and the remaining warnings (`low_history`, `high_cycle_variability`,
   `short/long_cycle_outlier_detected` — a skipped cycle never raises them, `perimenopause_age_range`,
   `long_absence_no_period`); `requires_user_input` = the six warnings of §2.33 (incl. `contradictory_anchor_data`).
6. Age at the target date from `BirthDate` (E-27); none → no age rules.

## Tests
- Regularity: age 30 [27,29,33] → regular; [26,30,34] → irregular; age 22 [26,30,34] → regular; 2 cycles →
  insufficient_data; 3 cycles with one skipped → insufficient_data; cycles older than 12 months ignored; outlier
  [28,28,50] → irregular; age 16 / age 46 bands.
- Flags: effective 23 → frequent; 39 → infrequent; teen 23 → normal; real 18-day cycles → frequent (E-06); last periods
  [9,9,9] → prolonged; none closed → null; last start 91 days ago in cycle mode → absent + warning; same in postpartum →
  false; no real period → null.
- data_quality: the §2.35 example → good; onboarding only → partial; 1 real cycle → partial; open past day 8 → poor;
  late 15 → poor; only outliers → poor; contradictions → poor; no anchor → insufficient; manual lengths → ≤ partial.
- Confidence: the §2.35 example → high, reasons []; age 46 with a perfect history → medium + `perimenopause_age_range`;
  variability 7 with bbt_confirmed → not low from variability (E-19); variability 7 without → low +
  `high_cycle_variability`; late 8 → low + `period_overdue`.
- Each of the 14 reasons is triggered by one fixture and asserted alone where possible.
- `requires_user_input` true exactly for the six warnings.

## Out of scope
- Suppression effects (T-M9-19), future-cycle confidence caps (T-M9-20).

## Acceptance
- Tests green; PROGRESS.md notes every case where v6 and the analysis-tab FIGO code disagree (input for T-M9-35).
