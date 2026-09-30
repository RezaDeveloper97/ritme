---
id: B-N3-02
title: Log preferences — quick tiles, order, custom items
milestone: N3
type: backend
status: todo
depends_on: [B-N3-01]
parallel_group: N3-B
touches: [backend-go/internal/healthlog,backend-go/db,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N3-02 — Log preferences — quick tiles, order, custom items

## Why
Personalised log sheet.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Customize.dc.html` (+ `nbd_Log_Customize`)

## Scope
- Per user: ordered categories, visibility, ≤8 pinned quick tiles (default by mode/phase), custom items.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Defaults per mode tested
- `verify` green
