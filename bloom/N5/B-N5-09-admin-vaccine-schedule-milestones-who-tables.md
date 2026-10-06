---
id: B-N5-09
title: Admin — vaccine schedule, milestones, WHO tables, postpartum content
milestone: N5
type: fullstack
status: done
depends_on: [B-N5-02]
parallel_group: N5-I
touches: [admin-web/src,backend-go/internal/admin,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N5-09 — Admin — vaccine schedule, milestones, WHO tables, postpartum content

## Why
Clinical catalogs are admin data.

## Scope
- CRUD for vaccine schedule and milestones (fa/en), read-only WHO table viewer/import, postpartum tips.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- admin-web build green
- `verify` green
