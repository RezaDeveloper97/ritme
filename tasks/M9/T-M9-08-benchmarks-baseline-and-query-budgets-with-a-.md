---
id: T-M9-08
title: Benchmarks, baseline and query budgets with a CI gate
milestone: M9
type: backend
status: done
depends_on: [T-M9-07]
parallel_group: M9-B
touches: [backend-go/internal/benchfixtures, backend-go/internal/querybudget, backend-go/internal/cycle/resolver/resolver_bench_test.go, backend-go/internal/cycle/view/view_bench_test.go, backend-go/internal/cycle/legacy/legacy_bench_test.go, backend-go/internal/cycle/cache/cache_bench_test.go, backend-go/internal/fertility/fertility_bench_test.go, backend-go/internal/analysis/analysis_bench_test.go, backend-go/internal/messages/rules/rules_bench_test.go, backend-go/internal/auth/passport/verify_bench_test.go, backend-go/bench, backend-go/Makefile, .github/workflows/bench.yml, docs/engine-v6/perf-baseline.md]
skills: []
verify: cd backend-go && go vet ./... && go test -run '^$' -bench . -benchtime 200x ./internal/cycle/... ./internal/fertility/... ./internal/analysis/... ./internal/messages/rules/... ./internal/auth/passport/... && make test-int PKG=./internal/querybudget/...
---

# T-M9-08 — Benchmarks, baseline and query budgets with a CI gate
**Model:** opus.

## Why
The repo has zero `Benchmark*` functions, no load scripts and no performance gate (gap 07 §5). Without a recorded
baseline, the engine and cache tasks cannot prove they made anything faster, and regressions go unnoticed.

## Scope
1. `internal/benchfixtures`: deterministic builders for histories (H = 6/24/60/120 periods), 365 days of logs with BBT
   and LH, and users in each mode.
2. Benchmarks (with `b.ReportAllocs()`): `resolver.Resolve`, `metrics.Calculate`, legacy month (`Snapshot.Month`),
   `view.Build`, cache key hashing, fertility insights/BBT chart, analysis summary, `rules.Select` (200 rules), month
   JSON marshal, auth `Verify`.
3. `internal/querybudget`: one integration test per hot route using the T-M9-07 counter — today, date, month, home,
   home cycle-overview, messages daily/selected, fertility insights/bbt, analysis summary/monthly/ttc/pregnancy. The
   budget table starts at today's counts (tests pass); later tasks lower the numbers.
4. `make bench` → `bench/current.txt`; `make bench-compare` → `benchstat bench/baseline.txt bench/current.txt`;
   commit `bench/baseline.txt` from this machine (record CPU/Go version in the file header).
5. `.github/workflows/bench.yml` (push + PR on paths `backend-go/**`): runs the benches with `-count 6`, fails when
   any `allocs/op` regresses, warns (annotation) when `sec/op` regresses > 20 % (DECISIONS.md X-09).
6. `docs/engine-v6/perf-baseline.md`: the baseline table (µs/op, allocs/op, queries per route) + how to re-record.

## Tests
- The benchmarks and the query-budget tests are the deliverable; `go test -bench` must run without failures.
- A unit test for the benchstat-gate script (a fixture with an allocs regression → exit 1; a time-only regression →
  exit 0 + warning).

## Out of scope
- Any optimisation (later tasks), k6 (T-M9-14).

## Acceptance
- Baseline numbers recorded in perf-baseline.md and bench/baseline.txt; CI workflow present and green on the branch;
  query budgets documented per route.
