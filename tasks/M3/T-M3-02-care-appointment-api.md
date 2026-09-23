---
id: T-M3-02
title: Care reminders — appointment API with prep checklist and cancel (Go)
milestone: M3
type: backend
status: todo
depends_on: [T-M3-01]
parallel_group: M3-B
touches: [backend-go/db/queries/care, backend-go/internal/care, backend-go/internal/http/routes_care.go, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/care/... && make test-int PKG=./internal/care/...
---

# T-M3-02 — Care reminders — appointment API with prep checklist and cancel (Go)

## Why
Artboards `v13_AddAppointment` and `v13_AppointmentDetail` need doctor visits / phone / online consultations with
who, specialty, topic, place, remind-before, add-to-calendar flag and a checkable "قبل از نوبت" list.

## Scope
1. Appointment meta type (v1) in `internal/care`: `kind`, `with`, `specialty`, `topic`, `location`,
   `remind_before`, `add_to_calendar`, `prep[]` (server assigns stable item ids), `status`. Validation + fa/en 422.
   `scheduled_at` required and parsed as Tehran wall-clock; `title` falls back to the topic label.
2. Routes: `GET /api/v1/care/appointments?scope=upcoming|past|all`, `POST`, `GET/PUT/DELETE /{id}`,
   `POST /{id}/cancel` (status=cancelled, is_active=false), `PATCH /{id}/prep/{itemId}` `{done}`.
3. Response includes computed `remind_at` (scheduled_at − remind_before) and `days_until` (civil days, Tehran).
4. OpenAPI entries; handler + integration tests (ownership, scope filter, cancel excluded from upcoming,
   prep toggle, unknown item 404).

## Out of scope
Writing to the phone calendar (the frontend produces an .ics — T-M3-08), notification delivery.

## Acceptance
- Contract in docs/care-reminders/README.md holds for every route; legacy `GET /reminders` shows the rows as
  `type=appointment` with `scheduled_at`.
- Tests green.
