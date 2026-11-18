---
id: T-M12-17b
title: Owner grant edit silently drops v6-only grants (urgent alert, vitals, household)
milestone: M12
type: backend
status: todo
depends_on: [T-M12-17]
parallel_group: M12-TEST
touches: [backend-go/internal/companion,backend-go/internal/family/grants.go,backend-go/internal/family/grants_test.go,backend-go/internal/http/routine_family_int_test.go]
skills: []
verify: cd backend-go && CGO_ENABLED=0 go test ./internal/family/... ./internal/companion/... && make test-int PKG="./internal/family/... ./internal/companion/... ./internal/http/..."
---

# T-M12-17b — Owner grant edit silently drops v6-only grants (urgent alert, vitals, household)

**Model:** opus. Found in T-M12-17 (2026-11-18); skipped test `TestRoutineFamily_GrantEditKeepsUrgentAlert`
(run with `FAMILY_KNOWN_BUGS=1`).

## Why
`PUT /companions/{id}/grants` (the only grant-edit path, also used by re-approve after a mode change) converts the
legacy sections with `toV6` into a *complete* v6 set and `family.Grants.Apply` deletes every row not in it
(`internal/companion/service.go` `SetGrants`/`applyGrants`, `internal/family/grants.go` `Apply`). So the first owner
edit silently turns off `urgent_alert`, `vitals`, `household`, `fetal_activity`, `ttc_fertility` granted by a v6
invite — a safety regression (F-23: the urgent-alert switch stays on).

## Scope
1. Grant edits through the legacy endpoint only change the domains that endpoint can express; v6-only domains are
   kept unless the request names them. Re-approve after a mode change keeps them too.
2. Unit test on `Apply`/`SetGrants` (v6-only grant survives a legacy edit; explicit removal still removes it) and
   remove the skip from `TestRoutineFamily_GrantEditKeepsUrgentAlert`.
3. Audit trail: the edit's audit row lists only what really changed.

## Acceptance
- Unskipped flow test green; unit tests red on the old code.
