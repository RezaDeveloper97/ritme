---
id: B-N4-07
title: Admin — companion tips content and link overview
milestone: N4
type: fullstack
status: done
depends_on: [B-N4-03]
parallel_group: N4-G
touches: [admin-web/src,backend-go/internal/admin,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N4-07 — Admin — companion tips content and link overview

## Why
Tips are admin content.

## Scope
- CRUD for per-phase companion tips (fa/en); masked list of companion links with counts.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- admin-web build green
- `verify` green
