---
id: B-N1-07
title: Calendar redesign (cycle + TTC variants)
milestone: N1
type: frontend
status: done
depends_on: [B-N1-04]
parallel_group: N1-G
touches: [frontend/src/screens/calendar,frontend/src/widgets/cycle-calendar]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-07 — Calendar redesign (cycle + TTC variants)

## Why
The calendar gets month/year views, PMS legend and a day-detail card.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Calendar.dc.html` (+ `nbd_Cycle_Calendar`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nb2_TTC_Calendar.dc.html`
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/v19_TTC_Calendar.dc.html`

## Scope
- Month ⇄ year toggle, two stacked months, legend (period, predicted, fertile, ovulation, PMS), tap a day → detail card (cycle day, phase, logged items, «ثبت جزئیات»). TTC mode uses the fertility variant (chance of pregnancy).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
