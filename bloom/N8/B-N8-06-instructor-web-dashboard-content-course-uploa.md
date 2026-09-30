---
id: B-N8-06
title: instructor-web — dashboard, content, course, upload
milestone: N8
type: frontend
status: todo
depends_on: [B-N8-05,B-N8-02]
parallel_group: N8-F
touches: [instructor-web/src,backend-go/internal/learning,backend-go/api]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd instructor-web && npm run typecheck && npm run lint && npm run test && npm run build
---

# B-N8-06 — instructor-web — dashboard, content, course, upload

## Why
Instructor content side.

## Design
- `docs/design/night-bloom/e-learning-instructor/nbl_Ins_Dashboard.dc.html` (+ `nbd_Ins_Dashboard`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Ins_Content.dc.html` (+ `nbd_Ins_Content`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Ins_Course.dc.html` (+ `nbd_Ins_Course`)
- `docs/design/night-bloom/e-learning-instructor/nbl_Ins_Upload.dc.html` (+ `nbd_Ins_Upload`)

## Scope
- Dashboard (active students, courses, watch hours, groups, quick actions, activity), Content list (filters), Course detail (lessons reorder, access list), Upload (resumable progress, placement in course/chapter or standalone, visibility, publish/draft).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- Upload resumes after refresh
- `verify` green
