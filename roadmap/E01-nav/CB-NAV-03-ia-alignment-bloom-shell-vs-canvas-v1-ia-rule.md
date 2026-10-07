---
id: CB-NAV-03
title: IA alignment: bloom shell vs canvas-v1 IA rules
epic: NAV
type: frontend
status: in_progress
depends_on: [B-N1-04, B-N2-03, B-N3-03, B-N1-10, B-N7-01, CB-NAV-02]
parallel_group: NAV-C
touches: [frontend/src/widgets/bottom-nav, frontend/src/screens/services, frontend/src/screens/hub, frontend/src/screens/mode, frontend/src/screens/log, frontend/src/screens/home, frontend/src/widgets/quick-access]
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
3. (CB-CORE-01) Today header of every mode: mode chip (→ mode screen) next to bloom's greeting + the search icon from CB-NAV-02. Today «برای امروز» card = reminders (M3) + todo count (B-N6-08, hidden until it exists) + companion suggestion (N4, hidden until it exists). Today «دسترسی سریع»: user-picked shortcuts (record, doctor, pain diary, courses…) with an edit sheet — no bloom task has it; persistence per `docs/canvas-build/README.md` open question (local first).
4. (CB-CORE-01) + sheet rows from `nbd_Nav_Plus`: «برای بعد» (appointment → reminders form, new task → todo when B-N6-08 exists) and «ثبت برای <child>» (feed/sleep/diaper, B-N5-07) when children exist — add to bloom's log sheet if B-N3-03/B-N3-06 lack them.
5. (CB-CORE-01) Teen Services: no shop, no ads; doctor & lab tiles only when a parent is linked (IA_Nav) — until CB-TEEN-01 exists, hide them for teen. The IVF «درمان» tab wiring lives in CB-IVF-02 and the menopause «علائم» target in CB-MENO-08; here only check label/icon/position.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Every IA_Nav rule has a test or a QA line; Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
