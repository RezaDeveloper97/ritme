---
id: CB-CORE-04
title: admin-web: catalog editor
epic: CORE
type: frontend
status: todo
depends_on: [CB-CORE-03]
parallel_group: CORE-C
touches: [admin-web/src/screens/catalog, admin-web/src/app/(panel)/catalog, admin-web/src/widgets/shell/model/nav.ts, admin-web/messages]
skills: []
boards: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# CB-CORE-04 — admin-web: catalog editor

## Why
One screen for every catalog group.

## Scope
1. Group picker, ordered list with active toggle, fa/en editor, per-group meta hints, fa digits; nav entry with the content permission (B-N9-02 roles when present).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- CRUD + reorder tested.
- `verify` green.
