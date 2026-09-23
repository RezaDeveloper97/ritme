---
id: T-M7-06
title: Admin API — week details, care plan, alert rules, create-message in registered groups
milestone: M7
type: backend
status: done
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

## Note from T-M7-08 (frontend contract)
The frontend parsers are in `frontend/src/entities/pregnancy/api/v2-schema.ts` (primary shape + tolerated
alternatives) — match them unless the spec says otherwise, and report differences. Admin text comes back as plain
localized strings; dates `YYYY-MM-DD` with sibling `*_label`; 409 `pregnancy_not_active` / 404 → frontend treats as
"go to Setup". Admin `illustration_key` enum = `FETUS_ILLUSTRATION_KEYS` in
`frontend/src/shared/ui/illustrations/fetus-keys.ts`. Symptom severities `mild|moderate|severe`; mood 1–5.
