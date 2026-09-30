---
id: B-N1-08
title: Edit period, cycle history, symptom pattern and phase sheet
milestone: N1
type: fullstack
status: todo
depends_on: [B-N1-06]
parallel_group: N1-H
touches: [frontend/src/screens/cycle,frontend/src/screens/phase-details,frontend/src/features/log-period,backend-go/internal/cycle,backend-go/internal/healthlog,backend-go/api]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-08 — Edit period, cycle history, symptom pattern and phase sheet

## Why
Cycle management screens from the new design.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_EditPeriod.dc.html` (+ `nbd_Cycle_EditPeriod`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_History.dc.html` (+ `nbd_Cycle_History`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Symptoms.dc.html` (+ `nbd_Cycle_Symptoms`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Phase.dc.html` (+ `nbd_Cycle_Phase`)

## Scope
- Edit period: multi-day bleeding selection on a month grid with suggested dashed days, «فقط لکه‌بینی است».
- Cycle history (`/cycle`): median length, period length, variability, regularity verdict, per-cycle bars with out-of-range marking, «افزودن سیکل‌های قبلی».
- Symptom pattern screen: per-symptom heat strip over the typical cycle (needs ≥3 cycles) — backend endpoint.
- Phase sheet with tabs بدن / حال / تغذیه / حرکت / رابطه (content from phase-contents).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
