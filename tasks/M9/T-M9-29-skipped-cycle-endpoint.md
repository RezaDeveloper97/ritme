---
id: T-M9-29
title: Skipped-cycle endpoint («پریود نداشتم»)
milestone: M9
type: backend
status: todo
depends_on: [T-M9-20, T-M9-25, T-M9-30]
parallel_group: M9-C
touches: [backend-go/internal/cycle/skips, backend-go/internal/http/routes_cycle.go, backend-go/db/queries/cycle/skips.sql, backend-go/api/openapi.yaml, backend-go/internal/cycle/revision/manifest.go]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/cycle/... ./internal/http/... && make test-int PKG=./internal/cycle/skips/...
---

# T-M9-29 — Skipped-cycle endpoint («پریود نداشتم»)
**Model:** opus.

## Why
§2.39.8: when the expected period does not come, the user records «پریود نداشتم»; the skip is stored, excluded from
effective values and regularity, the next prediction moves by one cycle, and `skipped_cycle_recorded` is shown. Today
the daily card offers `period_not_started` but nothing stores it (gap 01 §2.39). DECISIONS.md E-15.

## Scope
1. `POST /api/v1/cycle/skips {expected_start}`: allowed when, by the v6 state for `LocalToday`, `expected_start`
   equals the current `predicted_next_period_start` and `today ≥ expected_start` (any lateness, including > 14) and
   no real period has started since. Otherwise 422 with a code (`not_expected_yet`, `period_started`,
   `prediction_mismatch`). Repeating an existing skip → 200 (idempotent). Validation never returns 401.
2. `DELETE /api/v1/cycle/skips/{date}` → 204; unknown date → 404.
3. Both write in a transaction with the per-user lock and `revision.Bump`; manifest entries added.
4. Response of POST/DELETE: the new `cycle_view` for today (so the client updates without a second call) — encoded by
   the same function T-M9-32 uses; until T-M9-32 lands, return `{expected_start}` and note it.
5. OpenAPI + Go-only route allow-list (D-125). Throttle like other cycle writes.

## Tests
- Accept on the expected day and when 15 days late; duplicate → 200; before the expected day → 422; after a real start
  → 422; wrong date → 422; delete → previous late output restored; another user's skip → 404.
- Revision bumped once per successful write; OpenAPI route-coverage test green.

## Out of scope
- The frontend action (T-M9-47).

## Acceptance
- Tests green; PROGRESS.md cites D-125.
