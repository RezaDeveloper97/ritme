---
id: CB-LOSS-02
title: Frontend: loss start → care → next (full-screen)
epic: LOSS
type: frontend
status: done
depends_on: [CB-LOSS-01, CB-CORE-02]
parallel_group: LOSS-B
touches: [frontend/src/screens/loss-start, frontend/src/screens/loss-care, frontend/src/screens/loss-next, frontend/src/entities/loss, frontend/messages/fa/loss.json, frontend/messages/en/loss.json, frontend/src/app/[locale]/loss, frontend/src/app/message-scopes.ts, frontend/src/screens/pregnancy, frontend/src/screens/mode]
skills: [new-fsd-slice]
boards: [nbl_Loss_Start.dc.html, nbl_Loss_Care.dc.html, nbl_Loss_Next.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-LOSS-02 — Frontend: loss start → care → next (full-screen)

## Why
Replaces bloom's exit option with the full path; entry from pregnancy mode and IVF negative.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Loss_Start.dc.html`
- `nbl_Loss_Care.dc.html`
- `nbl_Loss_Next.dc.html`

## Scope
1. Start (types, date, stop-content switch, tell-companion switch), Care (danger card + 115, follow-up rows, mood chips, counsellor row hidden until N7 doctors exist, private note, hotlines), Next (3 options).
2. No tab bar, no banners/ads/shop, no celebratory UI.
3. (CB-CORE-01) Entry = bloom's calm exit option inside pregnancy mode (B-N2-03) and the IVF negative result (CB-IVF-05): point it at `/loss` instead of building a second entry.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
