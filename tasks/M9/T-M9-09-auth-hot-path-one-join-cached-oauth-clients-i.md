---
id: T-M9-09
title: Auth hot path: one JOIN, cached OAuth clients, interpolateParams, pool from env
milestone: M9
type: backend
status: todo
depends_on: [T-M9-07]
parallel_group: M9-B
touches: [backend-go/internal/auth/passport, backend-go/internal/auth/store, backend-go/db/queries/auth, backend-go/internal/platform/db, backend-go/internal/platform/config]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/auth/... ./internal/platform/... && make test-int PKG=./internal/auth/... && make contract ROUTES=auth
---

# T-M9-09 — Auth hot path: one JOIN, cached OAuth clients, interpolateParams, pool from env
**Model:** opus (security-sensitive).

## Why
Every authed request costs 3 queries (`GetAccessToken`, `GetClient`, `GetUserByID`, `auth/passport/verify.go:118,
128,143`) before any handler runs, and the DSN has no `interpolateParams`, so each parameterised query is prepare +
execute + close (3 round-trips). This is the largest fixed cost in the API (gap 07 §4.0, §7.3). T-M9-31 will extend the
same single query with the user's engine revision, so a warm engine GET becomes one query in total.

## Scope
1. One sqlc query joining `oauth_access_tokens` and `users` (token by id, user columns needed by the middleware),
   replacing the two separate reads. Keep every 401 code and message exactly (`token_revoked`, `token_expired`,
   `unauthenticated`, blocked and deleted users).
2. `oauth_clients` held in an in-process map: loaded at start, refreshed every 5 min, reloaded once on an unknown id.
3. DSN `interpolateParams=true` (verify charset/collation make it safe); pool from env: `DB_MAX_OPEN` (25),
   `DB_MAX_IDLE` (= max open), `DB_CONN_MAX_LIFETIME` (5m), `DB_CONN_MAX_IDLE_TIME` (2m).
4. No token-validity cache (DECISIONS.md X-02: revocation and the engine revision must be read fresh).

## Tests
- Session regression suite: 365-day sliding session, refresh window, a revoked token rejected on the very next request,
  blocked user, deleted user, expired token.
- Query budget: the auth step of an authed GET costs exactly 1 query (update `internal/querybudget`).
- `make contract ROUTES=auth` green; DSN builder unit test; `Verify` bench ≤ 150 µs.

## Out of scope
- The engine revision column in the query (T-M9-31).

## Acceptance
- 1 auth query per request; contract and session tests green; perf-baseline.md updated with the new numbers.
