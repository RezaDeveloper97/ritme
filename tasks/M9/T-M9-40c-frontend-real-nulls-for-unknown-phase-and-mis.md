---
id: T-M9-40c
title: Frontend: real nulls for unknown phase and missing fertility number (drop NaN/'' sentinels)
milestone: M9
type: frontend
status: todo
depends_on: [T-M9-40b,T-M9-46]
parallel_group: M9-C
touches: [frontend/src/entities/cycle/model,frontend/src/entities/cycle/api/schema.ts,frontend/src/entities/cycle/api/schema.test.ts,frontend/src/screens/companion-home]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M9-40c — Frontend: real nulls for unknown phase and missing fertility number (drop NaN/'' sentinels)

## Why
T-M9-40b made v6 `calculation.phase: null` and `final_probability: null` parse, but `CycleCalculation.phase` is
`string` and `fertilityPercent` is `number` (`entities/cycle/model/types.ts`), so schema.ts maps them to `''` and
`NaN` sentinels. `predictions.ts` then clamps NaN to NaN, `normalizePhase('')` falls back to luteal, and
`CyclePredictions.phase` / `markerPhase` show 'follicular' as a placeholder for an unknown day (T-M9-46).
The companion bar (`CompanionHomePage` → `phaseBar(day, length)`) still uses the L−14 fallback.

## Scope
- `types.ts`: `phase: string | null`, `fertilityPercent: number | null`, `CyclePredictions.phase` nullable.
- schema.ts: real nulls instead of the sentinels; tests updated.
- predictions.ts: pass null through (no 0%, no NaN); unknown day has no phase/marker instead of a placeholder.
- Fix every consumer the type change breaks (typecheck); an unknown day renders a neutral state.
- Companion home: pass engine options to `phaseBar` when the shared view has them; if `SharedCycleView` lacks the
  fields, note it and keep the fallback.

## Out of scope
- Backend changes.

## Acceptance
- No `NaN` or `''` sentinel left for these fields (grep); an overdue/unknown day is never shown as luteal or
  follicular; verify green.
