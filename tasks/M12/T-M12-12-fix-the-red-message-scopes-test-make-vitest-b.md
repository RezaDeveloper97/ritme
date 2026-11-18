---
id: T-M12-12
title: Fix the red message-scopes test, make vitest blocking, run admin-web tests, add coverage reports
milestone: M12
type: frontend
status: todo
depends_on: []
parallel_group: M12-TEST
touches: [frontend/src/app/message-scopes.ts, frontend/src/app/message-scopes.test.ts, frontend/src/app/[locale], frontend/vitest.config.ts, admin-web/vitest.config.ts, .github/workflows/claude-to-stage.yml, .github/workflows/ci.yml, backend-go/Makefile]
skills: [verify-all]
verify: cd frontend && npm test && npm run typecheck && cd ../admin-web && npm test
---

# T-M12-12 — Fix the red message-scopes test, make vitest blocking, run admin-web tests, add coverage reports
**Model:** sonnet. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-02: `message-scopes.test.ts:107` has been red for 8 weeks behind `continue-on-error` — 7 routes ship namespaces that no longer match their screens (a missing namespace = raw keys shown to users). admin-web tests never run.

## Scope
1. Fix `ROUTE_NAMESPACES` (or the pages) for companions, home, ivf, postpartum, postpartum/setup, pregnancy, services; check each in the browser for raw keys.
2. Remove `continue-on-error` from vitest; add admin-web `npm test` + typecheck to CI.
3. Coverage as a report only: `go test -coverprofile` + `go tool cover -func` summary, `vitest --coverage` (v8) for both apps; upload as CI artifacts. No thresholds yet.

## Out of scope
- Coverage thresholds.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- vitest 0 failures in both apps and blocking in CI; coverage artifacts present on a CI run.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
