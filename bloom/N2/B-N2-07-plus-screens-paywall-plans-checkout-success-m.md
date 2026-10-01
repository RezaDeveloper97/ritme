---
id: B-N2-07
title: Plus screens — paywall, plans, checkout, success, manage
milestone: N2
type: frontend
status: todo
depends_on: [B-N2-04,B-N2-05]
parallel_group: N2-G
touches: [frontend/src/screens/plus-*,frontend/src/entities/plus,frontend/src/features/purchase-plus,frontend/src/app/[locale]/plus]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N2-07 — Plus screens — paywall, plans, checkout, success, manage

## Why
User-facing purchase flow.

## Design
- `docs/design/night-bloom/a-start-plus/nbl_Prem_Paywall.dc.html` (+ `nbd_Prem_Paywall`)
- `docs/design/night-bloom/a-start-plus/nbl_Prem_Plans.dc.html` (+ `nbd_Prem_Plans`)
- `docs/design/night-bloom/a-start-plus/nbl_Prem_Checkout.dc.html` (+ `nbd_Prem_Checkout`)
- `docs/design/night-bloom/a-start-plus/nbl_Prem_Success.dc.html` (+ `nbd_Prem_Success`)
- `docs/design/night-bloom/a-start-plus/nbl_Prem_Manage.dc.html` (+ `nbd_Prem_Manage`)

## Scope
- Paywall (free vs plus table, testimonial, restore), Plans (1/3/6 months, popular badge), Checkout (discount code, VAT, method list — web shows bank gateway only; Bazaar/Myket rows hidden on web), redirect + return handling, Success, Manage (days left ring, auto-renew toggle, upgrade, payment history, cancel).

- Gateway return (from B-N2-05): Go redirects to `PLUS_CALLBACK_URL` (default `/plus/return`) with `reference`, `authority`, `status`; that page calls `POST /api/v1/plus/verify` and then shows success/failure (`/plus/success` per routes.md can be the result view). Money comes as integer rials (`currency: IRR`) — display toman (/10).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- Full purchase with the fake gateway works locally
- `verify` green
