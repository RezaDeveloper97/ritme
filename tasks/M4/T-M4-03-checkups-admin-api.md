---
id: T-M4-03
title: Checkups — admin API for the checkup-type catalog and stats
milestone: M4
type: backend
status: todo
depends_on: [T-M4-01]
parallel_group: M4-B
touches: [backend-go/internal/admin/checkups, backend-go/internal/http/routes_admin_checkups.go, backend-go/db/queries/checkups, docs/go-migration/admin-api.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/admin/checkups/... && make test-int PKG=./internal/admin/checkups/...
---

# T-M4-03 — Checkups — admin API for the checkup-type catalog and stats

## Why
The user wants the admin panel to manage this feature: the catalog content (titles, why, prep steps, self-exam
guide, findings, intervals, age window, cycle timing, visibility, order) must be editable without a deploy.

## Scope
1. `internal/admin/checkups` following the `admin/content` pattern (`httpadmin` chain, session + CSRF, roles
   `editor`/`super`): list (search, active filter), show, create, update, delete (refuse if records exist → 422
   `in_use`, offer deactivate), `reorder`, `stats` (README contract). Custom (user-owned) types are excluded.
2. Validation: bilingual required for fa (+ every active content language), interval > 0, `age_min ≤ age_max`,
   `cycle_day_from ≤ cycle_day_to ≤ 45`, icon/tone enums, step arrays ≤ 10 items.
3. Document the endpoints in `docs/go-migration/admin-api.md` (same table format).
4. Integration tests incl. role check and CSRF.

## Acceptance
- CRUD/reorder/stats green in tests; editing a type is visible immediately in `GET /api/v1/checkups/{id}`.
