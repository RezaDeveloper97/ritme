---
id: T-M11-22
title: Admin-web family tab and family KPI cards
milestone: M11
type: frontend
status: blocked
depends_on: [T-M11-15, T-M10-35]
parallel_group: M11-W
touches: [admin-web/src/screens/users, admin-web/src/screens/kpi, admin-web/messages/fa.json, admin-web/messages/en.json]
skills: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M11-22 — Admin-web family tab and family KPI cards
**Model:** sonnet.

## Why
SPEC §1.8 and §5.6: the user detail gets a "Family" tab (counts, roles, invite statuses, no grant contents) and the KPI
screen gets the four family KPIs (gap 05 §2.15).

## Scope
1. User detail → «خانواده» tab from `/api/admin/v1/users/{id}/family`; role-gated like the users screen.
2. KPI screen: invite acceptance, members per circle, proposal approval, dose after nudge; suppressed values shown
   as «کمتر از حد نمایش».

## Tests
- Vitest for the view-models; headless screenshots light and dark under `docs/engine-v6/qa/admin/`.

## Out of scope
- Backend.

## Acceptance
- Verify passes; screenshots saved.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.
