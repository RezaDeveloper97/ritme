---
id: CB-INS-01
title: Insurance backend (self-tracking): policy, coverage, claims, questionnaire, centres
epic: INS
type: backend
status: todo
depends_on: [CB-REC-01, B-N5-02, B-N10-06]
parallel_group: INS-A
touches: [backend-go/internal/insurance, backend-go/internal/http/routes_insurance.go, backend-go/db/queries/insurance, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations, backend-go/db/migrations, backend/database/migrations, docs/go-migration/deviations.md]
skills: [new-endpoint]
boards: [nbl_Ins_Home.dc.html, nbl_Ins_Coverage.dc.html, nbl_Ins_Claims.dc.html, nbl_Ins_ClaimNew.dc.html, nbl_Ins_ClaimDetail.dc.html, nbl_Ins_Health.dc.html, nbl_Ins_Status.dc.html, nbl_Ins_Centers.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/insurance/... && make test-int PKG=./internal/insurance/... && golangci-lint run && make schema-diff
---

# CB-INS-01 — Insurance backend (self-tracking): policy, coverage, claims, questionnaire, centres

## Why
bloom has an insurance info page (B-N10-06); the canvas has full self-tracking (DECISIONS: user tracks it herself, no insurer API).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Ins_Home.dc.html`
- `nbl_Ins_Coverage.dc.html`
- `nbl_Ins_Claims.dc.html`
- `nbl_Ins_ClaimNew.dc.html`
- `nbl_Ins_ClaimDetail.dc.html`
- `nbl_Ins_Health.dc.html`
- `nbl_Ins_Status.dc.html`
- `nbl_Ins_Centers.dc.html`

## Scope
1. policies (insurer from catalog, masked number, valid to, members = self + bloom family/children, deductibles %, deadline days, categories with caps + waiting periods); coverage used/remaining from paid claims.
2. claims (member, cost type, centre, date, amount, docs = record documents/labs, user-maintained status timeline incl. docs-requested + deadline, expected payable, notes), year totals, filters, needs-action count.
3. Questionnaire drafts prefilled from the record ('from record' flags), user confirms; export PDF only — nothing is sent anywhere.
4. Contracted centres: admin list (type, lat/lng, direct vs reimbursed). The info page from B-N10-06 stays as 'how it works'.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- Schema change → goose migration + Laravel schema-only twin, `make schema-diff` green (backend-go/CLAUDE.md).
- New Go-only route group listed in docs/go-migration/deviations.md.
- `verify` green.
