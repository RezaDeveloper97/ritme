---
id: T-M10-05
title: Compiled message rules cache with pub/sub invalidation
milestone: M10
type: backend
status: done
depends_on: [T-M10-04]
parallel_group: M10-B
touches: [backend-go/internal/messages/rules, backend-go/internal/messages/compiled, docs/engine-v6/perf-baseline.md]
skills: []
verify: cd backend-go && go vet ./... && go test -race ./internal/messages/... ./internal/platform/configcache/... && go test -run '^$' -bench 'BenchmarkCompile|BenchmarkSelect' -benchtime 2000x ./internal/messages/...
---

# T-M10-05 — Compiled message rules cache with pub/sub invalidation
**Model:** opus.

## Why
Rules change a few times a week; they are read on every home render for up to eight slots. T-M9-13 caches the raw rows
(`configcache`: atomic snapshot, Redis pub/sub `ritme-go:config-changed`, 60 s TTL). Parsing conditions, coercing
values and sorting per request is still wasted work, and the feed needs to know which facts each slot needs before it
builds them (gap 03 §4.1–4.2).

## Scope
1. `internal/messages/compiled`: `Compiled{BySlot map[Slot][]*Rule, NeedFacts map[Slot]facts.Set, Rev uint64,
   Fingerprint string}`; conditions pre-parsed (numbers as float64, enums and sets as small maps or bitsets, dates as
   epoch days), served content resolved (approved snapshot vs head, T-M10-02), rules pre-sorted by the selection order.
2. Compile on every `configcache` reload of the rule rows; serve through `atomic.Pointer`; singleflight; a rule that
   fails to compile is skipped and counted (`messages_rules_compile_errors_total`), the rest of the snapshot is
   served; a whole-snapshot failure keeps the previous snapshot.
3. `messages_rules_rev` gauge; `Rev` returned in feed responses for debugging.
4. `NeedFacts(slots...)` used by the feed to prefetch facts (T-M10-03 `Need`).

## Tests
- Atomic swap while 50 goroutines run `Select` (race detector).
- An admin edit is visible on the next request in the same process, in a second process through pub/sub (miniredis),
  and after the TTL with pub/sub disabled.
- One broken rule row: others served, metric incremented; snapshot-level failure keeps the previous snapshot.
- `NeedFacts` equals the union of facts referenced by the slot's rules (property test over random rules).
- `Select` on compiled rules returns the same winners as T-M10-04's parsed path over 1 000 random rule sets.

## Budgets
- Compile 500 rules ≤ 20 ms; `Select` for one slot over 200 compiled rules ≤ 50 µs and ≤ 2 allocs/op.

## Out of scope
- The feed endpoint (T-M10-06).

## Acceptance
- Verify passes; benchmark numbers recorded in `docs/engine-v6/perf-baseline.md` (append a "messages" row).
