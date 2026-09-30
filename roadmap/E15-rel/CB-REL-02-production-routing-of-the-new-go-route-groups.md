---
id: CB-REL-02
title: Production routing of the new Go route groups
epic: REL
type: release
status: todo
depends_on: [CB-REL-01]
parallel_group: REL-B
touches: [deploy/go-routes.inc, docs/go-migration/prod-cutover-log.md]
skills: [deploy, verify-all]
boards: []
verify: test -s docs/go-migration/prod-cutover-log.md
---

# CB-REL-02 — Production routing of the new Go route groups

## Why
New Go-only groups (catalog, search, menopause, ivf, loss, conditions, contraception, pelvic, teen, insurance, city services, shop extensions) must be routed on prod. **Only when the user explicitly asks — never on your own.**

## Scope
1. Add route groups to deploy/go-routes.inc (after T-M2-26 rules), deploy, smoke, log + rollback.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Prod smoke green; rollback documented.
- `verify` green.
