---
id: B-N5-03
title: Baby logs and pregnancy tools backend — feeding, sleep, diapers, kicks, contractions
milestone: N5
type: backend
status: done
depends_on: [B-N5-01]
parallel_group: N5-C
touches: [backend-go/db,backend-go/internal/babylog,backend-go/internal/pregnancy,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N5-03 — Baby logs and pregnancy tools backend — feeding, sleep, diapers, kicks, contractions

## Why
Timers and counters from the log sheets.

## Scope
- Feeding sessions (breast L/R with durations, bottle ml, pump), baby sleep, diapers; kick-count sessions (time to 10), contraction sessions (start, duration, interval, 5-1-1 alert → message).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Session maths tests
- `verify` green
