---
id: CB-INS-02
title: admin-web: insurers + centres
epic: INS
type: frontend
status: todo
depends_on: [CB-INS-01, CB-CORE-04]
parallel_group: INS-B
touches: [admin-web/src/screens/insurance-centers, admin-web/src/app/(panel)/insurance-centers, admin-web/src/widgets/shell/model/nav.ts, admin-web/messages]
skills: []
boards: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# CB-INS-02 — admin-web: insurers + centres

## Why
Ours to maintain.

## Scope
1. Centres CRUD (type, lat/lng, direct flag); insurer names via catalog.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- CRUD tested.
- `verify` green.
