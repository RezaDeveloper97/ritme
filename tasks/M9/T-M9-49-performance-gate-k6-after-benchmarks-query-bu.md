---
id: T-M9-49
title: Performance gate: k6 after, benchmarks, query budgets vs baseline
milestone: M9
type: investigate
status: done
depends_on: [T-M9-48]
parallel_group: M9-R
touches: [docs/engine-v6/perf-baseline.md, backend-go/bench, backend-go/internal/querybudget]
skills: [local-dev]
verify: grep -q '## After' docs/engine-v6/perf-baseline.md
---

# T-M9-49 — Performance gate: k6 after, benchmarks, query budgets vs baseline
**Model:** main session; opus subagent for any regression fix.

## Why
The user asked for an engine that answers very fast; README §8 sets the budgets. This task proves them (or blocks beta).

## Scope
1. Locally (docker MariaDB/Redis, `CYCLE_ENGINE=v6`, worker on): re-run the T-M9-14 k6 scenarios with the same seed and
   record the "After" table (p50/p95/p99, error rate, queries per request) next to "Before".
2. `make bench-compare`: no allocs/op regression; engine benches under README §8 ceilings; lock ceilings at measured ×
   1.2 in `bench/baseline.txt`.
3. `internal/querybudget`: budgets lowered to the README §8 targets and green.
4. `log_burst`: zero stale reads.
5. Any miss → a fix task (opus) before T-M9-50; record decisions in perf-baseline.md.

## Acceptance
- Every README §8 row met (or an accepted, documented exception approved by the user); the "After" table committed.
