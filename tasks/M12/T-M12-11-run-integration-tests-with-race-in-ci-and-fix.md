---
id: T-M12-11
title: Run integration tests with -race in CI and fix local `make test` on this Mac
milestone: M12
type: fullstack
status: done
depends_on: []
parallel_group: M12-TEST
touches: [.github/workflows/claude-to-stage.yml, .github/workflows/ci.yml, backend-go/Makefile, backend-go/internal/platform/db/testdb]
skills: []
verify: cd backend-go && make test && make test-int PKG="./internal/ai/... ./internal/plus/... ./internal/auth/... ./internal/family/..."
---

# T-M12-11 — Run integration tests with -race in CI and fix local `make test` on this Mac
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-01: the 297 integration test files — every money, credits, auth and family test — skip silently in CI; nothing runs with the race detector; pushes to `stage`/`main` and PRs run no tests; local cgo linking fails.

## Scope
1. New `ci.yml` on push to `stage`, `main`, `claude/**` and on PRs: MariaDB 11.4 + Redis service containers, `TEST_DB_DSN`/`TEST_REDIS_ADDR`, `go test -race ./...` (full, integration included). `claude-to-stage.yml` auto-merge depends on it.
2. `testdb`: when `CI=true` and `TEST_DB_DSN` is empty, fail instead of skipping, so a broken service config cannot go green.
3. Makefile: `test` falls back to `CGO_ENABLED=0` when the cgo link fails (or document the Command Line Tools update); `test-race` target.
4. Fix any data race the detector finds (each one is a real bug — note it in PROGRESS.md).

## Out of scope
- Coverage thresholds (T-M12-12).
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- CI run on a branch shows integration packages ran (not skipped) and `-race` green; link to the run in PROGRESS.md.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
