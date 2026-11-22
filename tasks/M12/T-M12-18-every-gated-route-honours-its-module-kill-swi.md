---
id: T-M12-18
title: Every gated route honours its module kill-switch (table test over the router)
milestone: M12
type: backend
status: done
depends_on: [T-M12-11]
parallel_group: M12-TEST
touches: [backend-go/internal/http/modules_int_test.go, backend-go/internal/modules, frontend/src/entities/app-module]
skills: []
verify: cd backend-go && make test-int PKG="./internal/http/..." RUN=TestModules && cd ../frontend && npx vitest run src/entities/app-module
---

# T-M12-18 — Every gated route honours its module kill-switch (table test over the router)
**Model:** sonnet. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-10: only one module's 403 is tested; a new route added without the gate passes CI.

## Scope
1. Walk the Fiber route table; map each route group to its module (from the router registration, not a hand list). For every module: switch it off, call every route in its group → 403 with the managed code; switch on → not 403.
2. Fail when a route under a module prefix is not gated or a module has no routes.
3. Frontend: unit test that the home sections and entry points hide when `/modules` reports the module off.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Test green and fails on purpose when a gate is removed (show it in PROGRESS.md).
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
