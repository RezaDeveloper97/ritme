---
id: T-M9-53
title: Postpartum before cycles return: no cycle day/period ring; skip fertile-window pushes while suppressed
milestone: M9
type: backend
status: todo
depends_on: [T-M9-44]
parallel_group: M9-C
touches: [backend-go/internal/cycle/engine,backend-go/internal/cycle/view,backend-go/internal/notifications,backend-go/contract/golden-v6]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... ./internal/notifications/... && make test-int PKG="./internal/cycle/... ./internal/notifications/..." && make contract ROUTES=cycle
---

# T-M9-53 — Postpartum before cycles return: no cycle day/period ring; skip fertile-window pushes while suppressed

## Why
T-M9-44 QA (postpartum persona: birth 2026-10-20, breastfeeding, `postpartum_cycles_not_returned`): Today shows
«روز ۱۶ چرخه», period dots on the ring and the daily card «یک روز معمول چرخه» eight days after birth. Before cycles
return there is no cycle to count. Also T-M9-44 hid the fertile-window reminder toggle while fertility is suppressed,
but a server-side sender may still push `fertile_window` (`internal/notifications/prefs.go` defines the category).

## Scope
- Check SPEC v6/DECISIONS for postpartum-not-returned output (cite it); the engine/view should not emit a cycle day,
  period ring days or "usual cycle day" copy in that state.
- Trace whether a scheduled `fertile_window` sender exists; if yes, skip users with `limits.fertility_suppressed`.
  If none exists, write that down in the task and PROGRESS and close that half.

## Out of scope
- Frontend (T-M9-42e handles the empty hero/header).

## Acceptance
- Tests for the postpartum state and (if a sender exists) the push skip; v1 contract unchanged; verify green.
