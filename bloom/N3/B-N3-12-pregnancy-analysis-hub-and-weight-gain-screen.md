---
id: B-N3-12
title: Pregnancy analysis hub and weight-gain screen
milestone: N3
type: fullstack
status: done
depends_on: [B-N3-08]
parallel_group: N3-L
touches: [frontend/src/screens/analysis-pregnancy,backend-go/internal/pregnancy,backend-go/internal/analysis,backend-go/api]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-12 — Pregnancy analysis hub and weight-gain screen

## Why
Pregnancy analysis.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Hub_Preg.dc.html` (+ `nbd_An_Hub_Preg`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_PregWeight.dc.html` (+ `nbd_An_PregWeight`)

## Scope
- Weight gain vs IOM band by pre-pregnancy BMI, BP vs 140/90, GDM glucose targets (Plus), kicks, symptoms by trimester, visits summary. Thresholds read from config (constants until B-N9-11).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
