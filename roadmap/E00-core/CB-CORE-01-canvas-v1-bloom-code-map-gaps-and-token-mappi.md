---
id: CB-CORE-01
title: Canvas-v1 → bloom → code map, gaps and token mapping
epic: CORE
type: investigate
status: done
depends_on: [B-N1-01]
parallel_group: CORE-A
touches: [docs/canvas-build]
skills: []
boards: [IA_Map.dc.html, IA_Nav.dc.html]
verify: test -s docs/canvas-build/README.md
---

# CB-CORE-01 — Canvas-v1 → bloom → code map, gaps and token mapping

## Why
The menopause canvas overlaps the Night & Bloom canvas (bloom/). Every CB task must know what bloom already builds so nothing is done twice.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `IA_Map.dc.html`
- `IA_Nav.dc.html`

## Scope
1. For every board in docs/design/canvas-v1/README.md: the bloom task that delivers it fully/partly (B-N*), the existing route/entity, or NEW. Flag conflicts (e.g. shop link-out vs internal COD shop) with the decision from roadmap/DECISIONS.md.
2. Token map: board hexes → Night & Bloom tokens from docs/night-bloom/README.md (light + dark). Both canvases share the nbl_/nbd_ language — no new palette.
3. Primitive gap list vs B-N1-03 (severity scale, 0–10 / 1–6 scales, step timeline, countdown ring, danger note…).
4. Gap list for things without design or backend (assistant, doctors, courses, Plus rows…) and which bloom milestone covers them.
5. Update CB task Scopes/deps where the map changes them (note it in roadmap/PROGRESS.md).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/canvas-build/README.md with board map, bloom overlap table, token map, primitive + gap lists.
- `verify` green.
