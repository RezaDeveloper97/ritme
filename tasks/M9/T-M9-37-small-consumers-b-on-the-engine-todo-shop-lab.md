---
id: T-M9-37
title: Small consumers B on the engine: todo, shop, labs, conditions, search, nudges
milestone: M9
type: backend
status: todo
depends_on: [T-M9-32]
parallel_group: M9-C
touches: [backend-go/internal/todo, backend-go/internal/shop/checklist, backend-go/internal/http/routes_shop_checklist.go, backend-go/internal/labs/context.go, backend-go/internal/labs/service.go, backend-go/internal/conditions, backend-go/internal/messages/conditionnudges, backend-go/internal/search, backend-go/internal/healthlog/cyclehistory.go, backend-go/db/queries/conditions]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/todo/... ./internal/shop/... ./internal/labs/... ./internal/conditions/... ./internal/messages/conditionnudges/... ./internal/search/... ./internal/healthlog/... && make test-int PKG="./internal/todo/... ./internal/shop/... ./internal/labs/... ./internal/conditions/..." && make contract ROUTES=todo,shop-checklist,labs,conditions,search
---

# T-M9-37 — Small consumers B on the engine: todo, shop, labs, conditions, search, nudges
**Model:** opus (labs and conditions touch health interpretation).

## Why
Four different "current cycle is stale" rules (45, 60, 4·L days, none), declared instead of effective lengths, raw
stored mode instead of the resolved one, a lab phase computed for *today* with its own off-by-one ovulation and stored
forever, and a shop reminder that goes stale after a period edit (gap 02 §5, §7). DECISIONS.md I-10.

## Scope
In v6/shadow mode (v1 unchanged):
1. Todo and shop gate on the resolved life mode and the engine `limits` instead of the raw stored mode.
2. Shop supply reminder: store the `engine_rev` it was computed from; resync on read when the revision moved
   (I-10).
3. Labs: the phase for the lab's `taken_on` date from the engine, stored labelled with that date; delete
   `labs.phaseOn`.
4. Conditions (PMDD, PBAC), condition nudges and search: cycle bounds from the engine's actual periods and
   `predicted_next`; delete the 45-day/60-day/profile-length rules; skipped cycles respected.
5. `healthlog.IsLikelyLutealSpotting` reads the engine phase.
6. `todo_suggestion_events.ref_date` unchanged (documented in PROGRESS.md).
7. `LocalToday` where you touch "today".

## Tests
- Shop: a period edit moves the stored reminder on the next read; pregnant user with no stored mode → no offer.
- Labs: a lab taken 2026-07-10 stores the phase of 2026-07-10 (not today's).
- PMDD/PBAC: a back-dated 1-day bleeding period no longer splits a cycle; a skip is respected.
- Search/nudges: 15 days late → no "current cycle" window.

## Out of scope
- Message rules (T-M9-38).

## Acceptance
- Tests green; PROGRESS.md cites D-136.
