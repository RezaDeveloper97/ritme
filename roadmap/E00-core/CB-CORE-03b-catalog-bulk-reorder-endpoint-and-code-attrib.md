---
id: CB-CORE-03b
title: Catalog bulk reorder endpoint and code attribute label
epic: CORE
type: backend
status: done
depends_on: [CB-CORE-04]
parallel_group: CORE-B
touches: [backend-go/internal/catalog,backend-go/resources/lang,backend-go/db/queries/catalog,docs/canvas-build/catalog.md,backend-go/internal/http/routes_admin_catalog.go,backend-go/api/openapi.yaml,backend-go/resources/translations,admin-web/src/screens/catalog]
skills: [new-endpoint]
boards: []
verify: cd backend-go && go vet ./... && go test ./internal/catalog/... && make test-int PKG=./internal/catalog/... && golangci-lint run && cd ../admin-web && npm run typecheck && npm run test
---

# CB-CORE-03b — Catalog bulk reorder endpoint and code attribute label

## Why

## Boards

## Scope
1.

## Out of scope

## Acceptance
-
## Why
CB-CORE-04 reorders items with one partial PUT per moved row (not transactional), and the 422 for a duplicate catalog `code` reads «کد تایید قبلاً ثبت شده است.» because the shared `code` attribute label means OTP code.

## Scope
1. `POST /api/admin/v1/catalog/:group/reorder` `{ids:[…]}` → one transaction rewriting sort_order 1..n (editor+, CSRF, audit); OpenAPI + int test.
2. Catalog-specific attribute label for `code` (fa «کد آیتم» / en "item code") so duplicate-code errors read correctly.
3. admin-web catalog screen: use the new endpoint instead of per-row PUTs.

## Out of scope
- Anything else in the catalog editor.

## Acceptance
- Reorder is atomic (int test); duplicate code 422 shows the catalog label; `verify` green.
