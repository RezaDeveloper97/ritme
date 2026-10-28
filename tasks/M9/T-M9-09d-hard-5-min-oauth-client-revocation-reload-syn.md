---
id: T-M9-09d
title: Hard ≤5 min OAuth client revocation: reload synchronously once the client snapshot is past its TTL
milestone: M9
type: backend
status: todo
depends_on: [T-M9-09c]
parallel_group: M9-B
touches: [backend-go/internal/auth/passport]
skills: []
verify: cd backend-go && go vet ./internal/auth/... && go test ./internal/auth/... && make contract ROUTES=auth
---

# T-M9-09d — Hard ≤5 min OAuth client revocation: reload synchronously once the client snapshot is past its TTL

## Why
Security review of T-M9-09c (Low, pre-existing): `clientCache.lookup` (`internal/auth/passport/clients.go:63-64`) serves
the old snapshot and only starts an async refresh when the snapshot is older than `ClientCacheTTL`. If `ListClients`
keeps failing, or on the first request after a long idle (now served from the boot-time warm snapshot), a revoked OAuth
client can still authenticate more than 5 minutes after revocation. D-140 says the delay must stay ≤ 5 min.

## Scope
- In `lookup`: within the TTL keep the async refresh; once age ≥ TTL reload synchronously (`loadIfOlder`) and fail
  closed (500, never a 401 — session rules) if the reload errors.
- Optional: document on `MustGuard` that the first call on an app fixes cfg/db for the shared verifier.
- Touches auth — needs the user's OK before it runs (same as T-M9-09c).

## Out of scope
- Token caching (tokens stay read fresh per request).

## Acceptance
- Tests: stale snapshot + revoked client → rejected on the first request; reload error past TTL → 500 not 401;
  within TTL no extra query. Contract auth 45/45; verify green.
