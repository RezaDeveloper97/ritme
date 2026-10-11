---
id: B-N10-03
title: Shop entry and catalog (link-out)
milestone: N10
type: fullstack
status: done
depends_on: [B-N10-01]
parallel_group: N10-C
touches: [backend-go/internal/shop,backend-go/api,admin-web/src,frontend/src/screens/shop]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N10-03 — Shop entry and catalog (link-out)

## Why
«فروشگاه» in services.

## Design
- `docs/design/night-bloom/d-doctor-assistant/nbl_v17_Main.dc.html` (+ `nbd_v17_Main`)

## Scope
- `internal/shop`: admin-managed categories («سیسمونی و نوزاد», «آرایشی و بهداشتی») and products (title, image, price text, badge, nullable `external_url`, partner label, order/active); no cart/payment (CB-SHOP-01 extends this schema; link-out products must keep rendering).
- Services «فروشگاه» entry → categories → product list → external link-out with an interstitial line; keeps the design's «فروشگاه از داده سلامت تو جداست…» disclosure; shop tables/code never join or import health data; no per-user browsing history; link carries no user identifiers.
- `ShopOrders` adapter stub (`fake` = unsupported). The Me «سفارش‌ها» partner-orders row stays hidden (DECISIONS #15: internal COD orders arrive in CB-SHOP-08 at `/profile/orders`).
- Full detail: `docs/night-bloom/extras.md` §2.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
