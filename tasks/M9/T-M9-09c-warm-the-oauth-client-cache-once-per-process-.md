---
id: T-M9-09c
title: Warm the OAuth client cache once per process (shared verifier per app)
milestone: M9
type: backend
status: todo
depends_on: [T-M9-09b]
parallel_group: M9-B
touches: [backend-go/internal/auth/guard.go,backend-go/internal/auth/guard_test.go,backend-go/internal/auth/passport]
skills: []
verify: cd backend-go && go vet ./internal/auth/... && go test -race ./internal/auth/...
---

# T-M9-09c — Warm the OAuth client cache once per process (shared verifier per app)

## Why
Split out of T-M9-09b, scope item 2. The auto-mode permission check refused an edit to `internal/auth/guard.go` ("Security Weaken"). **Needs an explicit user OK before it runs.** Today each of the ~73 guards runs one `ListClients` on its first request. The cost is small and the change is optional.

## Scope
1. `passport`: add `(*Verifier) WarmClients(ctx) error`, which runs one `ListClients` and fills the cache under the usual TTL.
2. `guard.go`: keep one shared `*passport.Verifier` per `*fiber.App` in `app.State()`, using the same pattern as `auth.Jobs`. On first build, warm it with a 5 s timeout; on failure, log it and fall back to lazy load. Only do this when `db != nil`.
3. No snapshot wrapper: the client-revoke delay must stay ≤ 5 min (D-140).

## Acceptance
- A test with a counting fake store: many guards on one app plus many Authenticate calls give exactly `ListClients == 1`.
- 401 codes and session rules unchanged; `make contract ROUTES=auth` stays 45/45.
