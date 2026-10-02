---
id: B-N3-11
title: TTC analysis hub and BBT/ovulation confirmation
milestone: N3
type: fullstack
status: done
depends_on: [B-N3-08]
parallel_group: N3-K
touches: [frontend/src/screens/analysis-ttc,backend-go/internal/fertility,backend-go/internal/analysis,backend-go/api]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-11 — TTC analysis hub and BBT/ovulation confirmation

## Why
TTC-specific analysis.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Hub_TTC.dc.html` (+ `nbd_An_Hub_TTC`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Fertility.dc.html` (+ `nbd_An_Fertility`)

## Scope
- Cycles trying, three-over-six BBT confirmation, LH positive day, intercourse timing (Plus), cervical mucus pattern (Plus), luteal length, regularity; referral advice by age (<35: 12 months, ≥35: 6 months).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
