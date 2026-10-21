---
id: T-M9-21
title: Engine v6 today_ui mapping and §2.35 JSON encoder
milestone: M9
type: backend
status: done
depends_on: [T-M9-20]
parallel_group: M9-B
touches: [backend-go/internal/cycle/engine]
skills: []
verify: cd backend-go && go vet ./internal/cycle/engine/... && go test -race ./internal/cycle/engine/...
---

# T-M9-21 — Engine v6 today_ui mapping and §2.35 JSON encoder
**Model:** opus.

## Why
The Today-screen rules (§2.43) are split between the server daily card and the client `hero-state.ts`, each with
different thresholds (start prompt at −2 instead of −3 with no +14 limit, end prompt from day 1, end-missing from a
client rule, no overdue/last-period banners, no single-banner priority). E-17 moves the mapping into the engine so web,
the daily card and later clients cannot drift. The §2.35 payload needs one fast, exact encoder.

## Scope
1. `TodayUI(st Status, s *State) TodayUI` (pure), per E-17 and §2.43:
   - `ring_action`: `start` when no period is open and target ∈ [next − 3, next + 14]; `end` when the current period's
     end is unconfirmed and `days_since ∈ [EPL − 1, hard_cap]`; otherwise `edit`;
   - `banner` (at most one, table order): `end_missing` (warning or hard cap exceeded) → `overdue`
     (`predicted_period_overdue`, `days_late ≥ 8`) → `ask_last_period` (`requires_user_input` with
     `stale_onboarding_anchor` or `cycle_unresolved_after_expected_period`) → null;
   - `show_pregnancy_test_button` = `suggest_pregnancy_test` on the overdue banner;
   - `prediction_card`: period `{next, next + EPL − 1}`, PMS `{next − 3, next − 1}`, fertile display window or null,
     ovulation `{date, source}` or null, `cycle_length`, `cycle_regularity`; null blocks under suppression/suspension.
2. Encoder for the §2.35 shape plus `today_ui` and `limits`: fixed key order, ISO dates, nulls where the spec says null,
   warnings/reasons in enum order; `future_cycles` included when passed. Implementation chosen by benchmark
   (`strconv.Append*` into a pooled buffer, or `encoding/json` over value structs if it meets the budget); the golden
   test fixes the bytes.
3. A matching decoder for tests and the shadow diff (T-M9-32).

## Tests
- §2.35 worked example: periods 07-01..07-05 confirmed, prior history giving ECL 29, variability 3, ≥ 2 closed periods
  of 5; target = today = 07-18 → **byte-identical** to the spec JSON (with `future_cycles` k=1..6 per E-05, `today_ui`
  and `limits` appended — the golden documents the additions; every spec field matches).
- today_ui boundaries: next − 4 → edit; next − 3 → start; next + 14 → start; next + 15 → edit; open period day EPL − 2 →
  edit, EPL − 1 → end, hard_cap → end, hard_cap + 1 → edit + `end_missing`; late 7 → no banner; late 8 ttc → overdue +
  test button; late 8 with pill → overdue without button; stale anchor → `ask_last_period`; end-missing and overdue at
  once → only `end_missing`.
- Prediction card under the pill → fertile and ovulation null; pregnancy → card null.
- Encoder round trip over 1 000 random statuses (decode(encode(x)) == x).

## Budgets
- `BenchmarkEncodeDay` ≤ 20 µs; `TodayUI` 0 allocs.

## Out of scope
- Serving it (T-M9-32), the frontend (T-M9-42/43).

## Acceptance
- Tests green; the §2.35 golden committed under `internal/cycle/engine/testdata/`.
