---
id: CB-MENO-09
title: Frontend: bleeding alert + menopause checkups
epic: MENO
type: frontend
status: done
depends_on: [CB-MENO-05, B-N1-15, CB-MENO-01b]
parallel_group: MENO-D
touches: [frontend/src/screens/menopause-alert, frontend/src/app/[locale]/menopause/alert, frontend/src/screens/checkups]
skills: [new-fsd-slice]
boards: [nbl_Meno_Alert.dc.html, nbl_Meno_Checkups.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-MENO-09 — Frontend: bleeding alert + menopause checkups

## Why
Post-menopause bleeding must reach a doctor; checkups reuse M4 (restyled by B-N1-15).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Alert.dc.html`
- `nbl_Meno_Checkups.dc.html`

## Scope
1. Alert: danger card, appointment CTA (M3 form / doctors when N7 exists), prepare-report CTA, 'tell early' list (catalog), 115 link.
2. Checkups filtered by audience=menopause with the board's groups and status chips; 'add lab result' → labs upload (B-N6-07).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
