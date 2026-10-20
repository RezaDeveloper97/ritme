---
id: T-M10-34
title: Admin-web dashboard alerts, engine health, KPI tiles and OKR status
milestone: M10
type: frontend
status: todo
depends_on: [T-M10-30, T-M10-31]
parallel_group: M10-W
touches: [admin-web/src/screens/dashboard, admin-web/src/screens/kpi, admin-web/src/screens/okr, admin-web/messages/fa.json, admin-web/messages/en.json]
skills: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M10-34 — Admin-web dashboard alerts, engine health, KPI tiles and OKR status
**Model:** sonnet.

## Why
SPEC §5.1 dashboard (alerts, engine health) and §5.6/§5.7 KPIs and OKR need UI for the data T-M10-30 and T-M10-31
produce (gap 06 AW-01, AW-02, AW-04).

## Scope
1. Dashboard: active alerts panel (rule, subject, value vs threshold, opened time, link to the source screen), engine
   health card (MAE and ±2-day accuracy labelled "from real predictions", confidence and data-quality distributions).
2. KPI screen: new tiles (RUI resolved in 7 days, AI cost share, loggers ≥ 15 days, OCR correction, CTR per version),
   suppressed values shown as «کمتر از حد نمایش».
3. OKR: status copy for the new rule; every §5.6 metric bindable to a KR.
4. Alert rule editor (threshold, enabled) for the dashboard manage capability.

## Tests
- Vitest for the view-models (alert rows, suppressed values, OKR status labels); headless screenshots light and dark.

## Out of scope
- Backend.

## Acceptance
- Verify passes; screenshots saved under `docs/engine-v6/qa/admin/`.
