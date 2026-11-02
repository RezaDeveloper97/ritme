---
id: T-M10-35
title: Admin-web users, segments, roles and audit chain
milestone: M10
type: frontend
status: done
depends_on: [T-M10-32, T-M10-33, T-M10-13]
parallel_group: M10-W
touches: [admin-web/src/screens/users, admin-web/src/screens/segments, admin-web/src/screens/access-control, admin-web/src/widgets/shell/model/nav.ts, admin-web/messages/fa.json, admin-web/messages/en.json]
skills: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M10-35 — Admin-web users, segments, roles and audit chain
**Model:** sonnet; the main session reviews what a role can see.

## Why
SPEC §5.2 user monitoring and §5.8 roles and audit need UI for T-M10-32 and T-M10-33 (gap 06 AW-03, AW-05).

## Scope
1. Users list: RT code, masked mobile, mode, plan, last activity, log days in 30, data_quality, main engine warning,
   churn; filters; "save as segment".
2. Segments screen (list, edit filters, delete) and the segment picker in the message rule editor's audience section.
3. User detail: engine card (today's output), cycle-length history chart, recent messages and challenges with the rule
   and version, health sections behind the existing grant prompt.
4. Roles screen shows the spec matrix and marks customised roles; audit screen shows the chain status and a verify
   button (super).

## Tests
- Vitest for filters ↔ query, segment save, role-gated rendering; headless screenshots light and dark.

## Out of scope
- Backend.

## Acceptance
- Verify passes; screenshots saved under `docs/engine-v6/qa/admin/`.
