---
id: B-N10-02
title: Home ad slots on the banner system
milestone: N10
type: fullstack
status: todo
depends_on: [B-N10-01]
parallel_group: N10-B
touches: [backend-go/internal/banners,backend-go/internal/admin,backend-go/api,admin-web/src,frontend/src/widgets]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N10-02 — Home ad slots on the banner system

## Why
Sponsored slots on home.

## Design
- `docs/design/night-bloom/a-start-plus/nbl_Prem_TrialHome.dc.html` (+ `nbd_Prem_TrialHome`)

## Scope
- Placements: slider 342×170, square 166×166, strip 342×76, labelled «تبلیغ»; targeting by mode only (never by health logs); never shown to teen mode or inside urgent paths; admin CRUD + schedule; impression/click events.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
