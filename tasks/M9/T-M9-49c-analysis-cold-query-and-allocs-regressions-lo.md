---
id: T-M9-49c
title: Analysis cold query and allocs regressions, lock bench baseline
milestone: M9
type: backend
status: done
depends_on: []
parallel_group: M9-R
touches: [backend-go/internal/analysis,backend-go/internal/cycle/cache,backend-go/db/queries/analysis,backend-go/bench,backend-go/internal/querybudget]
skills: []
verify: cd backend-go && CGO_ENABLED=0 make test-int PKG=./internal/querybudget/... && CGO_ENABLED=0 make bench-compare
---

# T-M9-49c — Analysis cold query and allocs regressions, lock bench baseline

## Why
T-M9-49 perf gate (perf-baseline.md, After): querybudget cold +1 query on analysis summary (15>14), ttc (16>15), pregnancy (19>18); `make bench-compare` shows 11 allocs/op regressions: analysis.BenchmarkSummary/{1m,3m,1y} +10–17 % (likely T-M10-28 cache/ETag) and cycle/cache.BenchmarkKeyString/* +1 alloc. Blocks T-M9-50.

## Scope
- Remove the extra cold query on the analysis routes (or, if it is the new cache/revision read by design, document why and get the user's OK before raising the budget).
- Remove the allocs regressions in analysis.Summary and cache KeyString.
- Then `make bench-baseline` (adds the 7 new benches) and lock ceilings at measured × 1.2 in bench/baseline.txt.

## Acceptance
- querybudget analysis rows green; bench-compare 0 allocs regressions; baseline locked.
- verify green.
