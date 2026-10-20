---
id: T-M9-04
title: Period writes: per-user lock, idempotent start, Idempotency-Key
milestone: M9
type: backend
status: todo
depends_on: []
parallel_group: M9-A
touches: [backend-go/internal/cycle/periods, backend-go/internal/fertility/service.go, backend-go/internal/platform/idempotency, backend-go/internal/http/routes_cycle.go, backend-go/internal/http/routes_fertility.go, backend-go/internal/http/routes_healthlog.go]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... ./internal/platform/... && make test-int PKG="./internal/cycle/periods/... ./internal/platform/idempotency/..."
---

# T-M9-04 — Period writes: per-user lock, idempotent start, Idempotency-Key
**Model:** opus.

## Why
B2 (gap 07 §6): period writes run in a REPEATABLE READ transaction with plain consistent reads, so two concurrent
`POST /cycle/period/start` (double tap, two devices, offline replay) both pass `overlapsExisting` and both insert.
The same date hits `UNIQUE(user_id, period_start_date)` → error 1062 → **500** (`periods/service.go:94-166,
202-247,377-398,437-448`). The engine must never see overlapping real periods (they become
`contradictory_anchor_data`, E-02).

## Scope
1. `periods.Service.tx` (`periods/service.go:450`) takes a per-user lock first: `SELECT id FROM users WHERE id = ?
   FOR UPDATE` (the `users` row always exists; `user_profiles` may still have duplicates until T-M9-23). All overlap
   and blocker checks then run serialised per user.
2. Map MySQL 1062 on the period insert: same start date → re-read and return the existing period (idempotent 200);
   anything else → the existing 422 overlap message. Never 500.
3. New `internal/platform/idempotency`: optional `Idempotency-Key` header on `POST /cycle/period*`, the health-log
   save routes and `PUT /fertility/days/{date}`. Redis key `ritme-go:idem:{uid}:{sha256(key)}` with a `SET NX` lock,
   stores status + body for 24 h; a replay returns the stored response; the same key with a different body → 422
   `idempotency_key_reused`; Redis down → process normally. The key never appears in logs.
4. Check the fertility day upsert for the same race; keep the health-log createOrFirst path.
5. Record in PROGRESS.md that the frontend offline outbox should send `Idempotency-Key` (follow-up, not here).

## Tests
- Integration: 20 goroutines × `POST /cycle/period/start` with the same date → exactly one row, every response 200/201,
  no 500; 20 goroutines with different dates inside one period range → no overlapping rows, the losers get 422.
- Replay with the same key → byte-identical response, no new row; same key + different body → 422; Redis stopped →
  the request still succeeds.
- Unit: middleware key hashing, TTL, lock release on handler error.

## Out of scope
- The bleeding-log bridge rewrite (T-M9-28), the frontend outbox.

## Acceptance
- No 500 on concurrent or replayed period writes; tests green; PROGRESS.md cites D-131.
