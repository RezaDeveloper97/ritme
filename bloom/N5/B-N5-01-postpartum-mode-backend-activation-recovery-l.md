---
id: B-N5-01
title: Postpartum mode backend — activation, recovery log, EPDS
milestone: N5
type: backend
status: done
depends_on: [B-N4-10]
parallel_group: N5-A
touches: [backend-go/db,backend-go/internal/postpartum,backend-go/internal/messages,backend-go/internal/pregnancy,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N5-01 — Postpartum mode backend — activation, recovery log, EPDS

## Why
Postpartum has only an enum today.

## Scope
- Activate from pregnancy (birth date, delivery type) or directly; weeks since birth; recovery log (lochia amount/colour, pain & location, breasts, sleep, feeds count); EPDS: weekly 3-question check + full 10-question every 2 weeks, Q10 positive or score ≥13 → urgent safety message with call action; tips/alerts through the message engine.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- EPDS scoring + safety trigger tests
- `verify` green
