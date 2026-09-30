---
id: CB-NAV-03
title: IA alignment: bloom shell vs canvas-v1 IA rules
epic: NAV
type: frontend
status: todo
depends_on: [B-N1-04, B-N2-03, B-N3-03, B-N1-10, B-N7-01, CB-NAV-02]
parallel_group: NAV-C
touches: [frontend/src/widgets/bottom-nav, frontend/src/screens/services, frontend/src/screens/hub, frontend/src/screens/mode, frontend/src/screens/log]
skills: [new-fsd-slice]
boards: [IA_Nav.dc.html, IA_Map.dc.html, nbd_Nav_Today.dc.html, nbd_Nav_Stage.dc.html, nbd_Nav_Plus.dc.html, nbd_Nav_Services.dc.html, nbd_Nav_Me.dc.html, nbd_Nav_Mode.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-NAV-03 — IA alignment: bloom shell vs canvas-v1 IA rules

## Why
The menopause canvas adds explicit IA rules; bloom built the shell from its own boards. Close the differences.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `IA_Nav.dc.html`
- `IA_Map.dc.html`
- `nbd_Nav_Today.dc.html`
- `nbd_Nav_Stage.dc.html`
- `nbd_Nav_Plus.dc.html`
- `nbd_Nav_Services.dc.html`
- `nbd_Nav_Me.dc.html`
- `nbd_Nav_Mode.dc.html`

## Scope
1. Check + fix: tab bar hidden on forms/log/checkout/loss path/lock; + always opens the mode's log sheet; re-tap active tab → tab root; stage-tab label/icon per mode incl. IVF 'درمان' and menopause 'علائم'; Services order (care, programs, mother & child, learning, shop LAST in a separate frame with the separation note, emergency 115); teen Services without shop; Me contains no health tool that is reachable only from Me; bookings + orders rows in Me.
2. Stage tab for cycle: تقویم · تحلیل · تاریخچه inside the tab (analysis moved next to its data) if bloom placed it elsewhere.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Every IA_Nav rule has a test or a QA line; Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
