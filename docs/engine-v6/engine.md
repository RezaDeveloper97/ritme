# Cycle engine v6 — package guide

`backend-go/internal/cycle/engine` is the v6 cycle engine (SPEC.md Part 2, README §2, DECISIONS.md X-01). Every
cycle consumer (today, calendar, home, messages, fertility, analysis) reads one engine output. This page covers the
API, which file implements each rule and the decision (E-id) behind it, the hardening suite, and how to add a rule.

## 1. Rules of the package

Enforced by `purity_test.go`:

- **Pure.** No I/O, no clock, and no mutable package-level variables. "Today" is always an argument. The package never
  imports `os`, `net/...`, `log/slog`, `database/sql`, redis, `math/rand`, `internal/clinicalconfig`,
  `internal/enums`, `internal/platform/db` or `internal/platform/clock`.
- **Inputs.** Everything arrives as `Input` (the user's data) and `Params` (the numbers).
  `internal/cycle/engineparams.For(d)` builds `Params` from the clinical config valid on day `d` (X-11).
  `DefaultParams()` holds the code defaults, which equal version 1 of every clinical key.
- **Dates** are `Day` values: `int32` days since 1970-01-01, using civil arithmetic. `NoDay` means "no date". No
  `time.Time` is used in the hot path.
- **Optional values** are `Opt[T]`, never pointers. The zero `Input` means "no data": `Day(0)` is a real date, so
  every optional date is an `Opt[Day]`.
- **Enums** are hand-written here. `String()` returns the spec spelling, and the zero value is the unknown/none
  member. They do not depend on the frozen v1.1 enums.
- **Hot paths do not allocate.** `Resolve`, `TodayUI`, `Suppression` and `AppendJSON` (into a caller-owned buffer)
  allocate 0 times. `Range` and `FutureCycles` allocate once (their result). `Prepare` allocates a small number of
  times that depends only on the input shape. §6 lists the locks.

## 2. API

```go
s := engine.Prepare(in, p)                     // today-independent; cache per (input revision, params fingerprint)
st := s.Resolve(target, today)                 // one day (§2.20–§2.34, §2.41, §2.42, E-03, E-15)
days := s.Range(from, to, today)               // ≤ MaxRangeDays (400) days; == map(Resolve), but faster
fcs := s.FutureCycles(today)                   // future_cycles k = 1..6 (E-03…E-05, E-28)
st, ui := s.Today(today)                       // today's status with FutureCycles filled + today_ui (E-17)
ui := s.TodayUI(st)                            // the §2.43 mapping of any status
lim := s.Suppression(day)                      // the E-24 limits block of one day
buf = engine.AppendJSON(buf[:0], &st, &ui)     // §2.35 JSON (+ limits, future_k, today_ui); ui may be nil
st, ui, err := engine.DecodeJSON(b)            // strict inverse (tests, the T-M9-32 shadow diff)
```

| Type | What it is |
|---|---|
| `Input` | `Periods` (real records, any order), `Skips`, `Declared` (onboarding start and lengths, `Manual` = E-20), `BirthDate` / `TeenBandOnly`, `Mode`, `Hormonal`, `DeliveryDate`, `Breastfeeding`, `LH`, `BBT`, `PregnancyTests` |
| `Status` | Every §2.35 field, plus `Limits` (E-24), `FutureK` (E-03) and `FutureCycles` (filled only by `Today`) |
| `TodayUI` | `RingAction`, `Banner`, `ShowPregnancyTestButton`, `PredictionCard` (E-17) |
| `State` | Read-only after `Prepare`, so it is safe to share between goroutines. Accessors: `Periods`, `Excluded`, `Contradictory`, `Skips`, `Cycles`, `CycleLength`, `PeriodLength`, `Variability`, `ValidCycleCount`, `ValidPeriodCount`, `Ovulations`, `LutealLength`, `BBT`, `LH`, `Diagnostics` (ignored skips), `Params` |

The loader (T-M9-24) builds `Input`. It must:

- set `End = NoDay` for an open period;
- wrap dates in `Some(...)`;
- map a faint LH test to "not positive";
- pass `Skips` exactly as stored.

## 3. Pipeline and files

`Prepare` runs five parts in order. None of them depends on today.

| Part | File | Rules |
|---|---|---|
| 1 | `prepare.go` | Sort the periods. Drop conflicting records: the newer one is ignored (E-02). Build real cycles, skips, outliers (§2.7, §2.8), effective lengths (§2.9, E-20) and variability (§2.27). |
| 2 | `ovulation.go`, `bbt/` | Per-cycle ovulation in priority BBT 3-over-6 > LH > calendar. The LH/BBT conflict (§2.17, E-16). The effective luteal length (§2.16). |
| 3 | `quality.go` | The clinical_flags bases (E-06, E-07) and the outlier signals (§2.28). |
| 4 | `limits.go` | Positive pregnancy tests (E-26). |
| 5 | `anchors.go` | Skip anchors (E-15). |

At resolve time:

- `anchors.go`: the anchor and the cycle of a target — the stale onboarding anchor (§2.41) and skip anchors.
- `layout.go`: one cycle's day map (§2.11, §2.16–§2.19, §2.23–§2.26).
- `resolve.go`: the §2.20 priority (menstrual tiers and caps, period_expected tiers, phases), source warnings and E-10.
- `quality.go` (`grade`): regularity, clinical flags, data_quality, confidence, reasons and requires_user_input.
- `limits.go`: suppression.
- `future.go`: today's projection (E-03…E-05, E-28) and `Range`.
- `todayui.go`: the §2.43 mapping (E-17).
- `encode.go` / `decode.go`: the §2.35 JSON.

## 4. Rules and the decisions they implement

| Decision | Rule | Where |
|---|---|---|
| E-01 | `prediction_based_output` only for a predicted_reference anchor (stale, skip, future) | `anchors.go` `cycleAt`, `future.go` `futureCycle` |
| E-02 | `contradictory_anchor_data`; the newer of two conflicting records is excluded; data quality is poor | `prepare.go` `preparePeriods`, `quality.go` `grade` |
| E-03 | A future target on or after today's next start resolves in `future_cycles[k]`; confidence is `min(today, k == 1 ? high : medium)`; no lateness after today | `future.go` `chainAt` / `futureCycle`, `resolve.go` `resolveWith` |
| E-04 | While late, `future_cycles[1]` starts tomorrow; late > max_late gives `[]` | `future.go` `chainAt` |
| E-05 | `future_cycles` items k = 1..6 with eight keys | `future.go` `FutureCycles`, `encode.go` |
| E-06, E-07 | `clinical_flags` frequency / duration / absent | `quality.go` `prepareQuality`, `clinicalFlags`, `absentAt` |
| E-08, E-27 | Teen (< 18, or teen band only) and age ≥ 45 rules, age taken at the target | `quality.go` `teenAt`, `ageAt`, `grade` |
| E-09 | Regularity over the 12 calendar months before `min(target, today)`; skips excluded, outliers included | `quality.go` `regularity` |
| E-10 | A cycle from the declared onboarding start resolves as `onboarding_based` (`default_based` with default lengths) | `resolve.go` `resolveCycle` |
| E-11 | > max_late days late: unknown phase, still `prediction`, poor and low | `resolve.go` `resolveCycle` |
| E-12, E-21 | Cap warnings are cumulative; an open period past a cap stays poor/low with requires_user_input | `resolve.go` `menstrual` |
| E-13 | Hormonal contraception: logged bleeding is menstrual, period-expected tiers stay, every other day is unknown, no fertile window or ovulation | `resolve.go`, `limits.go`, `future.go` |
| E-14 | Postpartum / breastfeeding: nothing is predicted until 2 real periods after delivery; confidence unknown, then low; `cycle_day` only on logged bleeding (T-M9-53) | `limits.go` `postpartumPeriods`, `quality.go` `limitGrade` |
| E-15 | A skip equal to the predicted start becomes a no-period anchor; other skips are ignored (Diagnostics) | `anchors.go` `prepareSkipAnchors`, `skipAnchorIn` |
| E-16 | BBT strict 3-over-6; LH = first positive + 1; BBT > LH; conflict when the gap is > 2 days | `bbt/bbt.go`, `ovulation.go` |
| E-17 | today_ui ring / banner / test button / prediction card | `todayui.go` |
| E-18 | Periods of 9–10 days: no extra rule | (none) |
| E-19 | High variability lowers confidence only without a BBT-confirmed ovulation | `quality.go` `grade` |
| E-20 | Manual lengths override the medians as `declared`, without `onboarding_data_used` | `prepare.go` `effective`, `resolve.go` |
| E-22 | A completed past cycle is laid out around its actual next start; it has no period_expected days | `resolve.go` `layoutFor` |
| E-23 | `data_quality = good` needs history-sourced lengths | `quality.go` `grade` |
| E-24 | The `limits` block, and suppression precedence pregnancy > postpartum > hormonal | `limits.go` |
| E-25 | Pregnancy mode suspends the engine: no warnings, no prediction, `cycle_day` only on logged bleeding | `resolve.go`, `quality.go` `limitGrade` |
| E-26 | A positive test since the cycle start prompts pregnancy mode (cycle and ttc modes) | `limits.go` `promptPregnancy` |
| E-28 | Nothing is predicted after `future_cycles[6].period_end` | `future.go` `futureCycle` |
| E-29 | Menopause and teen modes are computed normally | `limits.go` |
| §2.41 | A stale onboarding anchor (declared-only history) rolls forward by whole cycles | `anchors.go` `cycleAt` |
| X-11 | Clinical numbers come from `Params`, filled by `engineparams.For` | `params.go` |

Hardening fixes (T-M9-22). FuzzPrepare found these; each is a named case in `TestInvariants_FuzzRegressions`.

1. **A logged period that runs past today was projected.** A closed period with a logged end after today, or a
   period longer than the cycle, sent its later days to `future_cycles` and showed them as fertile. Fix: a target
   inside a closed real period resolves from the history (`loggedPeriodCovers`, `resolve.go`). This enforces §2.40,
   "no fertile on menstrual days".
2. **A skip inside a logged period became an anchor.** The predicted start fell on a logged bleeding day of an
   over-long period. Fix: the log wins and the skip is ignored (`prepareSkipAnchors`).
3. **An LH positive on the cycle's last day.** It put the ovulation on the next real start (O = next), so the
   ovulation was not before the period. Fix: such a positive yields no LH ovulation, and the cycle falls back to
   BBT or the calendar (`cycleOvulation`).

## 5. The hardening suite

| File | What it checks |
|---|---|
| `invariants_test.go` | Builds 2 000 seeded random histories (`math/rand/v2`; skips, BBT, LH, the pill, postpartum, every mode, ages 14–55, malformed records) and resolves 400 days of each. It checks the invariants below. `stateViolations`, `violations` and `futureViolations` are shared with the fuzz target. `TestInvariants_DetectBreakage` proves that each checker fires. |
| `fuzz_test.go` | `FuzzPrepare`: unsorted, overlapping, duplicated and open records, dates within ±100 years, and every field. Nothing panics and every invariant holds. Seed corpus: `testdata/fuzz/FuzzPrepare`. |
| `bbt/bbt_test.go` | `FuzzBBTDetect`. Seed corpus: `bbt/testdata/fuzz/FuzzBBTDetect`. |
| `golden_test.go` | 21 personas in `testdata/personas/*.json`. Each golden holds the today payload plus 365 days of `Range` as day-to-day deltas. The §2.36 (21/28/35) and §2.35 anchors are asserted by hand. |
| `budget_test.go` | Every engine benchmark with its locked time ceiling and allocation lock (§6). |
| `purity_test.go` | The §1 rules. |
| the other `*_test.go` | One table per spec section and decision (T-M9-15…21). |

The invariants:

- `cycle_day ≥ 1` and equals `date − start + 1` whenever a start exists. Pregnancy and postpartum before cycles
  return (E-14) hide it outside logged bleeding.
- `main_phase` is the main phase of the subphase.
- Every start and end has a source. `is_confirmed` holds exactly when the end source is `user_logged`. An ovulation
  date has a source, and it comes before the predicted next start.
- No day is fertile on or before `current_period_end`, on a logged bleeding day, or after O. So O+1 is never fertile.
- `fertility_level` is `unknown` exactly when the phase is period_expected or unknown, or fertility is suppressed.
  Otherwise it follows the §2.26 table.
- `days_to_ovulation` is `O − date`. It is null without an O, when days late > max_late, or under suppression.
- `days_to_period` and `days_late` are both present exactly when a next start is predicted.
- `requires_user_input` is true exactly when one of the six §2.33 warnings is present.
- The `limits` block matches the suppression. Under suppression there is no ovulation and no test suggestion. In
  pregnancy there are no warnings and no next start.
- `future_k > 0` only after today, with a predicted_reference start.
- `future_cycles` has 0 or 6 items, k = 1..6. Their periods come after today, strictly increase, never overlap, and
  are ECL apart. k ≥ 2 is at most medium. The fertile window ends on an ovulation that comes before its period.
  There are no fertile dates under suppression, and the list is empty in pregnancy or postpartum-not-returned.
- The State keeps every record (kept + excluded). Kept periods are sorted, do not overlap, and at most one is open.
  Cycles join consecutive starts. Skips and markers are sorted. Effective lengths stay in range.
- `Range == map(Resolve)`, byte for byte.
- `Resolve` and `Today` are deterministic: a second `Prepare` of the same input gives the same bytes. `Prepare`
  does not modify its input.

Regenerate goldens with `go test ./internal/cycle/engine -run TestGoldens -update`, then review the diff. A golden
change is a behaviour change, so name the decision it implements.

## 6. Performance locks

The budgets are in `budget_test.go`. The numbers are in [`perf-baseline.md`](perf-baseline.md) §2a.

- A time ceiling fails a benchmark when its mean is above it and b.N ≥ 1 000. The task verify runs with
  `-benchtime 2000x`.
- An allocation lock is checked by `TestBudgets_Allocs` in every `go test`. The bench gate also checks it against
  `bench/baseline.txt`.

| Benchmark | Fixture | Ceiling | Allocs lock |
|---|---|---|---|
| `Prepare` | 60 periods | ≤ 300 µs | 8 |
| `Prepare_Markers` | 60 periods + 400 BBT + 60 LH | ≤ 300 µs (README §8) | 10 |
| `Resolve`, `Resolve_Hormonal` | 24 periods | ≤ 3 µs (README §8: ≤ 3 allocs) | 0 |
| `Range31` | 24 periods, 31 days | ≤ 150 µs (README §8: ≤ 40 allocs) | 1 |
| `FutureCycles` | 24 periods | ≤ 5 µs | 1 |
| `TodayUI` | one status | ≤ 1 µs | 0 |
| `EncodeDay` | today payload | ≤ 20 µs (README §8) | 0 |
| `TodayPayload` | Resolve + FutureCycles + TodayUI + encode | ≤ 30 µs | 1 |

## 7. How to add a rule

1. **Decision first.** If the spec is silent or ambiguous, add or extend an E-id in `DECISIONS.md` and get the
   owner's OK. Name the E-id in the code comment.
2. **Numbers go in `Params`.**
   - A clinical value is a clinicalconfig key: add it to the registry and to `engineparams.For`, and bump the
     fingerprint keys.
   - A product constant is a `DefaultParams` field.
   - Never use a literal in a rule.
3. **Put the rule in the right part.**
   - Anything that does not depend on the target or today goes in `Prepare`, as a new `prepareX` step stored on
     `State`.
   - Anything per day goes in `resolveCycle`, `layout` or `grade`, in its §2.20 priority.
   - A new output field goes in `types.go`, `encode.go`, `decode.go` (strict), the spec golden
     (`testdata/spec_2_35_today.json`, regenerated with `-update`) and `TestEncode_RoundTrip`'s generator.
4. **Do not allocate.** `TestResolve_NoAllocs` and `TestBudgets_Allocs` must stay green. Use bitsets, fixed buffers
   or slices prepared in `Prepare`.
5. **Tests.**
   - Write a table per spec sentence or decision boundary (threshold − 1 / threshold / threshold + 1).
   - Extend `violations()` when the rule creates a new invariant.
   - Extend `randomHistory()` / `fuzzInput` so the new input is generated.
   - Add a persona when the rule is a new user situation.
6. **Verify.** Run the T-M9-22 verify command: race, fuzz 30 s, coverage ≥ 95 % and the benches under their
   ceilings. Regenerate the goldens and review them. If allocations or time move, re-record the engine rows of
   `bench/baseline.txt` (`BENCH_PKGS='./internal/cycle/engine/...' bench/run.sh` spliced in, or
   `make bench-baseline`).
