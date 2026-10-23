---
id: T-M9-28
title: Bleeding-log bridge v6, postpartum flow and breastfeeding switch
milestone: M9
type: fullstack
status: done
depends_on: [T-M9-01, T-M9-04, T-M9-24]
parallel_group: M9-C
touches: [backend-go/internal/healthlog, backend-go/db/queries/healthlog, backend-go/internal/cycle/periods, backend-go/internal/cycle/model, backend-go/internal/cycle/service/adapters.go, backend-go/internal/postpartum, backend-go/db/queries/postpartum, backend-go/internal/clinicalconfig, backend-go/internal/pregnancy/alerts, backend-go/db/migrations, backend-go/api/openapi.yaml, backend-go/contract/allowlist, frontend/src/screens/postpartum, frontend/src/entities/postpartum, frontend/messages/fa/postpartum.json, frontend/messages/en/postpartum.json]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && go test ./internal/healthlog/... ./internal/cycle/... ./internal/postpartum/... ./internal/clinicalconfig/... && make test-int PKG="./internal/healthlog/... ./internal/cycle/periods/... ./internal/postpartum/... ./db/migrations/..." && make contract ROUTES=healthlog,logs,period,fertility && cd ../frontend && npm run typecheck && npm run lint && npm run lint:dark && npm run test
---

# T-M9-28 — Bleeding-log bridge v6, postpartum flow and breastfeeding switch
**Model:** opus (health data). Runs alone in a batch (data migration).

## Why
§1.4: logging a flow intensity starts a period; v6 treats user-logged periods as actual data. Today bridge rows are
unconfirmed and only the newest is promoted (older ones never count), flow logged in pregnancy creates periods and
moves the LMP, deleting a log never shrinks the period, back-dated inserts leave the next row's `cycle_length` stale,
and postpartum users cannot log a returning period (gap 02 §3 B2–B8). DECISIONS.md I-02, I-04, I-05, H-14.

## Scope
1. Bridge writes `source = bleeding_log`, `is_confirmed = 1`; the end rule is unchanged (newest period open until the
   user confirms an end or the next period starts, then closed at its last logged bleeding day — I-02).
2. Rebuild on removal: deleting a log or a day, or clearing `flow`, rebuilds the affected `bleeding_log` period from the
   remaining bleeding days with the same adjacency rule as the forward path (shrink, split or delete). Rows created by
   the period button (`user_logged`) are never touched by the bridge.
3. Every bridge write runs in one transaction with the T-M9-04 per-user lock and calls the shared (now exported)
   `periods.RecomputeCycleLengths` and `reanchor`.
4. Mode guard: in effective mode pregnancy a `flow` log creates no period and does not move the LMP; it raises the
   pregnancy bleeding alert (find the existing path in `internal/pregnancy/alerts` / v2 rules; if none exists, add the
   alert with copy reviewed by the main session).
5. Postpartum: `flow` is offered in postpartum mode from `delivery + postpartum.flow_from_day` (new registry key, group
   pregnancy, default 42, advisor-pending H-14); earlier bleeding stays lochia. After that day the bridge works as in
   cycle mode.
6. `PUT /cycle/period/{id}` marks the row confirmed `user_logged` (gap 02 S5). `PromoteLoggedStart` and its call in
   `cycle/service/adapters.go` are removed (no unconfirmed rows remain).
7. Breastfeeding: `breastfeeding` read/write on the postpartum profile endpoint (OpenAPI), and a switch on the
   postpartum screen «شیر می‌دهی؟» (fa/en, light/dark, RTL); null = not asked.
8. Goose **data** migration: unconfirmed, non-estimated `user_logged` rows → `bleeding_log`, confirmed; bump
   `user_revisions` for affected users; `down` reverts by source.
9. Update `TestStoreSideEffectsMatchLaravel` (`applyD55`) and the contract allow-lists (D-124, D-138).

## Tests
Table-driven integration tests:
- forward start; extending an open and a closed period; a back-dated day between two periods (the next row's
  `cycle_length` recomputed); deleting the only bleeding day (period deleted); clearing the middle day of a 5-day period
  (split only when the gap exceeds adjacency); clearing the last day (end shrinks); spotting creates nothing;
- pregnancy mode: flow → no row, LMP unchanged, alert raised once;
- postpartum: day 41 → `flow` not offered/accepted (422 or ignored per taxonomy contract), day 42 → creates a period;
- the period button's row is never modified by a bridge rebuild;
- `PUT /cycle/period/{id}` on a bridge row → confirmed `user_logged`;
- 20 concurrent saves of overlapping flow days → no overlapping periods (lock);
- migration up/down; revision bumped for converted users.
- Frontend vitest for the breastfeeding switch mapper; light/dark screenshots of the postpartum screen.

## Out of scope
- Revision bumps on other writers (T-M9-30), the log sheet UI beyond what the taxonomy already drives.

## Acceptance
- Tests green; PROGRESS.md cites D-124 and D-138 and lists H-14 as advisor-pending.
