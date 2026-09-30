---
id: B-N8-07
title: instructor-web — groups, students, add users by phone/CSV
milestone: N8
type: frontend
status: todo
depends_on: [B-N8-06]
parallel_group: N8-G
touches: [instructor-web/src,backend-go/internal/learning,backend-go/api]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd instructor-web && npm run typecheck && npm run lint && npm run test && npm run build
---

# B-N8-07 — instructor-web — groups, students, add users by phone/CSV

## Why
Access management.

## Design
- `docs/design/night-bloom/e-learning-instructor/nbl_Ins_Groups.dc.html` (+ `nbd_Ins_Groups`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Ins_Group.dc.html` (+ `nbd_Ins_Group`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Ins_Students.dc.html` (+ `nbd_Ins_Students`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Ins_AddUser.dc.html` (+ `nbd_Ins_AddUser`)

## Scope
- Groups list/detail (members, progress, pending signups, SMS invite), Students (filters active/pending/expired), Add users (paste phones, CSV/Excel upload, scope group/course/standalone, duration, preview of what the student sees).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- `verify` green
