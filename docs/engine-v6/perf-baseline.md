# Engine v6 performance baseline (T-M9-08)

> Recorded 2026-10-21 03:13 UTC on Apple M4 (10 cores, 16 GB, darwin/arm64), go1.27.1, from the working tree at `60a451c3 (dirty tree)`. Load average at start:
> 26 / 38 / 45 (1 / 5 / 15 min) — other agents were building and testing at the same time, so **read µs/op as an order of magnitude**,
> not a precise number (the *spread* column is half the min–max range of the 6 samples). `allocs/op` is not
> affected by load: it is the number the CI gate holds.
>
> Raw data: [`backend-go/bench/baseline.txt`](../../backend-go/bench/baseline.txt). Older measurements (Laravel
> stack, T-M1-10, and the T-M9-09 auth numbers) are in [`docs/investigations/perf-baseline.md`](../investigations/perf-baseline.md).
> Targets: [README §8](README.md#8-budgets). Policy: [DECISIONS.md X-09](DECISIONS.md).

## 1. Fixtures

`backend-go/internal/benchfixtures` builds everything from a fixed day (**2026-10-21**) and a seeded PCG stream, so
every run benchmarks the same data:

- **Histories** of H = 6 / 24 / 60 / 120 confirmed, closed periods, newest first; cycle day 10 today; cycles 26–32
  days with one in nine 35–38 (outlier); bleeds 4–6 days. `Periods(n)` is a prefix of `Periods(m)`.
- **365 days of logs**: flow and cramps on bleed days, a biphasic BBT curve (≈ 36.40 → 36.85 after ovulation, one
  day in seven unmeasured), LH negative / faint / positive around ovulation, mucus, sleep, mood, energy, weight.
- **One user per life-stage mode** (`Users()`: cycle, ttc, pregnancy, postpartum, menopause, teen).
- `benchfixtures/cyclesnap` turns them into `cycle/service.Snapshot`s (the per-request engine input) without a DB.

## 2. Benchmarks (µs/op, allocs/op)

| Package | Benchmark | µs/op (median) | spread | B/op | allocs/op | allocs stable |
|---|---|---:|---:|---:|---:|:-:|
| cycle/cache | `KeyString/day/H=6` | 180.11 | ±74% | 7033 | 57 | ✔ |
| cycle/cache | `KeyString/month/H=6` | 1773.90 | ±80% | 68576 | 165 | ✔ |
| cycle/cache | `KeyString/day/H=24` | 474.90 | ±26% | 15658 | 147 | ✔ |
| cycle/cache | `KeyString/month/H=24` | 1899.89 | ±30% | 78228 | 255 | ✔ |
| cycle/cache | `KeyString/day/H=60` | 856.55 | ±29% | 31850 | 327 | ✔ |
| cycle/cache | `KeyString/month/H=60` | 2197.15 | ±21% | 97768 | 435 | ✔ |
| cycle/cache | `KeyString/day/H=120` | 961.00 | ±39% | 66230 | 627 | ✔ |
| cycle/cache | `KeyString/month/H=120` | 2587.69 | ±80% | 128088 | 735 | ✔ |
| cycle/engine | `EncodeDay` | 7.73 | ±16% | 0 | 0 | ✔ |
| cycle/engine | `TodayPayload` | 14.28 | ±62% | 192 | 1 | ✔ |
| cycle/engine | `Range31` | 12.60 | ±36% | 6528 | 1 | ✔ |
| cycle/engine | `FutureCycles` | 0.27 | ±8% | 192 | 1 | ✔ |
| cycle/engine | `Resolve_Hormonal` | 0.23 | ±49% | 0 | 0 | ✔ |
| cycle/engine | `Prepare_Markers` | 36.04 | ±17% | 8952 | 10 | ✔ |
| cycle/engine | `Prepare` | 4.07 | ±22% | 5272 | 8 | ✔ |
| cycle/engine | `Resolve` | 0.40 | ±12% | 0 | 0 | ✔ |
| cycle/engine | `TodayUI` | 0.02 | ±8% | 0 | 0 | ✔ |
| cycle/legacy | `SnapshotMonth/calendar/H=6` | 395.74 | ±26% | 334471 | 1159 | ✔ |
| cycle/legacy | `SnapshotMonth/calendar/H=24` | 430.43 | ±26% | 342308 | 1181 | ✔ |
| cycle/legacy | `SnapshotMonth/calendar/H=60` | 577.16 | ±27% | 358992 | 1219 | ✔ |
| cycle/legacy | `SnapshotMonth/calendar/H=120` | 643.33 | ±14% | 386118 | 1281 | ✔ |
| cycle/legacy | `SnapshotMonth/full/H=6` | 510.80 | ±22% | 361930 | 1526 | ✔ |
| cycle/legacy | `SnapshotMonth/full/H=24` | 505.31 | ±11% | 369750 | 1548 | ✔ |
| cycle/legacy | `SnapshotMonth/full/H=60` | 639.99 | ±23% | 386429 | 1586 | ✔ |
| cycle/legacy | `SnapshotMonth/full/H=120` | 654.15 | ±24% | 413564 | 1648 | ✔ |
| cycle/legacy | `MonthJSON/calendar` | 621.37 | ±35% | 310166 | 3844 | ✔ |
| cycle/legacy | `MonthJSON/full` | 6500.51 | ±11% | 2657358 | 26958 | 26956–26959 |
| cycle/resolver | `Resolve/H=6` | 1.92 | ±11% | 80 | 7 | ✔ |
| cycle/resolver | `Resolve/H=24` | 2.30 | ±10% | 80 | 7 | ✔ |
| cycle/resolver | `Resolve/H=60` | 4.44 | ±7% | 80 | 7 | ✔ |
| cycle/resolver | `Resolve/H=120` | 7.30 | ±27% | 80 | 7 | ✔ |
| cycle/resolver | `ResolveMonth` | 84.01 | ±7% | 2240 | 202 | ✔ |
| cycle/resolver | `Calculate/H=6` | 2.00 | ±16% | 1208 | 14 | ✔ |
| cycle/resolver | `Calculate/H=24` | 13.99 | ±10% | 4280 | 18 | ✔ |
| cycle/resolver | `Calculate/H=60` | 37.59 | ±21% | 10424 | 20 | ✔ |
| cycle/resolver | `Calculate/H=120` | 69.65 | ±26% | 20664 | 22 | ✔ |
| cycle/view | `Build/H=6` | 9.28 | ±7% | 5376 | 61 | ✔ |
| cycle/view | `Build/H=24` | 36.15 | ±30% | 15504 | 83 | ✔ |
| cycle/view | `Build/H=60` | 100.80 | ±18% | 37296 | 121 | ✔ |
| cycle/view | `Build/H=120` | 191.63 | ±14% | 72593 | 183 | ✔ |
| cycle/view | `DayJSON` | 139.21 | ±20% | 103644 | 880 | ✔ |
| fertility | `Insights/H=6` | 494.72 | ±26% | 69548 | 1765 | ✔ |
| fertility | `Insights/H=24` | 642.46 | ±55% | 70424 | 1765 | ✔ |
| fertility | `Insights/H=60` | 1041.19 | ±13% | 72344 | 1765 | ✔ |
| fertility | `Insights/H=120` | 1507.33 | ±14% | 75417 | 1766 | ✔ |
| fertility | `BBTChart/range=1` | 605.68 | ±13% | 55091 | 1490 | ✔ |
| fertility | `BBTChart/range=3` | 634.29 | ±7% | 83368 | 1980 | ✔ |
| fertility | `BBTChart/range=6` | 709.71 | ±11% | 133283 | 2848 | ✔ |
| analysis | `Summary/1m` | 1166.10 | ±18% | 348594 | 3152 | 3152–3153 |
| analysis | `Summary/3m` | 1488.99 | ±34% | 384970 | 3432 | ✔ |
| analysis | `Summary/1y` | 2321.81 | ±24% | 644095 | 4599 | 4598–4599 |
| messages/rules | `Select_200Rules` | 44.59 | ±18% | 82138 | 14 | ✔ |
| messages/rules | `Select_PrefetchedStats50` | 19.89 | ±23% | 21496 | 152 | ✔ |
| auth/passport | `VerifyParallel` | 21.83 | ±30% | 6986 | 65 | ✔ |
| auth/passport | `Verify` | 58.41 | ±17% | 6844 | 65 | ✔ |

*allocs stable* = all 6 samples had the same count (a range = amortised growth; the gate's 0.1 % tolerance covers it).
The `cycle/engine` rows are the v6 engine's own benchmarks; T-M9-20 was editing that package while this was recorded
(`EncodeDay` / `TodayPayload` appeared between two runs), so expect that task to re-record them.

What each one measures:

| Benchmark | Code path |
|---|---|
| `cycle/resolver` `Resolve/H`, `ResolveMonth` | `resolver.Resolve` (v1.1 status) for today; 31 days over one metrics value |
| `cycle/resolver` `Calculate/H` | `metrics.Calculate` (three-layer metrics) |
| `cycle/view` `Build/H` | `view.Build` — the §19 `cycle_view` of `/cycle/today` |
| `cycle/view` `DayJSON` | the cached `/cycle/today` body `{calculation, cycle_view}` encoded |
| `cycle/legacy` `SnapshotMonth/{calendar,full}/H` | `Snapshot.Month` with a fresh engine per iteration, as per request |
| `cycle/legacy` `MonthJSON/{calendar,full}` | the `/cycle/month` body encoded |
| `cycle/cache` `KeyString/{day,month}/H` | `cache.Key.String()` — JSON of every input row + SHA-256, paid before every Redis lookup |
| `cycle/engine` `*` | the v6 engine's own benches (T-M9-15…20; `Prepare`, `Resolve`, `Range31`, …) |
| `fertility` `Insights/H`, `BBTChart/range` | the pure part of `/fertility/insights` and `/fertility/bbt` (+ the body map) |
| `analysis` `Summary/{1m,3m,1y}` | `BuildSummary` + localized body + JSON encode |
| `messages/rules` `Select_200Rules` | `rules.Select` over 200 rules, history prefetched (README ceiling ≤ 200 µs) |
| `auth/passport` `Verify`, `VerifyParallel` | RS256 verify + the cached client lookup (ceiling ≤ 150 µs) |

First reading against README §8: `rules.Select` (200 rules) and auth `Verify` are already inside their ceilings.
The legacy path's costs sit in JSON: hashing the cache **key** of a month (every input row marshalled) costs more
than computing the calendar month itself, and the full month body allocates ~27 k times. Those are the targets of the engine/cache tasks.

## 3. Query budgets per route

`backend-go/internal/querybudget` — one integration test per hot route on the real router and MariaDB, counted by
the T-M9-07 per-request counter, **auth and module gate included**. Users come from `benchfixtures` (24 periods,
a year of logs). *Cold* = first request of a fresh process (empty Redis, guards without the OAuth client list,
config caches off); *warm* = the same request again with every cache filled. The budget table
(`querybudget/budgets.go`) starts at these counts; a task that removes queries lowers its row.

| Route | Fixture user | Cold | Warm | README §8 target |
|---|---|---:|---:|---|
| `GET /cycle/today` | cycle | 8 | 4 | warm 1, cold ≤ 5 |
| `GET /cycle/date/:date` | cycle | 8 | 4 | warm 1, cold ≤ 5 |
| `GET /cycle/month/:y/:m?view=calendar` | cycle | 6 | 4 | warm 1, cold ≤ 5 |
| `GET /cycle/month/:y/:m` (full) | cycle | 8 | 4 | — |
| `GET /home` | cycle | 20 | 15 | ≤ 8 |
| `GET /home/cycle-overview` | cycle | 9 | 7 | — |
| `GET /messages/daily` | cycle | 11 | 8 | ≤ 4 |
| `GET /messages/selected?placement=smart_tip` | cycle | 10 | 7 | ≤ 4 |
| `GET /fertility/insights` | ttc | 8 | 5 | ≤ 4 |
| `GET /fertility/bbt` | ttc | 7 | 4 | — |
| `GET /analysis/summary?range=3m` | cycle | 13 | 11 | ≤ 6 |
| `GET /analysis/monthly/:ym` | cycle | 16 | 14 | — |
| `GET /analysis/ttc` | ttc | 14 | 11 | — |
| `GET /analysis/pregnancy` | pregnancy | 17 | 15 | — |

The auth step alone (`querybudget.Auth`, T-M9-09) is **2 cold / 1 warm**, part of every warm row above. README §8
targets (warm 1 for the engine GETs, ≤ 8 home, ≤ 4 messages/fertility, ≤ 6 analysis) are what M9/M10 tasks
lower these rows to.

## 4. The CI gate

`.github/workflows/bench.yml` (push and PR on `backend-go/**`):

1. runs the benchmarks with `-count 6` (`bench/run.sh`) on the Go version named in the baseline header;
2. runs the **base commit** (PR base, or the push's previous commit) on the same runner, when it has `bench/run.sh`;
3. `go run ./bench/gate` compares medians, matching names without the `-GOMAXPROCS` suffix:
   - **fails** when any `allocs/op` is above `bench/baseline.txt` (tolerance 0.1 % of the baseline, i.e. none
     below 1 000 allocs/op — it only absorbs amortised map growth in the 27 k-alloc month body);
   - **warns** (annotation) when `sec/op` is > 20 % slower than the base-commit run; without a base run time is
     not compared, because the baseline file comes from another machine;
4. writes a benchstat table (base vs this commit) to the job summary and uploads the result files.

The gate's own tests (`bench/gate/gate_test.go`): an allocs regression → exit 1; a time-only regression → exit 0
plus a warning.

## 5. Re-recording

```bash
cd backend-go
make bench            # → bench/current.txt (BENCH_COUNT=6 BENCH_TIME=1s; header = machine + Go + load)
make bench-compare    # benchstat bench/baseline.txt bench/current.txt, then the gate
make bench-baseline   # → bench/baseline.txt — on a quiet machine, commit it
make test-int PKG=./internal/querybudget/... GOFLAGS=-v   # prints the measured query table
```

Re-record `bench/baseline.txt` (and update the tables here) in the same change when a task intentionally changes
allocations — an optimisation that lowers them, or a feature that adds work — or when the Go toolchain changes
(the CI job installs the version in the header). Lower the query budget rows in `budgets.go` whenever a count
drops. Prefer a quiet machine: the time columns are only comparable between runs on the same machine.
