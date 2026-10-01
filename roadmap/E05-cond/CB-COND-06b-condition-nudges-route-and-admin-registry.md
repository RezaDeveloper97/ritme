---
id: CB-COND-06b
title: Condition nudges route and admin registry
epic: COND
type: backend
status: todo
depends_on: [CB-COND-06]
parallel_group: COND-B
touches: [backend-go/internal/admin/messages/registry,backend-go/internal/http/routes_messages.go,backend-go/internal/messages/conditionnudges,backend-go/api/openapi.yaml,backend-go/contract,docs/go-migration/deviations.md]
skills: [new-endpoint]
boards: []
verify: cd backend-go && go vet ./... && go test ./internal/messages/... ./internal/admin/messages/... && make test-int PKG=./internal/messages/conditionnudges/... && golangci-lint run && go test ./internal/http -run OpenAPI && make contract ROUTES=all
---

# CB-COND-06b — Condition nudges route and admin registry

## Why
CB-COND-06 shipped the nudge engine (`internal/messages/conditionnudges`) but, inside its touches, could neither mount a route nor register the `condition_nudge` message group for admin-web create. Without this no client sees the nudges.

## Boards
- none (backend)

## Scope
1. `conditionNudgeGroup()` in `internal/admin/messages/registry` (typed items heavy_pain / heavy_bleeding; fields title, body, action, doctor_action; placeholder `days`) — draft in CB-COND-06's PROGRESS notes.
2. Mount `GET /api/v1/messages/nudges` (auth:api, localized) → `conditionnudges.Handlers.Index`; OpenAPI operation, contract case with Go-recorded golden, deviations.md row (next free D-number — ask the bloom session).

## Out of scope
- Frontend nudge card (CB-COND-02 home/hub picks it up); dismiss/dedupe; admin-tunable thresholds.

## Acceptance
- Admin can create/edit `condition_nudge` rows; the route returns nudges for a qualifying user and `[]` otherwise (int test).
- `verify` green.
