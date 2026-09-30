---
id: B-N4-05
title: Male onboarding and companion panel home
milestone: N4
type: frontend
status: todo
depends_on: [B-N4-03,B-N2-02]
parallel_group: N4-E
touches: [frontend/src/screens/onboarding-partner,frontend/src/screens/companion-home,frontend/src/widgets/bottom-nav]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N4-05 — Male onboarding and companion panel home

## Why
The men's side.

## Design
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Partner.dc.html` (+ `nbd_Onb_Partner`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_PartnerLinked.dc.html` (+ `nbd_Onb_PartnerLinked`)
- `docs/design/night-bloom/companion-family/nbl_Hamdam_Home.dc.html` (+ `nbd_Hamdam_Home`)

## Scope
- Gender=male → partner code entry (6 chars, what you'll see, invite link) → linked screen → companion home with its own nav.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
