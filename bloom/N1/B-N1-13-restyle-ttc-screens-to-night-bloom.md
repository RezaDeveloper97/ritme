---
id: B-N1-13
title: Restyle TTC screens to Night & Bloom
milestone: N1
type: frontend
status: todo
depends_on: [B-N1-03]
parallel_group: N1-M
touches: [frontend/src/screens/fertility-log,frontend/src/screens/fertility-bbt,frontend/src/screens/fertility-insights,frontend/src/widgets/fertility-tiles,frontend/src/widgets/bbt-chart]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-13 — Restyle TTC screens to Night & Bloom

## Why
M5 shipped TTC from v19; the canvas re-skins it (nb2 = dark, v19 = light).

## Design
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nb2_Main.dc.html`
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/v19_Main.dc.html`
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nb2_TTC_Log.dc.html`
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/v19_TTC_Log.dc.html`
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nb2_TTC_BBT.dc.html`
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/v19_TTC_BBT.dc.html`
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nb2_TTC_Insights.dc.html`
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/v19_TTC_Insights.dc.html`

## Scope
- TTC today tiles, day log, BBT chart (3/6-cycle tabs, baseline, fertile band), insights (evidence list, past ovulation days).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
