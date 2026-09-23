---
id: T-M7-06
title: Admin API — week details, care plan, alert rules, create-message in registered groups
milestone: M7
type: backend
status: todo
depends_on: [T-M7-01]
parallel_group: M7-B
touches: [backend-go/internal/admin/content, backend-go/internal/admin/pregnancy, backend-go/internal/admin/messages, backend-go/internal/http/routes_admin_pregnancy.go, docs/go-migration/admin-api.md]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/admin/... && make test-int PKG=./internal/admin/...
---

# T-M7-06 — Admin API — week details, care plan, alert rules, create-message

## Scope
1. `GET/PUT /api/admin/v1/pregnancy-weeks/{n}/details` (structured fields, bilingual, arrays ≤ 10, illustration key
   enum).
2. `/api/admin/v1/pregnancy-care-items` CRUD + reorder (delete refused when linked appointments exist → deactivate).
3. `/api/admin/v1/pregnancy-alert-rules` list/show/update: typed params per rule (schema from the T-M7-04 registry),
   level, enabled, texts per locale — persisted as `message_contents` rows.
4. `POST /api/admin/v1/messages`: create a row only for a registered group + key (registry), with payload shape
   validation; list shows missing keys of registered groups.
5. Roles editor/super, CSRF; tests; admin-api.md updated.

## Acceptance
- Every admin-defined area in docs/pregnancy-v2/README.md is writable through the API; tests green.
