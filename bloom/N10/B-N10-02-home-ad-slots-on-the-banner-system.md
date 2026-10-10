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
- Reuse the existing banner system: `kind=ad`, slots `ad_slider` 342×170, `ad_square` 166×166, `ad_strip` 342×76, labelled «تبلیغ» (client-rendered, not editable); advertiser name, CTA, internal/external link (`rel=noopener sponsored`), schedule, priority; admin CRUD + schedule in admin-web.
- Targeting by app mode only (never health logs/predictions/phase); never in teen mode, loss mode, emergency/safety, consent, record/labs, doctor chat or any urgent path; frequency cap client-side only.
- Impression/click events as per-day aggregate counters (`POST /api/v1/banners/{id}/events`, rate-limited, no user id, no health payload); no ad network/SDK; advertisers are manual deals.
- Full detail: `docs/night-bloom/extras.md` §1.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
