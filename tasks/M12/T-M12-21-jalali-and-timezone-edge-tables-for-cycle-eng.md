---
id: T-M12-21
title: Jalali and timezone edge tables for cycle engine, medication schedule and reminder planner
milestone: M12
type: backend
status: todo
depends_on: []
parallel_group: M12-TEST
touches: [backend-go/internal/cycle/engine, backend-go/internal/care/medication_test.go, backend-go/internal/reminders, backend-go/internal/platform/usertz]
skills: []
verify: cd backend-go && CGO_ENABLED=0 go test -count=1 ./internal/cycle/engine/... ./internal/care/... ./internal/reminders/... ./internal/platform/usertz/...
---

# T-M12-21 — Jalali and timezone edge tables for cycle engine, medication schedule and reminder planner
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-08: the 30 Esfand leap day, non-Tehran user timezones and the midnight boundary are not covered, and these are exactly where date math breaks.

## Scope
1. Engine: period spanning 29/30 Esfand → 1 Farvardin in leap and non-leap years; cycle length across Nowruz; logs at 23:30 vs 00:30 local.
2. Medication: a daily 08:00 dose for a user in Europe/Berlin across DST change; a dose exactly at quiet-hours start/end.
3. Reminder planner: same edges.
4. Fix any bug found (opus), with the case as the regression test.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Tables green; each found bug has its own named case.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
