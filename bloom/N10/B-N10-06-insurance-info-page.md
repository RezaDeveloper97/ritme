---
id: B-N10-06
title: Insurance info page
milestone: N10
type: fullstack
status: todo
depends_on: [B-N10-01]
parallel_group: N10-F
touches: [backend-go/internal/content,admin-web/src,frontend/src/screens/insurance]
skills: [verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N10-06 — Insurance info page

## Why
«بیمه» tile.

## Scope
- Admin content page (coverage, claims how-to, partner link); no health data sent.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
