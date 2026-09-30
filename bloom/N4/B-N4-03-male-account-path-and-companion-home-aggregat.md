---
id: B-N4-03
title: Male account path and companion home aggregate
milestone: N4
type: backend
status: todo
depends_on: [B-N4-02,B-N2-01]
parallel_group: N4-C
touches: [backend-go/internal/companion,backend-go/internal/profile,backend-go/internal/home,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N4-03 — Male account path and companion home aggregate

## Why
Men sign up only as companions.

## Scope
- gender=male → companion account (no cycle engine). `/api/v1/companion/home`: partner day/phase/next period (if granted), shared meds/appointments, «امروز چه کار کنی؟» tips per phase (admin content), articles, shared child card.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Tests for each access combination
- `verify` green
