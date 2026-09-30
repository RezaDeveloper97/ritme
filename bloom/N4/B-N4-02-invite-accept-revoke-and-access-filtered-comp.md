---
id: B-N4-02
title: Invite, accept, revoke and access-filtered companion APIs
milestone: N4
type: backend
status: todo
depends_on: [B-N4-01]
parallel_group: N4-B
touches: [backend-go/internal/companion,backend-go/internal/care,backend-go/internal/sms,backend-go/internal/http,backend-go/api]
skills: [new-endpoint,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N4-02 — Invite, accept, revoke and access-filtered companion APIs

## Why
Core companion behaviour.

## Scope
- Create invite (SMS via adapter, fake by default), accept by code (signup or later), revoke, change access.
- Companion reads return only granted sections; «ثبت برای …» writes to the owner's meds/appointments only with `edit`.
- Owner notified on accept and on companion writes.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- IDOR / privilege tests for every endpoint
- `verify` green
