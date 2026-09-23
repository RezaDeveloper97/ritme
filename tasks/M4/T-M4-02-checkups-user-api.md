---
id: T-M4-02
title: Checkups — user API (list, home, detail, records, custom, settings)
milestone: M4
type: backend
status: todo
depends_on: [T-M4-01]
parallel_group: M4-B
touches: [backend-go/db/queries/checkups, backend-go/internal/checkups, backend-go/internal/http/routes_checkups.go, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/checkups/... && make test-int PKG=./internal/checkups/...
---

# T-M4-02 — Checkups — user API (list, home, detail, records, custom, settings)

## Why
Every screen in `v14_*` reads or writes through these endpoints.

## Scope
1. `routes_checkups.go` with every `/api/v1/checkups*` route in the README contract (list with `filter`, `home`,
   detail, records CRUD + list with filters/pagination, `preview-next`, custom CRUD, settings).
2. Localized labels computed server-side (`interval_label` «هر ۳ سال», `timing_label` «روز ۷ تا ۱۰ سیکل»,
   `next_due_label` «۳ روز دیگر» / «عقب‌افتاده از فروردین» / «از ۱۴۰۹ (۴۰ سالگی)»), Jalali month names for fa,
   Gregorian for en — reuse existing i18n/civildate helpers.
3. Validation with fa/en 422s (`done_on` not in the future, `result` enum, `findings` ⊆ the type's options).
4. Ownership: records and custom types scoped by user (404 otherwise). Home endpoint ≤ 3 queries.
5. OpenAPI entries; integration tests per route.

## Acceptance
- Contract matches the README; tests green; query budget asserted for `/checkups/home`.

## Note from T-M4-05
`POST` record should return `{item, record}` (the frontend stores the on-device attachment under the record id; it
falls back to refetching the detail if `record` is missing). Findings: "nothing different" option key `none` or
`exclusive: true`.
