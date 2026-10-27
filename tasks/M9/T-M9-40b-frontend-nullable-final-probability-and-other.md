---
id: T-M9-40b
title: Frontend: nullable final_probability (and other null v6 fields) in cycle/today schema
milestone: M9
type: frontend
status: done
depends_on: [T-M9-40]
parallel_group: M9-C
touches: [frontend/src/entities/cycle/api/schema.ts,frontend/src/entities/cycle/api/schema.test.ts]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test
---

# T-M9-40b — Frontend: nullable final_probability (and other null v6 fields) in cycle/today schema

## Why
Found during T-M9-42c screenshots: Go `/cycle/today` under `CYCLE_ENGINE=v6` sends `calculation.final_probability: null`,
but `entities/cycle/api/schema.ts` has `z.number().default(0)` (default only covers `undefined`). The zod parse fails
and Home shows «اطلاعات امروز بارگذاری نشد». A 20-days-late persona (persona 11) fails the same way, cause not yet
found. T-M9-40 made the schema tolerant but missed these.

## Scope
- Make `final_probability` nullable (`z.number().nullable()` → map null to "no fertility number", not 0%).
- Reproduce persona 11 (late 15+ / ask_last_period) against local Go v6 and fix every other null/shape the schema
  rejects; add a schema test per field with a captured v6 payload.

## Out of scope
- Backend changes.

## Acceptance
- A v6 `/cycle/today` payload with `final_probability: null` and the late-15+ payload both parse; Home renders.
- No "0%" fertility shown when the server sent null.
- verify green.
