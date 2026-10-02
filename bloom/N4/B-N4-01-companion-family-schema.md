---
id: B-N4-01
title: Companion & family schema
milestone: N4
type: backend
status: done
depends_on: [B-N3-14]
parallel_group: N4-A
touches: [backend-go/db,backend-go/internal/companion]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N4-01 — Companion & family schema

## Why
«همدم» lets a partner or spouse see and help.

## Scope
- companions (owner, companion user, type partner|spouse, status invited|active|revoked), invites (phone or 6-char code, one-time, 24h), access grants per section (cycle, symptoms, meds, appointments, pregnancy × none|view|edit), families (owner + spouse) and shared children link (children table arrives in B-N5-02 — FK added there), audit log.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Migrations up/down tested
- `verify` green
