---
id: B-N6-08
title: To-do list with categories, shopping list and cycle suggestions
milestone: N6
type: fullstack
status: done
depends_on: [B-N5-11]
parallel_group: N6-H
touches: [backend-go/internal/todo,backend-go/db,backend-go/api,frontend/src/screens/todo*,frontend/src/entities/todo]
skills: [new-endpoint,new-fsd-slice,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N6-08 — To-do list with categories, shopping list and cycle suggestions

## Why
Section و · ابزارها.

## Design
- `docs/design/night-bloom/f-tools/nbl_Todo_Home.dc.html` (+ `nbd_Todo_Home`)
- `docs/design/night-bloom/f-tools/nbl_Todo_List.dc.html` (+ `nbd_Todo_List`)
- `docs/design/night-bloom/f-tools/nbl_Todo_Add.dc.html` (+ `nbd_Todo_Add`)
- `docs/design/night-bloom/f-tools/nbl_Todo_Empty.dc.html` (+ `nbd_Todo_Empty`)

## Scope
- Tasks (title, category shopping/work/personal/health, due date/time, reminder), shopping lists with items, done section, empty state, smart suggestion from cycle («خرید نوار بهداشتی» before period).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
