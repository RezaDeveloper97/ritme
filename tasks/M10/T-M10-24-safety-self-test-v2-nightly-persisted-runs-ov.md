---
id: T-M10-24
title: Safety self-test v2: nightly persisted runs over the live rules and registry
milestone: M10
type: fullstack
status: todo
depends_on: [T-M10-08, T-M10-19, T-M10-25, T-M9-06, T-M9-12]
parallel_group: M10-C
touches: [backend-go/internal/safety, backend-go/internal/http/routes_admin_safety.go, backend-go/db/queries/safety, docs/engine-v6/jobs.md, admin-web/src/screens/safety, admin-web/messages/fa.json, admin-web/messages/en.json]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/safety/... && make test-int PKG=./internal/safety/... && cd ../admin-web && npm run typecheck && npm run lint && npm run test
---

# T-M10-24 — Safety self-test v2: nightly persisted runs over the live rules and registry
**Model:** opus (safety).

## Why
SPEC §5.8: an automatic nightly test of every emergency scenario. Today 16 scenarios run on pure helpers in every API
process, `reduced_kicks` tested a dashboard counter (fixed in T-M9-05), there is no scenario for live rule rows or
copy, and results live in process memory (lost on restart) (gap 04 row 7.4). After T-M10-08 the pregnancy paths run
through the message engine, so the test must exercise that path. D-183.

## Scope
1. Job `safety.nightly` (scheduler, worker, queue `critical`, unique per day) persisting one row per scenario in
   `safety_scenario_runs` (T-M10-25): passed, duration, detail.
2. Two kinds of checks:
   - **Path scenarios** (in memory, no user data written): for each scenario, inputs at threshold − 1, threshold and
     threshold + 1 using the registry values valid today, through the same functions the API calls (vitals classify,
     kick-session rule, 5-1-1, postpartum red flags, EPDS, pregnancy BP via vitals, GDM, and the `preg.*` rules through
     the compiled message engine).
   - **Live configuration checks:** every locked emergency rule exists, is active and has fa/en copy; every emergency
     copy contains 115 (and 123 for EPDS Q10); the registry has a valid version today for every emergency key.
3. A failed run raises the metric `safety_selftest_failed` (T-M10-31 turns it into a dashboard alert) and logs one
   line per failure (no user data).
4. Admin-web safety screen: last 30 runs, per-scenario status and the failure detail.
5. Remove the in-process loop (T-M9-11 already scheduled it); catalogue entry in jobs.md.

## Tests
- Every scenario passes on today's config; deliberately breaking one (deactivate a locked rule row in the test DB, set
  a registry value so a boundary moves) makes exactly the expected scenario fail.
- The run is persisted and survives a process restart (read back from the DB).
- Admin-web vitest for the runs list.

## Out of scope
- Aggregate emergency statistics (existing dashboard).

## Acceptance
- Verify passes; PROGRESS.md cites D-183.
