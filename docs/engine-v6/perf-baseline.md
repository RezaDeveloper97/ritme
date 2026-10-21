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
| cycle/engine | `EncodeDay` | 0.62 | ±4% | 0 | 0 | ✔ |
| cycle/engine | `TodayPayload` | 1.13 | ±5% | 192 | 1 | ✔ |
| cycle/engine | `Range31` | 9.44 | ±42% | 6528 | 1 | ✔ |
| cycle/engine | `FutureCycles` | 0.28 | ±9% | 192 | 1 | ✔ |
| cycle/engine | `Resolve_Hormonal` | 0.21 | ±4% | 0 | 0 | ✔ |
| cycle/engine | `Prepare_Markers` | 18.78 | ±7% | 8952 | 10 | ✔ |
| cycle/engine | `Prepare` | 2.89 | ±12% | 5272 | 8 | ✔ |
| cycle/engine | `Resolve` | 0.21 | ±3% | 0 | 0 | ✔ |
| cycle/engine | `TodayUI` | 0.02 | ±0% | 0 | 0 | ✔ |
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
The `cycle/engine` rows are the v6 engine's own benchmarks, **re-recorded by T-M9-22** (2026-10-21 03:56 UTC, load
18 / 16 / 22, `07737dec` + the T-M9-22 working tree) after the budgets were locked — see §2a. The earlier rows were
taken at load 26 / 38 / 45 while T-M9-20/21 were still editing the package (`EncodeDay` 7.73 µs and `TodayPayload`
14.28 µs then; the encoder and its fixture changed after that run).

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

## 2a. Engine v6 locked budgets (T-M9-22)

`backend-go/internal/cycle/engine/budget_test.go` defines every engine benchmark from one table and locks it:

- **Time.** A run with b.N ≥ 1 000 **fails** when its mean is above the ceiling. The README §8 ceiling is used
  where it names one; otherwise the ceiling is set here.
- **Allocations.** The count measured at lock time. `TestBudgets_Allocs` checks it in every `go test`, and the bench
  gate checks the same numbers against `bench/baseline.txt`.

The medians below come from `-count 6 -benchtime 1s` (Range31: `-count 10`). Load average was 11–18 on 10 cores: two
other agents (T-M9-13b, T-M9-14) were building and testing at the same time. The ceilings are 10–30× above every
median, so load does not make the lock flaky.

| Benchmark | Fixture | median | min–max | ceiling | allocs (lock) | README §8 |
|---|---|---:|---:|---:|---:|---|
| `Prepare` | 60 periods, a skip, declared values | 2.89 µs | 2.48–3.18 | 300 µs | 8 | Prepare ≤ 300 µs |
| `Prepare_Markers` | 60 periods + 400 BBT + 60 LH | 18.78 µs | 17.61–20.22 | 300 µs | 10 | ≤ 300 µs ✔ |
| `Resolve` | 24 periods, 330 targets | 0.21 µs | 0.20–0.22 | 3 µs | 0 | ≤ 3 µs, ≤ 3 allocs ✔ |
| `Resolve_Hormonal` | 24 periods on the pill | 0.21 µs | 0.20–0.22 | 3 µs | 0 | as Resolve ✔ |
| `Range31` | 24 periods, 31 days crossing today | **7.62 µs** (count 10) | 6.41–8.08 | 150 µs | 1 | ≤ 150 µs, ≤ 40 allocs ✔ |
| `FutureCycles` | 24 periods, 6 cycles | 0.28 µs | 0.24–0.29 | 5 µs | 1 | — |
| `TodayUI` | one status | 0.015 µs | 0.015–0.015 | 1 µs | 0 | — |
| `EncodeDay` | the §2.35 today payload, reused buffer | 0.62 µs | 0.61–0.66 | 20 µs | 0 | ≤ 20 µs ✔ |
| `TodayPayload` | Resolve + FutureCycles + TodayUI + encode | 1.13 µs | 1.08–1.19 | 30 µs | 1 | — |

**Range31 re-measured.** T-M9-20 saw 9.1 µs idle and T-M9-21 saw 29.9 µs under load. Now it is 7.62 µs (median of
10 runs, load ≈ 11–14) and 9.44 µs in the 6-run baseline (load ≈ 18). The baseline's spread is 7.0–14.9 µs. Its
one allocation is the 31-status result slice (6.5 kB). One short `-benchtime 2000x` verify run reported 6 530 B/op:
over so few iterations, runtime background bytes are averaged in. allocs/op stayed at 1.

Raising a lock is a decision. Change it in `budget_test.go`, re-record the engine rows (below) and give the reason
in the task report. The engine rows of `bench/baseline.txt` were re-recorded with
`BENCH_PKGS='./internal/cycle/engine/...' bench/run.sh` and spliced over the old engine block (header line 8 names
the run); `make bench-gate` passes against it.

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

## T-M9-13b — config caches on /home and the language registry (2026-10-21)

Measured with `make test-int PKG="-v ./internal/querybudget/..."` (cold = configcache off, warm = caches filled).

| Route | Before (cold / warm) | After (cold / warm) |
|---|---|---|
| `/api/v1/home` | 20 / 15 | 20 / **11** |
| `/api/v1/analysis/summary?range=3m` | 13 / 11 | 14 / 11 |
| `/api/v1/analysis/monthly/:ym` | 16 / 14 | 17 / 14 |
| `/api/v1/analysis/ttc` | 14 / 11 | 15 / 11 |
| `/api/v1/analysis/pregnancy` | 17 / 15 | 18 / 15 |

- Home warm drops by 4: task templates, affirmations, articles and challenges come from configcache snapshots.
- The language list is a configcache snapshot, so there is no Redis read per request. The analysis cold rows gain one query because with configcache off the snapshot reloads on each call; warm is unchanged, and in production the snapshot loads once per process.
- `messages selected` stays constant in the number of rules (1 rule and 50 rules: 8 warm queries). With capped or cooled-down candidate rules, the single `ListUserMessageEventStats` prefetch adds 1 over the table row.
