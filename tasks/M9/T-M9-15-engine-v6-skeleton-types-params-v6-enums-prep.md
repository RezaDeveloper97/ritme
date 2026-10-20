---
id: T-M9-15
title: Engine v6 skeleton: types, params, v6 enums, Prepare and effective values
milestone: M9
type: backend
status: todo
depends_on: [T-M9-01]
parallel_group: M9-B
touches: [backend-go/internal/cycle/engine, backend-go/internal/cycle/engineparams, backend-go/internal/clinicalconfig]
skills: []
verify: cd backend-go && go vet ./internal/cycle/engine/... ./internal/cycle/engineparams/... ./internal/clinicalconfig/... && go test -race ./internal/cycle/engine/... ./internal/cycle/engineparams/... ./internal/clinicalconfig/...
---

# T-M9-15 — Engine v6 skeleton: types, params, v6 enums, Prepare and effective values
**Model:** opus (health logic). First of the engine chain T-M9-15 → 22; read `docs/engine-v6/README.md` §2 and
DECISIONS.md E-01…E-29, I-03, X-01, X-11 before writing code.

## Why
The v1.1 resolver is a PHP port pinned by Laravel goldens and PHP-generated enums; it reads a package-global
clinical-config snapshot (5 lookups per resolve, may spawn a reload goroutine), has no LH/BBT/skip/contraception/mode
inputs and builds `time.Time` values on every date step (gap 01 §1, §4). v6 changes semantics in ~15 places and the
output shape, so it gets a new pure package (X-01). This task lays the foundation every later engine task builds on:
types, enums, parameters and the today-independent `Prepare` step (§2.4–§2.10, §2.27 variability).

## Scope
1. Package `internal/cycle/engine` (doc.go states the rules: no I/O, no globals, no clock, no `clinicalconfig`,
   no `internal/enums` import; everything the engine needs arrives in `Input` and `Params`).
2. `Day` = `int32` days since 1970-01-01 with `NoDay = math.MinInt32`; `DayOf(civildate.Date)` / `Day.Civil()` /
   `Day.AppendISO([]byte)` using integer civil-date arithmetic (no `time.Time` in the engine's hot path). Optional
   values use a small `Opt[T]` value type, never pointers.
3. Hand-written v6 enums with `String()` = the spec spelling: main_phase, subphase, fertility_level,
   start source (§2.10: user_logged | onboarding_declared | predicted_reference | unknown), end source, ovulation
   source, cycle_regularity, data_quality, confidence, resolution_source, frequency, duration, mode
   (cycle, ttc, pregnancy, postpartum, menopause, teen), suppression reason (E-24). `Warning` = the 20 values of §2.32
   plus `contradictory_anchor_data` (E-02) as a `WarningSet uint32` bitset; `Reason` = the 14 values of §2.29 as a
   `ReasonSet uint16` bitset. Sets iterate in enum order (deterministic output, zero allocations).
4. Types (declared now, filled by later tasks): `Input` {Periods []Period{Start, End (NoDay = open), Source
   user_logged|bleeding_log}, Skips []Day, Declared{LastPeriodStart, CycleLength, PeriodLength, Manual}, BirthDate,
   TeenBandOnly, Mode, Hormonal, DeliveryDate, Breastfeeding, LH []{Date, Positive}, BBT []{Date, Centi int32},
   PregnancyTests []{Date, Positive}}; `Status` (every §2.35 field + `Limits` + `FutureK`); `Anchors`; `Metrics`;
   `ClinicalFlags`; `FutureCycle` (E-05 shape); `Limits` (E-24); `TodayUI` (E-17).
5. `Params` with every number the engine uses and `DefaultParams()` (code constants: 6 future cycles, prompt window
   −3..+14, absent 90 days, default lengths 28/5, valid ranges 21..45 and 2..10, teen < 18, perimenopause ≥ 45,
   2 postpartum periods).
6. Clinical registry (`internal/clinicalconfig/registry.go`, group engine, new source string
   `"Ritme cycle engine v6 (docs/engine-v6/SPEC.md Part 2)"` for the new keys only): `engine.luteal {default 14,
   min 10, max 16}`, `engine.bbt {strict 1, min_readings 10, rise_centi 20}`, `engine.lh_bbt_conflict {days 2}`,
   `engine.pregnancy_test {cycle_days_late 8, ttc_days_late 1}`, `engine.stale_anchor {extra_days 14}`,
   `cycle.teen.frequency {min 21, max 45}`, with lo/hi bounds and `Order` pairs. `engine.fertile_window.after` stays
   for v1.1 and is ignored by v6 (retired in T-M9-51).
7. `internal/cycle/engineparams`: `For(d civildate.Date) engine.Params` — the only reader of `engine.*`,
   `cycle.figo.*` and `cycle.teen.*` config, values valid on `d`; `Fingerprint(d)` = `clinicalconfig.Fingerprint(d,
   Keys...)` + a params schema version. `Keys` is the one list of keys the engine reads.
8. `Prepare(in Input, p Params) *State`, part 1 (today-independent, data-dependent only):
   - copy and sort periods; contradiction check (E-02): end < start, overlapping real periods, a start inside another
     period, more than one open period → state flag `contradictory`, and the newer conflicting record is excluded from
     medians;
   - real cycles from consecutive real starts; a cycle containing a recorded skip is neither valid nor an outlier
     (§2.8, E-15) and is marked `skipped`;
   - outlier flags (< 21 short, > 45 long), valid cycle lengths (21..45, not skipped), valid period lengths (closed
     periods, 2..10; an open period never counts);
   - effective cycle/period length = median of the last ≤ 3 valid records with half-up rounding
     (`platform/phpround`), fallback declared → 28 / 5, with the length source (history | declared | default);
     manual lengths (E-20) override with source declared;
   - `cycle_variability` = max − min of the last 3 valid cycle lengths (null with < 2).
   Effective values are computed over all stored records (as v1.1 does); they do not depend on the target date.

## Tests
- `Day`: round trip vs `time.Date` for every day 1900-01-01…2100-12-31; 2024-02-29 valid, 2100-02-29 invalid; ISO
  formatting.
- Medians: [28,29] → 29; [27,28] → 28; [28,30,35] → 30; one record 31 → 31; none + declared 31 → 31 (declared); none
  → 28/5 (default); four valid → only the last 3 count; the onboarding declared length never enters history medians.
- Skip: starts 01-01, 01-29, 02-26, skip 03-26, start 04-23 → valid [28,28]; the 56-day gap is neither valid nor an
  outlier and is marked skipped.
- Outliers [18,50] → no valid record, both flags; a 1-day and an 11-day closed period are invalid; an open period is
  excluded from period medians.
- Contradictions: end < start; two overlapping periods; two open periods → `contradictory`, newer record ignored.
- Manual lengths: history [28,28,28] + manual 32/6 → 32/6, source declared.
- Variability: one valid → null; [27,29,33] → 6.
- engineparams: `For` equals `DefaultParams()` with no approved versions; an approved version of `engine.luteal`
  valid from 2026-11-01 changes `For(2026-11-01)` and its fingerprint but not `For(2026-10-31)`; every registry key with
  prefix `engine.`, `cycle.figo.` or `cycle.teen.` is in `Keys` (except `engine.fertile_window.after`, documented).
- Registry: the new keys validate bounds and order like the existing ones.
- Purity: a test parses the package (`go/parser`) and fails on `time.Now`, mutable package-level vars and imports of
  `database/sql`, `net/...`, `os`, `log/slog`, redis, `internal/clinicalconfig`, `internal/enums`, `internal/platform/db`.

## Out of scope
- Resolve (T-M9-16), BBT/LH (T-M9-17), quality (T-M9-18), limits (T-M9-19), future/skip/Range (T-M9-20), JSON (T-M9-21).

## Acceptance
- Package builds with the types, enums and `Prepare` part 1; tests green with `-race`; `BenchmarkPrepare` (60 periods)
  exists (budget locked in T-M9-22).
