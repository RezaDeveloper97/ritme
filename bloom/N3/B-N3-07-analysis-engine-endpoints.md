---
id: B-N3-07
title: Analysis engine endpoints
milestone: N3
type: backend
status: done
depends_on: [B-N3-01]
parallel_group: N3-G
touches: [backend-go/internal/analysis,backend-go/internal/http,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N3-07 — Analysis engine endpoints

## Why
The «تحلیل» screens need computed, explainable stats.

## Scope
- `/api/v1/analysis/{summary,cycle,period,symptoms,correlations,body,monthly/:ym}` with range filter (3m/6m/1y/all).
- FIGO ranges, min-data rules (patterns need ≥3 cycles; trends ≥2 points), top finding sentence, correlations with strength (strong/medium/weak) and a `not_causal` flag, 7-day moving-average weight.
- Plus-gated sections marked in the payload; free users get the summary.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Golden tests on seeded histories
- `verify` green
