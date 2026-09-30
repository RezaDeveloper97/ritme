---
id: B-N10-04
title: Home lab test and screening booking requests
milestone: N10
type: fullstack
status: todo
depends_on: [B-N10-01]
parallel_group: N10-D
touches: [backend-go/internal/partners,backend-go/api,admin-web/src,frontend/src/screens/home-lab]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N10-04 — Home lab test and screening booking requests

## Why
Home suggestions «آزمایش در منزل» / «چکاپ سالانه — رزرو».

## Scope
- Request form → partner adapter (fake), status, results can land in labs (B-N6-06) with consent.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
