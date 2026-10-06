---
id: CB-REC-01
title: Record documents + timeline backend (on bloom's health record)
epic: REC
type: backend
status: done
depends_on: [B-N6-03, CB-CORE-05]
parallel_group: REC-A
touches: [backend-go/internal/healthrecord, backend-go/internal/http/routes_healthrecord.go, backend-go/db/queries/healthrecord, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations, backend-go/db/migrations, backend/database/migrations, docs/go-migration/deviations.md]
skills: [new-endpoint]
boards: [nbl_Rec_Home.dc.html, nbl_Rec_Timeline.dc.html, nbl_Rec_Doc.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/healthrecord/... && make test-int PKG=./internal/healthrecord/... && golangci-lint run && make schema-diff
---

# CB-REC-01 — Record documents + timeline backend (on bloom's health record)

## Why
bloom's record (B-N6-03) is a summary; labs (B-N6-06) handles lab sheets. The canvas adds every document kind and a timeline. Built ON TOP of the bloom/ queue: never re-implement what a `B-N*` dependency delivers — extend it.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Rec_Home.dc.html`
- `nbl_Rec_Timeline.dc.html`
- `nbl_Rec_Doc.dc.html`

## Scope
1. record_documents (kind imaging/visit/prescription/hospital/other — labs stay in labs but appear in the timeline), date, centre, doctor, file ids, extracted JSON, review state, links (claims, pregnancy).
2. Record extras: allergies shown on emergency card flag, family history, surgeries if B-N6-03 lacks them; category counts; timeline by month with filters incl. labs; document detail with 'where used'.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- Schema change → goose migration + Laravel schema-only twin, `make schema-diff` green (backend-go/CLAUDE.md).
- New Go-only route group listed in docs/go-migration/deviations.md.
- `verify` green.
