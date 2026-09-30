---
id: CB-CORE-02
title: Extra UI primitives missing from bloom's set
epic: CORE
type: frontend
status: todo
depends_on: [CB-CORE-01, B-N1-03]
parallel_group: CORE-B
touches: [frontend/src/shared/ui]
skills: [new-fsd-slice]
boards: [nbl_Meno_Log.dc.html, nbl_IVF_Home.dc.html, nbl_Cond_Endo.dc.html, nbl_Cond_PMDD.dc.html, nbl_Pelvic_Kegel.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-CORE-02 — Extra UI primitives missing from bloom's set

## Why
Build once what many canvas-v1 boards repeat and B-N1-03 does not provide.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Log.dc.html`
- `nbl_IVF_Home.dc.html`
- `nbl_Cond_Endo.dc.html`
- `nbl_Cond_PMDD.dc.html`
- `nbl_Pelvic_Kegel.dc.html`

## Scope
1. Only the primitives CB-CORE-01 lists as missing — expected: SeverityScale (none/mild/moderate/severe), NumericScale (0–10 and 1–6 with end labels), StepTimeline (done/current/todo + dates), CountdownRing (timers), DangerNote.
2. Accessible (radios/buttons, ≥44px), RTL, N&B tokens light+dark, unit tests, ui-kit entries next to bloom's.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Primitives exported with tests and ui-kit entries.
- `verify` green.
