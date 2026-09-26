---
id: T-M7-14
title: Frontend — pregnancy «هشدارها» (Alerts) v2 screen
milestone: M7
type: frontend
status: done
depends_on: [T-M7-08, T-M7-04]
parallel_group: M7-D
touches: [frontend/src/screens/pregnancy-alerts, frontend/src/app/[locale]/pregnancy/alerts]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M7-14 — Frontend — pregnancy Alerts v2

## Scope
`/pregnancy/alerts` from `Alerts.dc.html`: subtitle window; cards grouped by date with level chip (4 levels,
token colors per level, urgent shows the admin-defined contact line/phone link), «چی دیدیم», «چقدر مطمئنیم»,
advice, actions (add to visit note, ack, deep links like «ثبت وزن» → log focus weight); level legend from the API;
disclaimer + link to notification settings. Marks shown alerts read.

## Acceptance
- Light/dark × fa/en screenshots for every level; build green.
