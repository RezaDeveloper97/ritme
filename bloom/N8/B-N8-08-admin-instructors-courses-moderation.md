---
id: B-N8-08
title: Admin — instructors & courses moderation
milestone: N8
type: fullstack
status: in_progress
depends_on: [B-N8-01]
parallel_group: N8-H
touches: [admin-web/src,backend-go/internal/admin,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N8-08 — Admin — instructors & courses moderation

## Why
Sidebar «مدرسین و دوره‌ها».

## Scope
- Approve/revoke instructors, course list, content review queue (feeds B-N9-10 safety queue), usage stats.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- admin-web build green
- `verify` green
