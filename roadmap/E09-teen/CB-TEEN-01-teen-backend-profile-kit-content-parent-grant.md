---
id: CB-TEEN-01
title: Teen backend: profile, kit, content, parent grant type
epic: TEEN
type: backend
status: todo
depends_on: [B-N2-03, B-N4-02, CB-CORE-03]
parallel_group: TEEN-A
touches: [backend-go/internal/teen, backend-go/internal/http/routes_teen.go, backend-go/db/queries/teen, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations, backend-go/db/migrations, backend/database/migrations, docs/go-migration/deviations.md, backend-go/internal/companion]
skills: [new-endpoint]
boards: [nbl_Teen_Onb.dc.html, nbl_Teen_Home.dc.html, nbl_Teen_Parent.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/teen/... && make test-int PKG=./internal/teen/... && golangci-lint run && make schema-diff
---

# CB-TEEN-01 — Teen backend: profile, kit, content, parent grant type

## Why
bloom gives teen mode a simplified cycle home; the canvas adds onboarding, kit, FAQ and mother sharing. Built ON TOP of the bloom/ queue: never re-implement what a `B-N*` dependency delivers — extend it.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Teen_Onb.dc.html`
- `nbl_Teen_Home.dc.html`
- `nbl_Teen_Parent.dc.html`

## Scope
1. teen_profiles (age band 10–12/13–15/16–17, menarche not-yet/<1y/>1y), kit checklist state, readiness text.
2. Extend bloom companion with type `parent` and teen-only grants: next period as week only, kit reminder, optional notes; parent read returns only that card; parent can never write.
3. Server flag disables shop, ads/banners and commercial recommendations for teen users (verify bloom's rule, extend if needed). Catalog teen_faq, teen_signs, teen_kit_items `[needs review]`.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- Schema change → goose migration + Laravel schema-only twin, `make schema-diff` green (backend-go/CLAUDE.md).
- New Go-only route group listed in docs/go-migration/deviations.md.
- `verify` green.
