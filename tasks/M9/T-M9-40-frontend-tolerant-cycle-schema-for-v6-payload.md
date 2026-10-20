---
id: T-M9-40
title: Frontend: tolerant cycle schema for v6 payloads
milestone: M9
type: frontend
status: todo
depends_on: []
parallel_group: M9-A
touches: [frontend/src/entities/cycle]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test
---

# T-M9-40 — Frontend: tolerant cycle schema for v6 payloads
**Model:** sonnet; the main session reviews the schema diff (a parse error blanks the whole Home).

## Why
The zod schemas require fields v6 may send as null (`predictions.*` dates, `metrics.effective_*`, the value layers,
the per-day `calculation.estimated_ovulation_day`); `.parse` throws, the query errors and Home shows its error card —
most likely under hormonal contraception (gap 02c "Main risks" 1). Every new v6 field is silently dropped, and the
schedule prefers the calendar-only `predictions` block over the engine's anchors (risk 3). This task is safe on
production (Laravel payloads keep working) and unblocks every other frontend task.

## Scope
1. `entities/cycle/api/schema.ts`: nullable `predictions.*` dates, `metrics.effective_*`, value layers defaulted,
   nullable per-day `estimated_ovulation_day`; month days parsed one by one (a bad day is dropped, not the month); new
   optional fields: `future_cycles` (`.catch([])`), `clinical_flags` (`.nullable().catch(null)`),
   `suggest_pregnancy_test` (`.default(false)`), `metrics.cycle_regularity` (`.catch(null)`),
   `anchors.ovulation_source` (`.catch('unknown')`), `today_ui` and `limits` (`.nullable().catch(null)`); daily card
   `fertility_level` accepts `peak` and `unknown`; unknown enum values degrade to `unknown`, never throw.
2. `model/types.ts`: the matching types (v6 fields optional so Laravel payloads type-check).
3. `model/schedule.ts` `deriveCycleSchedule`: anchors first; the `predictions` block only when `cycle_view` is absent
   (old backend); a null ovulation means no window and no invented ovulation.
4. `isFertilitySuppressed(view)`: `limits.fertility_suppressed`, else warnings containing
   `hormonal_contraception_active` or `postpartum_cycles_not_returned`. Export it from `index.ts`.

## Tests
- Parse: the §2.35 example payload (hand-copied from `docs/engine-v6/SPEC.md`, plus `today_ui`/`limits`); a Laravel
  payload (existing fixture); a pill payload with null ovulation/window/predictions; a payload without value layers; a
  month with one malformed day; an unknown `main_phase` string.
- `deriveCycleSchedule`: anchors win when anchors and forecast disagree; null ovulation → no window; Laravel payload →
  today's behaviour (existing tests stay green).
- `isFertilitySuppressed` truth table.

## Out of scope
- Screens (T-M9-42…47).

## Acceptance
- Verify passes; no screen changes behaviour on a Laravel payload.
