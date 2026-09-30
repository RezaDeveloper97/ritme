---
id: B-N3-09
title: Analysis details — cycle, period, symptoms, correlations, body
milestone: N3
type: frontend
status: todo
depends_on: [B-N3-08]
parallel_group: N3-I
touches: [frontend/src/screens/analysis-*,frontend/src/widgets/charts]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-09 — Analysis details — cycle, period, symptoms, correlations, body

## Why
Detail reports.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Cycle.dc.html` (+ `nbd_An_Cycle`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Period.dc.html` (+ `nbd_An_Period`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Symptoms.dc.html` (+ `nbd_An_Symptoms`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Correlations.dc.html` (+ `nbd_An_Correlations`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Body.dc.html` (+ `nbd_An_Body`)

## Scope
- Five screens with their charts and footnotes (FIGO, not-causal).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
