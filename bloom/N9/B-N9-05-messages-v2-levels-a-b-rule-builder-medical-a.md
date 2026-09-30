---
id: B-N9-05
title: Messages v2 — levels, A/B, rule builder, medical approval, events
milestone: N9
type: fullstack
status: todo
depends_on: [B-N9-02]
parallel_group: N9-E
touches: [backend-go/internal/messages,backend-go/internal/admin,backend-go/db,backend-go/api,admin-web/src]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-05 — Messages v2 — levels, A/B, rule builder, medical approval, events

## Why
The message editor in the design is far richer than today's.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_Messages.dc.html` (+ `nbd_Admin_Messages`)
- `docs/design/night-bloom/h-admin/nbl_Admin_MessageEditor.dc.html` (+ `nbd_Admin_MessageEditor`)

## Scope
- User levels (info / suggestion / worth follow-up / follow up soon) with required fields; priority bands (safety 1000, engine 800–999, clinical 100–799, product 1–99); variants A/B with stable hash assignment; variables; condition groups (AND/OR) + exceptions; placement, caps, cooldown, hours, start/end, audience %, goal; medical-approval workflow (edit invalidates); preview for a user id; conflict check; message_events (impression/click/dismiss/goal); versioning.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Engine selection tests incl. safety-first and A/B stability
- `verify` green
