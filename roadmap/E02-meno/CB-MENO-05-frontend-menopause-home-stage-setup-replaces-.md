---
id: CB-MENO-05
title: Frontend: menopause home + stage setup (replaces bloom's minimal home)
epic: MENO
type: frontend
status: done
depends_on: [CB-MENO-02, B-N2-03]
parallel_group: MENO-C
touches: [frontend/src/screens/menopause, frontend/src/screens/menopause-stage, frontend/src/entities/menopause, frontend/messages/fa/menopause.json, frontend/messages/en/menopause.json, frontend/src/app/[locale]/menopause, frontend/src/app/message-scopes.ts, frontend/src/screens/home]
skills: [new-fsd-slice]
boards: [nbl_Meno_Home.dc.html, nbl_Meno_Stage.dc.html, Main.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-MENO-05 — Frontend: menopause home + stage setup (replaces bloom's minimal home)

## Why
Menopause Today and the stage questionnaire.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Home.dc.html`
- `nbl_Meno_Stage.dc.html`
- `Main.dc.html`

## Scope
1. Stage screen (4 radio cards, last-period month Jalali, surgical, HRT) — also opened from the mode switcher.
2. Home: months-without-period card + stage chip, 3 quick actions (hot flash now, log today, bleeding/spotting), today stats, score card + 6-month sparkline, 3 upcoming checkups, treatment card, doctor-report button.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
