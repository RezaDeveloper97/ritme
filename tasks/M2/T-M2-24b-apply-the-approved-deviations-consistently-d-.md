---
id: T-M2-24b
title: Apply the approved deviations consistently (D-01, D-02, D-16) and mark all decided
milestone: M2
type: backend
status: done
depends_on: [T-M2-24]
parallel_group: M2-I
touches: [backend-go/internal/reminder,backend-go/internal/healthlog,backend-go/internal/pregnancy,backend-go/internal/profile,backend-go/internal/home,backend-go/internal/cycle,backend-go/internal/content,backend-go/internal/messages,backend-go/internal/admin/content,backend-go/contract/allowlist,backend-go/api/openapi.yaml,docs/go-migration/deviations.md, docs/go-migration/admin-api.md]
skills: []
verify: cd backend-go && go test ./... && make contract ROUTES=all && go test ./internal/http/... -run OpenAPI
---

# T-M2-24b — Apply the approved deviations consistently (D-01, D-02, D-16) and mark all decided

## Why
On 2026-09-23 the user approved every proposed deviation in `docs/go-migration/deviations.md` (D-01…D-04,
D-08…D-16). Some are implemented inconsistently: D-02 (non-numeric typed `{id}` → 404) is applied in pregnancy but
reminders (and possibly others) still reproduce Laravel's 500; D-01 (3rd locale on `/reminders/enums` → English
labels) and D-16 (internal banner `link_url` must start with `/`) are not implemented.

## Scope
1. D-01: `/reminders/enums` with a locale that has no labels falls back to English instead of 500.
2. D-02: every route with a typed integer param (`{id}`, `{week}`, …) answers non-numeric values with the standard
   JSON 404 (same body as a missing model / route-not-found as appropriate — match what pregnancy already does).
   Grep all routes_*.go; make them consistent.
3. D-16: admin banner create/update — when link type is internal, `link_url` must start with `/` (and not `//`);
   422 with a validation message otherwise. Test it.
4. Contract: add allow-list entries (with the D-nn id) in `backend-go/contract/allowlist/<group>.yaml` for every
   golden that now legitimately differs (D-01, D-02). Nothing else may be allow-listed.
5. OpenAPI: update the documented 500s for D-01/D-02 to the new responses.
6. `deviations.md`: set every proposed row to `decided (approved 2026-09-23)`; D-11 stays "preserve".

## Acceptance
- `make contract ROUTES=all` green with only D-01/D-02 allow-list entries; `go test ./...` green; OpenAPI test green.
- No route anywhere still answers a non-numeric typed id with 500.
