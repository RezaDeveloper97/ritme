---
id: B-N2-03
title: Life-stage mode switcher (6 modes) and minimal menopause/teen homes
milestone: N2
type: fullstack
status: todo
depends_on: [B-N2-01,B-N1-10]
parallel_group: N2-C
touches: [frontend/src/screens/mode,frontend/src/entities/user,frontend/src/screens/home,backend-go/internal/profile,backend-go/internal/home,backend-go/api]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N2-03 — Life-stage mode switcher (6 modes) and minimal menopause/teen homes

## Why
Me → حالت اپ lists six modes, each changing tabs, home and log tiles.

## Design
- `docs/design/night-bloom/g-me-settings/nbl_Me_Mode.dc.html` (+ `nbd_Me_Mode`)

## Scope
- Mode screen per artboard (tab + log hint per mode, IVF/IUI toggle, contraception tracking, data kept).
- Menopause home: reuse cycle home layout with menopause copy + symptom tiles (hot flash, sleep, mood); teen: simplified cycle home, no ads/shop. Record the missing-artboard decision in `docs/night-bloom/README.md`.
- Pregnancy-loss path: a calm exit option inside pregnancy mode (copy from admin), no celebratory content afterwards.
- (B-N1-01) Defaults already recorded in `docs/night-bloom/gaps.md` #3 and `nav.md` (menopause tab «علائم» → `/analysis/symptoms`; teen tab «تقویم», no ads/shop/Plus upsell); the full menopause/teen versions come from `roadmap/E02-meno` / `E09-teen`.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
