// Package engine is the Ritme cycle engine v6 (docs/engine-v6/SPEC.md Part 2, README.md §2, DECISIONS.md E-*/X-01).
//
// Rules — enforced by purity_test.go:
//   - Pure: no I/O, no globals (no mutable package-level variables), no clock. "Today" is always an argument.
//   - Everything the engine needs arrives in Input (the user's data) and Params (the numbers, built by
//     internal/cycle/engineparams from the clinical config valid on the target date). This package never imports
//     internal/clinicalconfig, internal/enums, internal/platform/db, database/sql, net/..., os, log/slog or redis.
//   - Dates inside are Day values (int32 days since 1970-01-01, civil arithmetic); no time.Time in the hot path.
//   - Optional values are Opt[T] values, never pointers.
//   - The v6 enums are hand-written here; their String() is the spec spelling. They are independent of the v1.1
//     generated enums, which stay frozen with the v1.1 resolver until T-M9-51.
//
// Pipeline: Prepare (today-independent: sort, validate, cycles, medians, variability, per-cycle BBT/LH ovulation via
// the bbt sub-package, effective luteal length) → State.Resolve / Range /
// FutureCycles (T-M9-16…20) → the §2.35 encoder (T-M9-21). A State never depends on "today", so it can be cached per
// input revision and parameter fingerprint.
package engine
