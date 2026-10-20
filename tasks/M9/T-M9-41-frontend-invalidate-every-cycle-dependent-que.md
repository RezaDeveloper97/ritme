---
id: T-M9-41
title: Frontend: invalidate every cycle-dependent query after cycle writes
milestone: M9
type: frontend
status: todo
depends_on: []
parallel_group: M9-A
touches: [frontend/src/shared/api/invalidation.ts, frontend/src/shared/api/invalidation.test.ts, frontend/src/shared/api/index.ts, frontend/src/features/log-period/api/mutations.ts, frontend/src/features/log-day/api/save.ts, frontend/src/features/log-fertility-day/api/mutations.ts, frontend/src/features/edit-profile/api/mutations.ts, frontend/src/screens/cycle-settings/api/settings.ts, frontend/src/screens/contraception-setup/ui/ContraceptionSetupPage.tsx, frontend/src/screens/mode/ui/ModePage.tsx, frontend/src/entities/contraception/api/queries.ts]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test
---

# T-M9-41 — Frontend: invalidate every cycle-dependent query after cycle writes
**Model:** sonnet.

## Why
After a period or log write the app refreshes cycle data but not fertility (window, insights, chance), analysis
(regularity) or reminders; a contraception save does not refresh the cycle at all — yet in v6 contraception suppresses
fertility (gap 02b §7). Users would see stale predictions until a reload.

## Scope
1. `shared/api/invalidation.ts`: `CYCLE_DEPENDENT_ROOTS` (`['cycle']`, `['fertility']`, `['analysis']`,
   `['messages']`, `['period-status']`, `['period-history']`, plus the care/reminder root that shows before-period
   reminders — check the key factories) and `invalidateCycleDependents(queryClient)`. FSD: `shared` cannot import
   entity key factories, so a test asserts the roots equal the factories' `all` keys.
2. Use it in: period start/end/range/edit/delete/reconcile, day-log save, fertility day log, profile update, cycle
   settings, contraception save and stop, recalculate (`entities/cycle/api/queries.ts:151` — if that file is busy with
   T-M9-40, do it after; note in PROGRESS.md). Mode change keeps its full invalidation.

## Tests
- Each mutation's `onSuccess` calls the helper (mocked QueryClient), and the helper invalidates every root.
- The roots test against the entity key factories.

## Out of scope
- Backend.

## Acceptance
- Verify passes.
