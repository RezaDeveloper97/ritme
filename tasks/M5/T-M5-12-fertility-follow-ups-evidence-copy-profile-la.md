---
id: T-M5-12
title: Fertility follow-ups — evidence copy, profile latency, per-cycle period length, window anchors
milestone: M5
type: backend
status: in_progress
depends_on: [T-M5-11]
parallel_group: M5-D
touches: [backend-go/internal/fertility,backend-go/internal/profile,backend-go/internal/cycle,backend-go/db/queries,backend-go/api/openapi.yaml,backend-go/contract,frontend/src/screens/fertility-insights,frontend/src/screens/home,frontend/src/entities/fertility,frontend/src/entities/cycle,frontend/src/widgets/fertility-tiles,docs/fertility-ttc,docs/go-migration]
skills: [verify-all,new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && golangci-lint run && make contract ROUTES=all && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test
---

# T-M5-12 — Fertility follow-ups — evidence copy, profile latency, per-cycle period length, window anchors

## Why
Open items of T-M5-11 (tasks/PROGRESS.md § T-M5-11, docs/fertility-ttc/design-audit.md #18).

## Scope
1. Audit #18: insights evidence titles per design (backend copy in `internal/fertility/lang/*` / `insights.go`), drop
   the «(±۰)» parentheses. Tests.
2. `/profile` latency: the home now waits on `GET /profile`, and ~5 s responses were seen on stage. Measure it locally
   with realistic data (and read the handler/queries): find the slow part (N+1, missing index, cold cache,
   synchronous side work) and fix it; target p95 < 300 ms locally. If the fix needs no backend change, make the home
   render from the cached profile (TanStack Query persisted/initial data) so a slow refetch doesn't block it.
3. History strips: add the per-cycle period length to `/fertility/insights` history (`period_days`), OpenAPI +
   contract (Go-only route); frontend uses it instead of the effective length.
4. Home schedule vs `/fertility/bbt` fertile window off by one day: find which anchor is right per task.md (cycle
   engine v1.1 spec) and make both use the same source. Test with the cycle-day-16 case.

## Acceptance
- All 4 done with tests; latency numbers before/after recorded in docs/fertility-ttc/README.md; verify green.
