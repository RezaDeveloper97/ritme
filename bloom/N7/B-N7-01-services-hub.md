---
id: B-N7-01
title: Services hub (خدمات)
milestone: N7
type: fullstack
status: todo
depends_on: [B-N6-10]
parallel_group: N7-A
touches: [frontend/src/screens/services,backend-go/internal/services,backend-go/api]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N7-01 — Services hub (خدمات)

## Why
Replaces the N1 placeholder.

## Design
- `docs/design/night-bloom/d-doctor-assistant/nbl_v17_Main.dc.html` (+ `nbd_v17_Main`)

## Scope
- Search, upcoming booking card, care tiles (assistant, doctors, record, labs, vitals, insurance, checkups & meds), care programs (admin content: pain/endometriosis, PMDD, heavy bleeding, pelvic floor, contraception), mother & child, learning, shop, emergency 115 card. Section config from admin.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
