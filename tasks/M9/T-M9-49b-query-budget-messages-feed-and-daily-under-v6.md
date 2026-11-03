---
id: T-M9-49b
title: Query budget: messages feed and daily under v6
milestone: M9
type: backend
status: done
depends_on: []
parallel_group: M9-R
touches: [backend-go/internal/messages,backend-go/db/queries/messages,backend-go/internal/querybudget]
skills: []
verify: cd backend-go && CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 make test-int PKG=./internal/messages/... && CGO_ENABLED=0 make test-int PKG=./internal/querybudget/...
---

# T-M9-49b — Query budget: messages feed and daily under v6

## Why
T-M9-49 perf gate (docs/engine-v6/perf-baseline.md, After): under CYCLE_ENGINE=v6 `/messages/feed` runs warm 7 queries (budget 4) and cold 16 (budget 13); `/messages/daily` warm 8 (budget ≤ 4). Likely added by T-M10-06/09/09b/10 (module reads, inbox sweep, facts). Blocks T-M9-50.

## Scope
- Profile the queries of feed and daily under v6 (querybudget test output); remove the extra round trips (batch module/inbox reads, reuse already-loaded facts, cache per request, skip module reads for users not in that mode).
- Keep behaviour identical: messages, feed and modulefacts int tests green.
- Do not raise budgets.

## Acceptance
- querybudget feed v6 warm ≤ 4, cold ≤ 13; daily v6 warm ≤ 4.
- verify green.
