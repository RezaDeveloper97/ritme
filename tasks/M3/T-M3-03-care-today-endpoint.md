---
id: T-M3-03
title: Care reminders — GET /care/today aggregate for the home card
milestone: M3
type: backend
status: todo
depends_on: [T-M3-02]
parallel_group: M3-C
touches: [backend-go/db/queries/care, backend-go/internal/care, backend-go/internal/http/routes_care.go, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/care/... && make test-int PKG=./internal/care/...
---

# T-M3-03 — Care reminders — GET /care/today aggregate for the home card

## Why
The home «یادآورهای امروز» card and the Reminders screen's «امروز» strip need today's doses with taken state
and the next appointment in one request (home is the hottest screen — one query budget, no N+1).

## Scope
1. `GET /api/v1/care/today?date=Y-m-d` (default = today, Tehran) with the exact shape in the README:
   `doses[]` (one per active medication × slot covering the date, sorted by slot, `taken` from `reminder_intakes`),
   `taken_count`, `total`, `next_appointment` (earliest scheduled ≥ now, else null).
2. At most 3 SQL queries; test with the clock fixed (platform/clock) across weekday filter, start/end window,
   inactive medication, cancelled appointment.
3. OpenAPI entry.

## Out of scope
Changes to `/home` sections.

## Acceptance
- Shape matches the README byte-for-byte on a fixture; tests green; query count asserted.
