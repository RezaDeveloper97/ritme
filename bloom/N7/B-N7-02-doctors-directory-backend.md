---
id: B-N7-02
title: Doctors directory backend
milestone: N7
type: backend
status: todo
depends_on: [B-N7-01]
parallel_group: N7-B
touches: [backend-go/db,backend-go/internal/telemed,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N7-02 — Doctors directory backend

## Why
Telemedicine marketplace.

## Scope
- Doctors/midwives (admin-managed profile, licence no., specialty, bio), visit types video/phone/in-person with price and duration, availability slots, reviews & rating, filters (type, today, specialty, city, insurance).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Slot generation tests
- `verify` green
