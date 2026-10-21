---
id: T-M9-22
title: Engine v6 hardening: invariants, fuzz, goldens, benchmarks, 95 % coverage
milestone: M9
type: backend
status: done
depends_on: [T-M9-21]
parallel_group: M9-B
touches: [backend-go/internal/cycle/engine, backend-go/bench, docs/engine-v6/engine.md, docs/engine-v6/perf-baseline.md]
skills: []
verify: cd backend-go && go vet ./internal/cycle/engine/... && go test -race ./internal/cycle/engine/... && go test -run '^$' -fuzz FuzzPrepare -fuzztime 30s ./internal/cycle/engine && c=$(mktemp) && go test -coverprofile=$c ./internal/cycle/engine/... && go tool cover -func=$c | awk '/^total:/ {exit ($3+0 < 95)}' && go test -run '^$' -bench . -benchmem -benchtime 2000x ./internal/cycle/engine/...
---

# T-M9-22 — Engine v6 hardening: invariants, fuzz, goldens, benchmarks, 95 % coverage
**Model:** opus.

## Why
The engine becomes the single source for every cycle consumer; it must not panic on bad data, must keep its
invariants on any history, and must stay fast. This task locks that in before anything is served from it (README §9).

## Scope
1. Invariant suite (seeded `math/rand/v2`, 2 000 random histories × 400 days, including skips, BBT, LH, pill,
   postpartum, ages 14–55): `cycle_day ≥ 1` whenever a start exists; never fertile on a menstrual day; O+1 never
   fertile; every start/end has a source; `fertility_level = unknown` iff period_expected/unknown/suppressed;
   `days_to_ovulation` null rules; `requires_user_input` ⇔ one of the six warnings; future cycles never overlap and
   are increasing; `Range == map(Resolve)`; `Resolve` is deterministic (same input → same bytes).
2. Fuzz targets `FuzzPrepare` (unsorted, overlapping, duplicated, open, absurd dates within ±100 years) and
   `FuzzBBTDetect`: no panic, invariants hold; seed corpus committed.
3. Engine goldens: ~20 synthetic personas in `testdata/personas/*.json` (`engine.Input` + today): regular 28, 21, 35,
   irregular, teen 16, age 46, pill, copper IUD, postpartum breastfeeding, pregnant, onboarding-only fresh, onboarding
   stale (90 days), skip, BBT strict pass, BBT strict fail, LH/BBT conflict, open period past the hard cap, late 8,
   late 15, contradictory data. Each golden = one year of `Range` + `FutureCycles` + `TodayUI`, regenerated with
   `go test ./internal/cycle/engine -run TestGoldens -update`; the 21/28/35 and §2.35 anchors are checked by hand in the
   test, not only by the golden.
4. Benchmarks locked (README §8): Prepare (60 periods + 400 markers) ≤ 300 µs, Resolve ≤ 3 µs / ≤ 3 allocs, Range31
   ≤ 150 µs / ≤ 40 allocs, EncodeDay ≤ 20 µs; add them to `bench/baseline.txt` with the T-M9-08 tooling (if T-M9-08 is
   not done yet, record them in `docs/engine-v6/perf-baseline.md` and note it).
5. Coverage ≥ 95 % for `internal/cycle/engine/...`.
6. `docs/engine-v6/engine.md`: the API, the decisions each rule implements (E-ids), how to add a rule.

## Tests
- The suites above are the deliverable.

## Out of scope
- Serving, inputs, caching.

## Acceptance
- Verify passes (fuzz 30 s, coverage ≥ 95 %, benches under budget on this machine); engine.md written.
