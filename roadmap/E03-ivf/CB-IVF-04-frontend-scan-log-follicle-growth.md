---
id: CB-IVF-04
title: Frontend: scan log + follicle growth
epic: IVF
type: frontend
status: todo
depends_on: [CB-IVF-02]
parallel_group: IVF-C
touches: [frontend/src/screens/ivf-scan, frontend/src/app/[locale]/(app)/ivf/scan]
skills: [new-fsd-slice]
boards: [nbl_IVF_Scan.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-IVF-04 — Frontend: scan log + follicle growth

## Why
Ultrasound numbers in one place.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_IVF_Scan.dc.html`

## Scope
1. Per-ovary NumberSteppers per bin, endometrium + E2, growth chart (two series), 'interpretation is your doctor's' note.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
