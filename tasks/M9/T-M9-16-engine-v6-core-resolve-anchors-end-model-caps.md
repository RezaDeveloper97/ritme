---
id: T-M9-16
title: Engine v6 core resolve: anchors, end model, caps, phases (§2.11–§2.26)
milestone: M9
type: backend
status: todo
depends_on: [T-M9-15]
parallel_group: M9-B
touches: [backend-go/internal/cycle/engine]
skills: []
verify: cd backend-go && go vet ./internal/cycle/engine/... && go test -race ./internal/cycle/engine/...
---

# T-M9-16 — Engine v6 core resolve: anchors, end model, caps, phases (§2.11–§2.26)
**Model:** opus (health logic).

## Why
The core of the spec: which phase, subphase and fertility level a date has, from which anchor, with which warnings.
v1.1 gets most tiers right but differs in the luteal priority order, emits `prediction_based_output` on every
prediction day (E-01), and scans all histories per call (gap 01 §2.20–§2.26, §4).

## Scope
1. `(*State).Resolve(target, today Day) Status` for the **current-cycle and history** cases (future cycles, skips and
   the stale anchor are T-M9-20; suppression is T-M9-19).
2. Anchor: latest real start ≤ target by binary search (O(log H)); with no real start, the declared onboarding start
   (`onboarding_declared`, E-10 override → resolution `onboarding_based`, or `default_based` when both lengths are
   defaults); with neither → `unknown` / insufficient anchor (`insufficient_anchor_data`).
3. End model §2.11 and caps §2.12 from `Params` (soft = EPL, warning = max(EPL + extra, min), hard).
4. Resolve priority §2.20; menstrual tiers §2.21 with cumulative cap warnings (E-12); period_expected §2.22 with the
   three tiers, > max_late → `unknown`, `resolution_source = prediction` (E-11), anchors and `days_late` still
   returned, `days_to_period = 0`.
5. Calendar ovulation (`predicted_next − 14`; T-M9-17 adds BBT/LH) and `ovulation_source = calendar`; biological
   window O−5..O and display window `max(O−5, period_end+1)..O` (§2.18–§2.19), empty window allowed.
6. Follicular split §2.23, fertile subphases §2.24, luteal §2.25 with the **spec priority**
   pms_possible > late_luteal > post_ovulation > early/mid and the ceil/floor assignable split; fertility_level §2.26.
7. `days_to_period`, `days_late`, `days_to_ovulation` (§2.34 null rules; suppression hook stays false until T-M9-19),
   `resolution_source` §2.30, `is_predicted` §2.31, `cycle_day` §2.5.
8. `suggest_pregnancy_test` (§2.22): mode ∈ {cycle, ttc}, not hormonal, `days_late ≥ ttc_days_late` (ttc) or
   `≥ cycle_days_late` (cycle) — values from `Params`.
9. Warnings in this task: the period-end family, `predicted_period_overdue`,
   `cycle_unresolved_after_expected_period`, `insufficient_anchor_data`, `onboarding_data_used`,
   `default_values_used`. `prediction_based_output` only per E-01 (not on ordinary prediction days).
10. Completed cycles in history use the **retrospective layout** (E-22): inside `[start_i, start_{i+1})` the actual next
    start replaces the prediction, ovulation = `start_{i+1} − 14` (BBT/LH in T-M9-17), no `period_expected` days.
11. No allocation in `Resolve` beyond the returned value (bitsets, `Opt`, no slices).

## Tests
- §2.36–§2.38 tables verbatim: 21/28/35-day cycles with a confirmed 5-day period — every day 1…(L+1) asserts
  main_phase, subphase, fertility_level (one table-driven case per day, names cite §2.36).
- Open period with EPL 5 on days 5/6/8/9/12/13/20: tiers, warnings cumulative, day 13+ resolves onward
  (follicular/fertile/luteal) with `period_end_missing_hard_cap_exceeded`.
- `days_late` 0/7/8/14/15: tiers; 15 → `unknown`, `prediction`, `days_to_period = 0`, `days_to_ovulation` null, anchors
  present; 14 → `days_to_ovulation = −28`.
- Empty and short windows: 21-day cycle with a confirmed 8-day period → day 8 menstrual, day 9 post_ovulation; 21 with
  7-day → day 8 ovulation_likely.
- O+1 is never fertile; a menstrual day inside the biological window stays menstrual.
- Luteal priority with an injected ovulation (internal test hook) so O+1 = next − 3 → `pms_possible`; a 2-day assignable
  range → 1 early + 1 mid; an empty range → mid_luteal.
- Onboarding only (declared 07-01, 28/5): every phase `onboarding_based`, period_expected days 0–14 included; declared
  date without lengths → `default_based`; nothing → unknown/insufficient.
- `suggest_pregnancy_test` matrix: ttc + 1 → true; cycle + 7 → false; cycle + 8 → true; cycle + 8 hormonal → false;
  pregnancy mode → false; late 0 → false.
- Retrospective: cycles 28 then 35 (completed) → ovulation of the 35-day cycle on its day 22, no period_expected days
  inside it; the current cycle still predicts from the effective length.
- Gregorian arithmetic across Jalali boundaries: a cycle spanning 2025-03-20/21 (Esfand 1403 leap → Farvardin 1404)
  and one spanning 2026-09-22/23 (Shahrivar 31 → Mehr) keep `cycle_day` continuous.
- `testing.AllocsPerRun` on `Resolve` = 0.

## Budgets
- `BenchmarkResolve` (24 periods) ≤ 3 µs and 0 allocs on this machine (locked in T-M9-22).

## Out of scope
- BBT/LH, quality layer, limits, future cycles, stale anchor, skips, JSON.

## Acceptance
- Tests green; PROGRESS.md lists every deliberate difference from v1.1 found while porting (for T-M9-32's shadow
  classification).
