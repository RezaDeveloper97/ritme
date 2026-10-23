---
id: T-M9-31
title: Engine state cache v2 (L1 + L2 + singleflight) and ETag/304
milestone: M9
type: backend
status: done
depends_on: [T-M9-09, T-M9-13, T-M9-25, T-M9-30]
parallel_group: M9-C
touches: [backend-go/internal/cycle/state, backend-go/internal/platform/etag, backend-go/internal/auth/passport, backend-go/db/queries/auth, backend-go/internal/platform/config, backend-go/go.mod, backend-go/go.sum]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test -race ./internal/cycle/state/... ./internal/platform/etag/... ./internal/auth/... && make test-int PKG="./internal/cycle/state/... ./internal/auth/..."
---

# T-M9-31 — Engine state cache v2 (L1 + L2 + singleflight) and ETag/304
**Model:** opus.

## Why
The current Redis cycle cache saves CPU, not I/O: its content-hash key needs every input row loaded first, so a hit
costs as much DB time as a miss, and it misses LH, contraception, mode and the clinical config (gap 07 §1, §7.1). With a
DB-authoritative revision read for free in the auth query, a warm engine GET becomes one query, and an unchanged
screen becomes a 304. DECISIONS.md X-02, README §2–§3.

## Scope
1. Auth: extend the T-M9-09 single auth query with `LEFT JOIN user_revisions` → `engine_rev`, `analysis_rev` in the
   request context (0 when no row). Still one query.
2. `internal/cycle/state.Provider.For(ctx, uid, rev, params) (*engine.State, error)`:
   - L1: in-process LRU keyed `(uid, rev, paramsFP, build)`, size from `ENGINE_L1_ENTRIES` (default 20 000), states are
     immutable and shared read-only;
   - L2: Redis `ritme-go:eng:in:{uid}:{rev}` = the versioned `engine.Input` JSON (T-M9-25), TTL 7 d; Redis errors →
     DB;
   - singleflight on `uid:rev` around L2/DB + Prepare;
   - `CYCLE_ENGINE_CACHE=off` bypasses both layers; `ENGINE_CACHE_VERIFY=1` (default on in tests, sampled on stage via
     `ENGINE_CACHE_VERIFY_SAMPLE`) rebuilds from the DB on hits and counts `engine_cache_verify_mismatch_total`;
   - metrics: `cache_requests_total{cache="engine_l1|engine_l2",result}`, `engine_build_seconds`.
   Add `github.com/hashicorp/golang-lru/v2` (or promote an existing LRU) with `go get …@<version>`; promote
   `golang.org/x/sync` to direct.
3. `internal/platform/etag`: helper for engine GETs — `W/"u{uid36}.e{rev}-{localToday}-{locale}-{paramsFP}-{cfgFP}-
   {build}"`, `Cache-Control: private, no-cache`, `Vary: Authorization, Accept-Language`; `If-None-Match` match → 304
   before the handler does any other work. `cfgFP` from `configcache.Fingerprint` (T-M9-13).
4. Not wired into routes here (T-M9-32…35 use it); the v1 cache (`internal/cycle/cache`) stays for `CYCLE_ENGINE=v1`.

## Tests
- Coherence: write (rev bump) → the next `For` misses L1 and L2 and returns the new state; old rev keys are never read.
- Stampede: 50 concurrent `For` calls for one cold user → 1 DB load (query counter), 1 Prepare.
- Redis down → served from DB; L2 corrupt JSON or unknown version → reload.
- Config change (params fingerprint) → re-Prepare from the same L2 snapshot (no DB).
- Verify mode catches an injected stale L1 entry.
- ETag: same inputs → 304 with no body and only the auth query; different user with identical rev/day/locale →
  different tag (privacy); locale change → new tag; Tehran midnight → new tag.
- `-race` clean.

## Budgets
- Warm `For` (L1 hit) ≤ 2 µs; L2 hit + Prepare (24 periods) ≤ 400 µs; 304 path ≤ 5 ms p95 locally with 1 query.

## Out of scope
- Serving v6 payloads (T-M9-32+).

## Acceptance
- Tests green; README §3 unchanged or updated to match the implementation; PROGRESS.md cites D-132.
