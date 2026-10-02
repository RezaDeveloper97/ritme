---
id: B-N3-06
title: Pregnancy and postpartum log sheets on the new taxonomy
milestone: N3
type: frontend
status: done
depends_on: [B-N3-03]
parallel_group: N3-F
touches: [frontend/src/screens/pregnancy-log,frontend/src/screens/postpartum-log,frontend/src/features/log-day]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-06 — Pregnancy and postpartum log sheets on the new taxonomy

## Why
Mode-specific «امروز چه خبر؟» sheets.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Sheet_Preg.dc.html` (+ `nbd_Log_Sheet_Preg`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Sheet_Post.dc.html` (+ `nbd_Log_Sheet_Post`)

## Scope
- Pregnancy: kicks, contractions (links to B-N5-08), symptom groups, weight, BP (links to vitals B-N6-02), sleep, meds. Postpartum: lochia, pain & stitches, breasts, mood weekly check-in, baby feed/sleep/diapers (B-N5-07 links).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
