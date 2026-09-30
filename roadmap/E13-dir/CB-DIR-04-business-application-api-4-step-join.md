---
id: CB-DIR-04
title: Business application API (4-step join)
epic: DIR
type: backend
status: todo
depends_on: [CB-DIR-01]
parallel_group: DIR-C
touches: [backend-go/internal/cityservices/join, backend-go/internal/http/routes_cityservices.go, backend-go/db/queries/cityservices, backend-go/api/openapi.yaml]
skills: [new-endpoint]
boards: [nbl_Dir_Join.dc.html, nbl_Dir_JoinForm.dc.html, nbl_Dir_JoinDocs.dc.html, nbl_Dir_JoinDone.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/cityservices/... && make test-int PKG=./internal/cityservices/... && golangci-lint run
---

# CB-DIR-04 — Business application API (4-step join)

## Why
Owners apply; the team reviews.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Join.dc.html`
- `nbl_Dir_JoinForm.dc.html`
- `nbl_Dir_JoinDocs.dc.html`
- `nbl_Dir_JoinDone.dc.html`

## Scope
1. Auto-saved draft per step, photos (≥4, no children's faces note) + private docs, submit → tracking code + SMS, status endpoint.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- `verify` green.
