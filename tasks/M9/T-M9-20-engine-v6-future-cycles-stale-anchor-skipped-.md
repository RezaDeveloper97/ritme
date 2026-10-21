---
id: T-M9-20
title: Engine v6 future cycles, stale anchor, skipped cycles and Range
milestone: M9
type: backend
status: done
depends_on: [T-M9-19]
parallel_group: M9-B
touches: [backend-go/internal/cycle/engine]
skills: []
verify: cd backend-go && go vet ./internal/cycle/engine/... && go test -race ./internal/cycle/engine/...
---

# T-M9-20 — Engine v6 future cycles, stale anchor, skipped cycles and Range
**Model:** opus (health logic).

## Why
Future dates in v1.1 roll the anchor forward and show "period day 5 tomorrow" while today says "3 days late"; there is
no `future_cycles`, no stale-anchor rule (a 90-day-old onboarding date resolves `unknown`, 62 days late) and no
skipped cycle (gap 01 §2.16, §2.39, §2.41). The calendar needs a fast range API instead of 31 resolves.

## Scope
1. `(*State).FutureCycles(today) []FutureCycle`, k = 1..6, shape per E-05: `{k, period_start, period_end,
   fertile_start, fertile_end, ovulation, ovulation_source, confidence}`; k = 1 uses the current cycle's ovulation and
   source; k ≥ 2 use `period_start[k] − 14` (`calendar`); under suppression (T-M9-19) fertile/ovulation are null and in
   pregnancy/postpartum-not-returned the list is empty. Late rebase (E-04): while no new start is logged and
   `today ≥ predicted_next`, `period_start[1] = today + 1`; when `days_late > max_late` the list is empty.
2. Future targets (E-03): `period_expected` applies only to targets ≤ today; a target after today and ≥ the current
   cycle's next start resolves inside `future_cycles[k]` (start source `predicted_reference`,
   `prediction_based_output`, confidence `min(today's, k == 1 ? high : medium)`, `FutureK = k`); targets after
   `future_cycles[6].period_end` → `unknown` (E-28).
3. Stale onboarding anchor (§2.41): no real period, declared anchor, `target − anchor > ECL + stale_anchor.extra_days`
   → `n = floor(diff / ECL)`, start `anchor + n·ECL` (`predicted_reference`), confidence low,
   `stale_onboarding_anchor`, `requires_user_input` true.
4. Skipped cycles (E-15, §2.39.8): a skip whose `expected_start` matches a predicted start creates a virtual anchor
   there (`predicted_reference`, low), `cycle_day` restarts, `predicted_next = skip + ECL`; `skipped_cycle_recorded`
   until the next real start; consecutive skips chain; the gap across the skip is excluded from medians, variability
   and regularity (Prepare already marks it). A skip that no longer matches any predicted start (data changed) is
   ignored and reported in `State.Diagnostics` (no warning).
5. `(*State).Range(from, to, today Day) []Status` computing each cycle's layout once and filling days; equals
   `Resolve` for every day (property-tested). `from`/`to` may cross today and cycle boundaries; max span 400 days.

## Tests
- Chain: start 07-01, L = 29 → k=1 07-30, k=2 08-28 (ovulation 08-14), k=6 12-22 (07-30 + 5×29; was a 12-24 typo); k=1 confidence = today's, k ≥ 2
  ≤ medium; `period_end = start + EPL − 1`; fertile = O−5..O.
- Late rebase: anchor 01-01, L = 28, today 02-01 (late 3) → k=1 starts 02-02; target 02-02 resolves menstrual (predicted,
  `FutureK = 1`), never "day 5"; days 01-29…02-01 are period_expected; late 15 → empty list, future targets unknown.
- Beyond k = 6 → unknown, no fertile data.
- Stale anchor: diff 42 (= 28 + 14) → no stale (period_expected, late 14); 43 → stale, n = 1; 90 → start 09-23,
  cycle_day 7 on 09-29, `mid_follicular`, low, warning, requires input; any real period → no stale.
- Skip: start 07-01, L = 28, skip 07-29 → next 08-26, 08-10 is not late (cycle_day 13 from 07-29, `predicted_reference`,
  low, `skipped_cycle_recorded`); a real start 08-24 clears the warning and the 07-01→08-24 gap is excluded from medians;
  two consecutive skips; deleting the skip (input without it) restores the late output.
- Suppression: pill → k items without fertile/ovulation; postpartum not returned → [].
- `Range == map(Resolve)` over 2 000 seeded random histories × a 400-day window crossing today.
- §6.1: "edit 3 months back" — changing an old period's start changes medians/regularity of the current status (pure
  recompute; cache side is T-M9-30/31).

## Budgets
- `BenchmarkRange31` (24 periods) ≤ 150 µs, ≤ 40 allocs; `BenchmarkFutureCycles` ≤ 5 µs.

## Out of scope
- The skip endpoint (T-M9-29), storage (T-M9-24), calendar payload (T-M9-33).

## Acceptance
- Tests green, including the property test; PROGRESS.md records the E-03/E-04/E-15 behaviours with one example each.
