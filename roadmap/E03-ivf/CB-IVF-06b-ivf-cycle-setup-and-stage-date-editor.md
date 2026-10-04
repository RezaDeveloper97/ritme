---
id: CB-IVF-06b
title: IVF cycle setup and stage-date editor
epic: IVF
type: frontend
status: done
depends_on: [CB-IVF-06]
parallel_group: IVF-E
touches: [frontend/src/screens/ivf,frontend/src/screens/ivf-cycle,frontend/src/screens/ivf-meds,frontend/src/entities/ivf,frontend/src/app/[locale]/ivf,frontend/messages,backend-go/resources/translations,backend-go/internal/i18n/testdata,docs/qa/canvas]
skills: [new-fsd-slice]
boards: [nbl_IVF_Home.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-IVF-06b — IVF cycle setup and stage-date editor

## Why
CB-IVF-06 QA: no screen edits the cycle's stage or its dates (retrieval, transfer, beta, next scan), and starting a cycle always uses defaults — a real user never reaches the TWW countdown. Stimulation meds also stay active after the stage moves on (the med form has no `is_active`).

## Boards
- `nbl_IVF_Home.dc.html` (stage timeline is the entry)

## Scope
1. Cycle setup sheet/page when starting a cycle (protocol, start date, stimulation start) — `POST /ivf/cycles`.
2. Stage + dates editor reachable from the IVF home timeline («مرحله فعلی» / «ویرایش»): stage, next scan, retrieval, transfer, beta → `PUT /ivf/cycles/current` (partial). Dates create/move the linked care appointments (backend already does).
3. Med form: active switch (`is_active`) so stimulation meds can be stopped when moving to the next stage; optional prompt on stage change to stop stimulation meds.
4. Home dose log sends the suggested site (same rule as /ivf/meds).

## Out of scope
- IUI-specific stages; backend changes (none expected).

## Acceptance
- A user can go ttc → IVF on → set up cycle → move stages with dates → reach TWW countdown from the UI only; light + dark screenshots in docs/qa/canvas/ivf.md.
- `verify` green.
