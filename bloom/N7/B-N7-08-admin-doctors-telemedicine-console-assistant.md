---
id: B-N7-08
title: Admin — doctors & telemedicine console, assistant departments
milestone: N7
type: fullstack
status: todo
depends_on: [B-N7-04,B-N7-06]
parallel_group: N7-H
touches: [admin-web/src,backend-go/internal/admin,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N7-08 — Admin — doctors & telemedicine console, assistant departments

## Why
Sidebar «پزشکان و تله‌مدیسین».

## Scope
- Doctors CRUD + slots, bookings list, doctor reply console (chat + prescription) under a `doctor` admin role, assistant department prompt/guardrail editor with medical-approval flag.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- admin-web build green
- `verify` green
