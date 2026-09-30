---
id: CB-CORE-06
title: Neshan map wrapper
epic: CORE
type: frontend
status: done
depends_on: [CB-CORE-01]
parallel_group: CORE-B
touches: [frontend/src/shared/ui/map, frontend/src/shared/config, docs/canvas-build/map.md]
skills: [new-fsd-slice]
boards: [nbl_Dir_Map.dc.html, nbl_Ins_Centers.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-CORE-06 — Neshan map wrapper

## Why
Directory and insurance centres need a map (DECISIONS: Neshan).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Map.dc.html`
- `nbl_Ins_Centers.dc.html`

## Scope
1. Lazy-loaded Neshan web map (key `NEXT_PUBLIC_NESHAN_KEY`): pins, selected card slot, 'search this area' bbox callback, my-location; N&B light/dark map style if available.
2. No key → renders nothing and callers show the list (flag). Doc env + CSP in docs/canvas-build/map.md.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Fallback unit-tested.
- `verify` green.
