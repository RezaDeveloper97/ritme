---
id: T-M9-47
title: «پریود نداشتم» skip flow and the last-period prompt end to end
milestone: M9
type: frontend
status: todo
depends_on: [T-M9-29, T-M9-44]
parallel_group: M9-C
touches: [frontend/src/features/skip-period, frontend/src/screens/home, frontend/src/entities/cycle/api, frontend/messages/fa/home.json, frontend/messages/en/home.json, backend-go/resources/translations/fa/home.json, backend-go/resources/translations/en/home.json]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- home
---

# T-M9-47 — «پریود نداشتم» skip flow and the last-period prompt end to end
**Model:** sonnet; the main session reviews copy.

## Why
§2.39.8 lets the user say «پریود نداشتم» when the expected period did not come; the backend endpoint is T-M9-29. The
stale-anchor / unresolved banner (§2.41) must lead to a working "when did your last period start?" flow.

## Scope
1. New feature slice `features/skip-period` (`new-fsd-slice` skill): `useSkipPeriod()` → `POST /cycle/skips
   {expected_start}`, `useUndoSkip()` → `DELETE /cycle/skips/{date}`; on success use the returned `cycle_view` and
   `invalidateCycleDependents` (T-M9-41).
2. Entry points (v6 payload only): the overdue banner and the ring while `today_ui.ring_action = start` and today ≥
   `predicted_next` show «پریود نداشتم»; a confirm sheet explains what it does (one short paragraph); after success a
   toast with «برگرداندن» (undo) for the rest of the day.
3. 422 codes mapped to calm messages (not expected yet, period already started, prediction changed — refresh).
4. `ask_last_period` → `PeriodDateEditor` → period start write → the banner disappears after refetch (verify end to
   end on local Go).

## Tests
- Feature hooks with a mocked client (success, each 422, network error); entry-point visibility rules; undo.
- E2E-style integration test with the real query client and a mocked fetch: overdue → skip → banner gone → undo →
  overdue again.
- Visual QA light/dark of the confirm sheet and toast.

## Out of scope
- Backend.

## Acceptance
- Verify passes; flow checked on local Go with `CYCLE_ENGINE=v6`.
