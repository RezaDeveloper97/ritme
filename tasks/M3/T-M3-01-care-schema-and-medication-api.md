---
id: T-M3-01
title: Care reminders — schema, reminder_intakes table and medication API (Go)
milestone: M3
type: backend
status: done
depends_on: []
parallel_group: M3-A
touches: [backend-go/db/migrations, backend-go/db/queries/care, backend-go/sqlc.yaml, backend-go/internal/care, backend-go/internal/http/routes_care.go, backend/database/migrations, backend-go/api/openapi.yaml, docs/go-migration/deviations.md, docs/care-reminders/README.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/care/... && make test-int PKG=./internal/care/... && make schema-diff
---

# T-M3-01 — Care reminders — schema, reminder_intakes table and medication API (Go)

## Why
The new reminders design (docs/care-reminders/README.md, artboards `v13_AddMedication`, `v13_Reminders`) needs
structured medication reminders (dose, form, 1–4 daily times, weekdays, amount, duration) and per-dose
"taken" tracking. The user chose **Go only**: new endpoints live under `/api/v1/care/` in `backend-go/`.

## Scope
1. Migration pair (migrations.md rule): goose `db/migrations/00002_reminder_intakes.sql` **and** a schema-only
   Laravel twin in `backend/database/migrations/` creating `reminder_intakes` exactly as in the README.
   `make schema-diff` green. No Laravel model/controller.
2. `internal/care` package: medication meta type (v1) with parse/validate, Laravel-compatible 422 family with fa/en
   messages (reuse `platform/validation`), derived `subtitle` and `recurrence`/`recurrence_time` for legacy readers,
   `pregnancy_end` duration → `ends_on` from the active pregnancy's due date.
3. sqlc queries `db/queries/care/` (+ `sqlc.yaml` package): CRUD scoped by `user_id` on `reminders` with
   `type='medication'`; intakes insert-ignore / delete.
4. `routes_care.go` registering: `GET/POST /api/v1/care/medications`, `GET/PUT/DELETE /api/v1/care/medications/{id}`,
   `POST/DELETE /api/v1/care/medications/{id}/intakes`, `GET /api/v1/care/enums` (forms, units, durations,
   plus the appointment enums for T-M3-02 — kinds, topics, remind_before — localized by Accept-Language).
5. OpenAPI entries for every route (T-M2-19 conventions) and a `deviations.md` note: "`/api/v1/care/*` is Go-only,
   no Laravel golden (new feature, M3)".
6. Handler + integration tests: validation errors, ownership (404 on another user's id), intake idempotency,
   weekday/start/end window.

## Out of scope
Appointments (T-M3-02), `/care/today` (T-M3-03), push notification delivery, nginx routing (T-M3-09).

## Acceptance
- Migration pair present; `make schema-diff` exit 0.
- All medication routes behave per the README contract; 422 messages in fa and en.
- Rows are visible through the legacy `GET /api/v1/reminders` (type `medication`, sensible title/subtitle/time).
- `go vet`, unit + integration tests green; OpenAPI validates.
