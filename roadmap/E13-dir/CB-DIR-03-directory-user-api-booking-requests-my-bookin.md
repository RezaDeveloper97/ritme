---
id: CB-DIR-03
title: Directory user API: booking requests, my bookings, cancel, reviews
epic: DIR
type: backend
status: todo
depends_on: [CB-DIR-02, B-N5-02]
parallel_group: DIR-C
touches: [backend-go/internal/cityservices/booking, backend-go/internal/http/routes_cityservices.go, backend-go/db/queries/cityservices, backend-go/api/openapi.yaml, backend-go/contract]
skills: [new-endpoint]
boards: [nbl_Dir_Book.dc.html, nbl_Dir_Booked.dc.html, nbl_Dir_MyBookings.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/cityservices/... && make test-int PKG=./internal/cityservices/... && golangci-lint run
---

# CB-DIR-03 — Directory user API: booking requests, my bookings, cancel, reviews

## Why
Bookings are requests confirmed by the Ritme team (DECISIONS).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Book.dc.html`
- `nbl_Dir_Booked.dc.html`
- `nbl_Dir_MyBookings.dc.html`

## Scope
1. Create request (child, age-fit, slot hold), upcoming/past, cancel (policy window), reschedule request, review after completed; on confirm → care reminders (1 day + 2 h) + SMS adapter; place sees only name, child name/age, phone.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Capacity race + cancel window tests.
- `verify` green.
