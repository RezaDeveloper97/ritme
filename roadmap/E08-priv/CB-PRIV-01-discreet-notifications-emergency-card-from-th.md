---
id: CB-PRIV-01
title: Discreet notifications + emergency card from the lock screen
epic: PRIV
type: fullstack
status: todo
depends_on: [B-N1-12, B-N1-11, CB-REC-03]
parallel_group: PRIV-A
touches: [frontend/src/screens/privacy, frontend/src/features/app-lock, backend-go/internal/notify, backend-go/internal/profile, backend-go/api/openapi.yaml, backend-go/db/migrations, backend/database/migrations]
skills: [new-endpoint, new-fsd-slice]
boards: [nbl_Priv_Settings.dc.html, nbl_Priv_Lock.dc.html, nbl_Rec_Emergency.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build && cd ../backend-go && go test ./internal/notify/... ./internal/profile/...
---

# CB-PRIV-01 — Discreet notifications + emergency card from the lock screen

## Why
bloom builds the app lock and hide-preview (B-N1-12); the canvas adds discreet notification texts. Icon disguise and hide-in-recents are dropped (native only — DECISIONS).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Priv_Settings.dc.html`
- `nbl_Priv_Lock.dc.html`
- `nbl_Rec_Emergency.dc.html`

## Scope
1. `discreet_notifications` flag: push/SMS texts become neutral ('یادآور امروز') everywhere (notify layer, not per feature); switch in privacy settings per board.
2. Lock screen shows an 'emergency card' link when the user enabled it (CB-REC-03).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Every notification template respects the flag (test); Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
