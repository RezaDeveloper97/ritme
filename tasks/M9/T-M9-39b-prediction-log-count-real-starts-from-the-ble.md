---
id: T-M9-39b
title: Prediction log: count real starts from the bleeding-log bridge (healthlog StartHook)
milestone: M9
type: backend
status: done
depends_on: [T-M9-39]
parallel_group: M9-C
touches: [backend-go/internal/healthlog,backend-go/internal/http/routes_healthlog.go,backend-go/internal/cycle/snapshots,backend-go/internal/http/routes_logs.go]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/healthlog/... ./internal/cycle/snapshots/... && make test-int PKG="./internal/healthlog/... ./internal/cycle/snapshots/..." && make contract ROUTES=healthlog && make contract ROUTES=logs
---

# T-M9-39b — Prediction log: count real starts from the bleeding-log bridge (healthlog StartHook)

## Why
T-M9-39 fires `periods.StartHook` only from `periods.Service.Start/Store`. `internal/healthlog` also creates real
period starts through the bleeding-log bridge but has no hook, so those cycles never get `last`/`actual_start`/
`error_days` in `engine_prediction_log` and the engine MAE (T-M10-30) would miss them.

## Scope
- Fire the same `StartHook` (after commit, ids + `rev_before`/`rev_after` only) when the bleeding-log bridge creates a
  new real start or promotes an estimate/seed. Wire it in the healthlog routes file like `routes_cycle.go`.
- Check the routes file name with `ls backend-go/internal/http/routes_*` and fix `touches` if it differs.

## Out of scope
- KPI computation (T-M10-30).

## Acceptance
- Int test: a bleeding log that starts a new period produces `last` + `actual_start`/`error_days` + `first`; a
  bleeding log inside an existing period produces nothing. Contract for health-logs unchanged; verify green.
