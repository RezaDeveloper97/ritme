---
id: T-M9-09b
title: Auth follow-ups: contract clock for near-expiry token, client cache warm-up, DB pool env docs
milestone: M9
type: backend
status: done
depends_on: [T-M9-09]
parallel_group: M9-B
touches: [backend-go/cmd/contract,backend-go/internal/auth/guard.go,backend-go/.env.example,docs/investigations/perf-baseline.md,docs/go-migration/deviations.md]
skills: []
verify: cd backend-go && go vet ./... && go test ./cmd/contract/... ./internal/auth/... && make contract ROUTES=auth
---

# T-M9-09b — Auth follow-ups: contract clock for near-expiry token, client cache warm-up, DB pool env docs

## Why
T-M9-09 left four items outside its touches (see its PROGRESS.md section).

## Scope
1. `cmd/contract`: mint the `token_near_expiry` session against the contract clock (`DefaultNow`), not `time.Now()`, so `auth/refresh_session_due` passes again (it fails on HEAD since ~2026-10-13 because the 30-day window has passed).
2. Warm the OAuth client cache once at startup (one shared cache for all guards instead of one `ListClients` per guard on first use) in `internal/auth/guard.go`.
3. Document `DB_MAX_OPEN`, `DB_MAX_IDLE`, `DB_CONN_MAX_LIFETIME`, `DB_CONN_MAX_IDLE_TIME` in `backend-go/.env.example`.
4. Record in `docs/investigations/perf-baseline.md`: auth queries per request 3 → 1, Verify ≈ 106 µs/op (M4), interpolateParams = 1 round-trip per query. Add a deviation entry: revoking an OAuth *client* takes effect within ≤ 5 min (cache refresh); token revocation stays immediate.

## Acceptance
- `make contract ROUTES=auth` → 45/45.
- One `ListClients` per process at startup, verified by a test.
- `.env.example`, perf-baseline and deviations updated.
