---
id: T-M10-28
title: Analysis cache on analysis_rev with ETag and a nightly warm job
milestone: M10
type: backend
status: todo
depends_on: [T-M10-26, T-M10-27, T-M9-30, T-M9-31, T-M9-12]
parallel_group: M10-D
touches: [backend-go/internal/analysis/cache.go, backend-go/internal/analysis/handlers.go, backend-go/internal/analysis/pregnancy_handlers.go, backend-go/internal/analysis/ttc_handlers.go, backend-go/internal/analysis/postpartum_handlers.go, backend-go/internal/cycle/revision, backend-go/internal/vitals/service.go, backend-go/internal/labs/service.go, backend-go/internal/postpartum, backend-go/internal/babylog, backend-go/internal/pregnancy, backend-go/internal/healthlog, docs/engine-v6/jobs.md]
skills: []
verify: cd backend-go && go vet ./... && go test -race ./internal/analysis/... ./internal/cycle/revision/... && make test-int PKG="./internal/analysis/... ./internal/cycle/revision/..." && go test -run '^$' -bench BenchmarkAnalysis -benchtime 200x ./internal/analysis/...
---

# T-M10-28 — Analysis cache on analysis_rev with ETag and a nightly warm job
**Model:** opus.

## Why
Every analysis request runs a Plus check, loads all cycle histories, reads the health log from 40 days before the
range, and recomputes everything; the hub recomputes four analyses (gap 06 §2.1, §2.8). SPEC §4.1: compute in a
nightly job or when the tab opens if data changed; editing the past invalidates. `user_revisions.analysis_rev`
(T-M9-24) is bumped by engine-input writes (T-M9-30) but not by vitals, labs, EPDS, baby logs or pregnancy logs.
DECISIONS.md X-02; D-177.

## Scope
1. `analysis_rev` bumps inside the write transaction of every analysis-only input (vitals readings, lab marker
   confirm/edit, EPDS, postpartum recovery, baby logs, pregnancy weekly logs, weight) through the revision package;
   extend the T-M9-30 manifest test so a new write query on these tables without a bump fails the build.
2. Cache-aside per endpoint: Redis `ritme-go:ana:{uid}:{analysis_rev}:{engine_rev}:{endpoint}:{range}:{plus}:{localToday}:{cfgFP}`
   (+ locale only if the payload is localised during computation; prefer caching the data before localisation),
   24 h TTL, singleflight per key, Redis errors fall back to computing.
3. ETag `W/"u{uid36}.a{analysis_rev}.e{engine_rev}-{localToday}-{endpoint}-{range}-{plus}-{locale}-{cfgFP}-{build}"`,
   `Cache-Control: private, no-cache`, `Vary: Authorization, Accept-Language`; 304 on match (the T-M9-31 `etag`
   package).
4. Job `analysis.warm` (worker, queue `low`, nightly, unique per day): users active in the last 7 days, summary and the
   default range, batches of 200, skipped when already cached.

## Tests
- miniredis hit/miss; a past vitals edit invalidates (rev bump); Plus purchase invalidates (key); a registry change
  invalidates (`cfgFP`); 304 path; two accounts never share a body (uid in key and tag).
- Manifest test fails on a fixture write query without a bump.
- Job idempotent and skips cached users.

## Budgets
- Summary hit ≤ 15 ms p95 and no queries beyond auth; 304 ≤ 5 ms; cold summary ≤ 150 ms p95 (bench recorded in
  perf-baseline.md).

## Out of scope
- New analyses.

## Acceptance
- Verify passes; PROGRESS.md cites D-177.
