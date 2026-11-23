---
id: T-M12-20
title: Deterministic tests: fixed clock instead of time.Now, no sleeps, no self-skips, golden sweeps recorded or removed
milestone: M12
type: backend
status: done
depends_on: [T-M12-11]
parallel_group: M12-TEST
touches: [backend-go/internal/labs/labs_int_test.go, backend-go/internal/admin/users/v2_int_test.go, backend-go/internal/admin/messages/simulate_int_test.go, backend-go/internal/admin/aicredits/api_int_test.go, backend-go/internal/cityservices/admin_dir_int_test.go, backend-go/internal/messages/inbox/inbox_int_test.go, backend-go/internal/safety/scenarios_test.go, backend-go/internal/services/services_test.go, backend-go/internal/voicelog/voicelog_test.go, backend-go/internal/messages/feed_int_test.go, backend-go/internal/assistant/security_int_test.go, backend-go/internal/platform/idempotency/idempotency_test.go, backend-go/internal/cycle/state/state_test.go, backend-go/internal/analysis/cache_test.go, backend-go/internal/cycle/legacy/golden_sweep_test.go, backend-go/internal/cycle/view/golden_sweep_test.go]
skills: []
verify: cd backend-go && for i in 1 2 3; do go test -race -count=1 ./internal/safety/... ./internal/services/... ./internal/voicelog/... ./internal/cycle/... ./internal/analysis/... ./internal/platform/idempotency/... || exit 1; done && make test-int PKG="./internal/labs/... ./internal/admin/... ./internal/messages/... ./internal/assistant/... ./internal/cityservices/..."
---

# T-M12-20 — Deterministic tests: fixed clock instead of time.Now, no sleeps, no self-skips, golden sweeps recorded or removed
**Model:** sonnet (opus reviews). Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-07: tests that read the real clock break at month ends and near midnight; `labs_int_test.go:786` hides itself on days 1-2; 13 sleeps make CI flaky; two golden sweeps have never run.

## Scope
1. Inject the existing test clock in the listed tests; pin instants at the edges (last day of month 23:59 Tehran, 1 Farvardin, 30 Esfand).
2. Remove the self-skip in labs; make the test pass on any date.
3. Replace `time.Sleep` with channels/`Eventually` with a deadline or the fake clock.
4. Golden sweeps: record the goldens if the legacy engine is still a reference, otherwise delete the sweeps (T-M9-51 retired v1.1 — check).
5. Add a grep check (Makefile `lint-tests`) that fails on new `time.Sleep(` in `_test.go` without a `//nolint:sleep` reason.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Listed tests pass 3× in a row with `-race`, and with the system date faked to the 1st and the last day of a month (`faketime` or the test clock).
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
