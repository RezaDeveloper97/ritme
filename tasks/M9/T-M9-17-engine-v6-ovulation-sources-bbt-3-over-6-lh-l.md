---
id: T-M9-17
title: Engine v6 ovulation sources: BBT 3-over-6, LH, luteal median, conflict
milestone: M9
type: backend
status: todo
depends_on: [T-M9-16]
parallel_group: M9-B
touches: [backend-go/internal/cycle/engine]
skills: []
verify: cd backend-go && go vet ./internal/cycle/engine/... && go test -race ./internal/cycle/engine/...
---

# T-M9-17 — Engine v6 ovulation sources: BBT 3-over-6, LH, luteal median, conflict
**Model:** opus (health logic).

## Why
Ovulation is always `next − 14` in the resolver; BBT/LH priority exists only in `fertility/insights.go` for finished
cycles, `bbt.go` uses a different rise rule (first reading ≥ +0.20 °C, 9 readings) and nothing implements
`lh_bbt_conflict` or the BBT-anchored next period (§2.16–§2.17, gap 01 §2.17).

## Scope
1. `internal/cycle/engine/bbt`: pure detector over one cycle's readings (integer hundredths °C), rules per E-16:
   ≥ `min_readings` readings in the cycle; baseline = max of the previous 6 readings; 3 consecutive *readings* above it
   (a missing day does not break the run); strict mode: the third reading ≥ baseline + `rise_centi`; ovulation = the day
   before the first high reading; confirms only after the third reading exists. Returns the ovulation day and the
   index of the confirming reading.
2. LH: first `positive` test in the cycle → ovulation = that day + 1 (`faint`/negative ignored; the loader maps them).
3. `Prepare` part 2 (per real cycle, today-independent): BBT and LH ovulation per cycle, source priority BBT > LH >
   calendar, conflict when |O_lh − O_bbt| > `lh_bbt_conflict.days` → BBT + `lh_bbt_conflict` on that cycle.
4. `effective_luteal_length` = §2.9 median of the last ≤ 3 BBT-confirmed luteal lengths (next real start − BBT
   ovulation) within `[luteal.min, luteal.max]`, else `luteal.default` (E-16).
5. Current cycle: when its source is bbt_confirmed or lh_positive, `predicted_next = O + effective_luteal_length`
   (§2.16), and every downstream window/phase uses that O and next. Completed cycles (E-22) use detected O when present.
6. `anchors.ovulation_source` and `estimated_ovulation_date` filled; `lh_bbt_conflict` warning on targets of that cycle.
7. A reading or test dated on a menstrual day still counts (the spec does not exclude it).

## Tests
Cycle start 07-01, 28-day calendar default:
- BBT: 6 baseline readings ≤ 36.50 on days 8–13, then 36.62/36.65/36.72 on days 14–16, 10+ readings → bbt_confirmed,
  O = day 13; only 9 readings in the cycle → not confirmed → LH or calendar.
- Strict vs standard: 36.51/36.52/36.60 over a 36.50 maximum → standard confirms, strict does not; 36.55/36.60/36.72
  over 36.50 → both confirm (36.72 ≥ 36.70).
- A missing day inside the high run still confirms; a high run of only 2 readings does not.
- LH positive day 12 → O = day 13, lh_positive; a `faint` result alone → calendar.
- Conflict: LH O = 11 and BBT O = 14 → BBT wins + `lh_bbt_conflict`; LH O = 12 and BBT O = 14 → BBT wins, no warning.
- Luteal: history [12,13] → 13; [9,17] → both invalid → 14; three cycles [11,12,15] → 12.
- BBT O 07-14 + luteal 13 → next 07-27 (period_expected from 07-27); LH O 07-14 (no BBT) + default 14 → 07-28.
- Luteal 10 with BBT → assignable range O+2..O+3 → 1 early + 1 mid; pms days follow the moved next.
- A completed past cycle with BBT O → its retrospective layout uses that O.
- Fuzz target `FuzzBBTDetect` (random readings) never panics and never confirms with < `min_readings`.

## Out of scope
- Switching `internal/fertility` to this detector (T-M9-35), quality/confidence effects of bbt_confirmed (T-M9-18).

## Acceptance
- Tests green; package `engine/bbt` coverage ≥ 95 %.
