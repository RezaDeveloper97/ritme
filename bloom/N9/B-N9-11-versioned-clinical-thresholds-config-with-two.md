---
id: B-N9-11
title: Versioned clinical thresholds & config with two approvals
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-02]
parallel_group: N9-K
touches: [backend-go/internal/config,backend-go/internal/vitals,backend-go/internal/postpartum,backend-go/internal/cycle,backend-go/db,backend-go/api,admin-web/src]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-11 — Versioned clinical thresholds & config with two approvals

## Why
Clinical constants must be auditable.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_Config.dc.html` (+ `nbd_Admin_Config`)

## Scope
- Keys with value, unit, source, version, effective-from, approvers; change proposal → 2 medical approvals → effective date; server reads the version valid at `target_date`; migrate hard-coded constants (BP, glucose, EPDS, FIGO, engine caps, fertile window, ferritin) to it.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Historic outputs reproducible with old versions (test)
- `verify` green
