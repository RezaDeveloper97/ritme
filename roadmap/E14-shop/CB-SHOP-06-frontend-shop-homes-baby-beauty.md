---
id: CB-SHOP-06
title: Frontend: shop homes (baby + beauty)
epic: SHOP
type: frontend
status: todo
depends_on: [CB-SHOP-02, B-N7-01]
parallel_group: SHOP-E
touches: [frontend/src/screens/shop, frontend/src/screens/shop-beauty, frontend/src/entities/shop, frontend/messages/fa/shop.json, frontend/messages/en/shop.json, frontend/src/app/[locale]/services/shop, frontend/src/app/message-scopes.ts]
skills: [new-fsd-slice]
boards: [nbd_Shop_Home.dc.html, nbd_Shop_Beauty.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-SHOP-06 — Frontend: shop homes (baby + beauty)

## Why
Shop in its own frame under Services.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Shop_Home.dc.html`
- `nbd_Shop_Beauty.dc.html`

## Scope
1. Baby home (city, search, categories, layette card, best sellers, curated set, trust row, separation note). Beauty home (categories, supply-reminder card off by default, sections, IRC trust card). Hidden for teen.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
