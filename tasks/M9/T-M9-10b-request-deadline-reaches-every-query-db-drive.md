---
id: T-M9-10b
title: Request deadline reaches every query (DB driver reads httpx.DeadlineFrom)
milestone: M9
type: backend
status: todo
depends_on: [T-M9-10]
parallel_group: M9-B
touches: [backend-go/internal/platform/db,backend-go/internal/platform/httpx]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/platform/... && make test-int PKG=./internal/platform/db/...
---

# T-M9-10b — Request deadline reaches every query (DB driver reads httpx.DeadlineFrom)

## Why
T-M9-10 put a 5 s deadline on `c.Context()`. But about 457 service calls pass the `fiber.Ctx` itself as `context.Context`, against about 93 that pass `c.Context()`. `fiber.Ctx` never carries a deadline, so most queries still run without one.

## Scope
1. In `platform/db`'s instrumented connector (`instrument.go`), resolve the context with `httpx.DeadlineFrom(ctx)`. That function already finds the bounded context stored in Locals when it is handed a `fiber.Ctx`. Every query is then bounded, with no change at the ~457 call sites. Watch for an import cycle (`httpx` → `db`?); if there is one, move the lookup key into a tiny shared package.
2. Long-running routes (`httpx.LongRunning`) keep running without the deadline.

## Acceptance
- An integration test: a handler that passes `c` (not `c.Context()`) and runs `SELECT SLEEP(6)` is cancelled at about 5 s.
- A timeout gives 500, never 401.
