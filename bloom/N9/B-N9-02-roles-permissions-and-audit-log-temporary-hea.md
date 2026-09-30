---
id: B-N9-02
title: Roles, permissions and audit log (temporary health-data access)
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-01]
parallel_group: N9-B
touches: [backend-go/internal/admin,backend-go/db,backend-go/api,admin-web/src]
skills: [new-endpoint,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-02 — Roles, permissions and audit log (temporary health-data access)

## Why
Least privilege; every health-data access is logged.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_Roles.dc.html` (+ `nbd_Admin_Roles`)

## Scope
- Roles: product manager, medical advisor, support, clinical support, analyst, marketing, engineering, doctor, instructor-moderator; permission matrix enforced in admin API; audit log (who, what, when, detail) with export; temporary health-data access (reason + ticket, 30 min) only for clinical support.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Permission tests per role
- `verify` green
