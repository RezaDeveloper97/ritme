---
id: CB-MENO-04
title: admin-web: menopause content
epic: MENO
type: frontend
status: todo
depends_on: [CB-CORE-04, CB-MENO-01]
parallel_group: MENO-C
touches: [admin-web/src/screens/catalog, admin-web/src/screens/checkup-types, admin-web/messages]
skills: []
boards: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# CB-MENO-04 — admin-web: menopause content

## Why
Clinical copy editable without deploys.

## Scope
1. meno_* catalog hints; audience=menopause filter in checkup-types; taxonomy items editable where bloom's admin exposes the taxonomy.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Admin edits every menopause list.
- `verify` green.
