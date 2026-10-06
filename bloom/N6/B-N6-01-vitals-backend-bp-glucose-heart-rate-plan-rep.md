---
id: B-N6-01
title: Vitals backend — BP, glucose, heart rate, plan, reports, safety
milestone: N6
type: backend
status: done
depends_on: [B-N5-11]
parallel_group: N6-A
touches: [backend-go/db,backend-go/internal/vitals,backend-go/internal/messages,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N6-01 — Vitals backend — BP, glucose, heart rate, plan, reports, safety

## Why
Vitals exist only inside pregnancy today.

## Scope
- Readings: BP (sys/dia/pulse, arm, position), glucose (mg/dL|mmol/L, context, method), HR (context).
- Classification (ACC/AHA BP, ADA glucose) from a thresholds table (moved to versioned config in B-N9-11).
- Weekly measurement plan; reports (7/30/90d avg, min/max, distribution, morning vs night, time in range).
- Safety: BP >180/120 or glucose <54 → urgent modal message. Decide & document whether pregnancy BP/HR migrates in.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Classification + safety tests
- `verify` green
