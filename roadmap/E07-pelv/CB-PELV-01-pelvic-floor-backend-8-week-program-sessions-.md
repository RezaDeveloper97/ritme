---
id: CB-PELV-01
title: Pelvic floor backend: 8-week program, sessions, bladder diary
epic: PELV
type: backend
status: done
depends_on: [CB-CORE-03]
parallel_group: PELV-A
touches: [backend-go/internal/pelvic, backend-go/internal/http/routes_pelvic.go, backend-go/db/queries/pelvic, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations, backend-go/db/migrations, backend/database/migrations, docs/go-migration/deviations.md]
skills: [new-endpoint]
boards: [nbl_Pelvic_Plan.dc.html, nbl_Pelvic_Kegel.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/pelvic/... && make test-int PKG=./internal/pelvic/... && golangci-lint run && make schema-diff
---

# CB-PELV-01 — Pelvic floor backend: 8-week program, sessions, bladder diary

## Why
A program bloom only lists as content.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Pelvic_Plan.dc.html`
- `nbl_Pelvic_Kegel.dc.html`

## Scope
1. pelvic_programs (start, week, level from catalog pelvic_levels: hold/rest/reps/sets), sessions, streak; bladder_diary (leak none/cough/urgency/unexplained, night voids, uti symptoms[]).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- Schema change → goose migration + Laravel schema-only twin, `make schema-diff` green (backend-go/CLAUDE.md).
- New Go-only route group listed in docs/go-migration/deviations.md.
- `verify` green.
