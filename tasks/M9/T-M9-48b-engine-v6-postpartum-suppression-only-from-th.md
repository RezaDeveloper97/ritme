---
id: T-M9-48b
title: Engine v6: postpartum suppression only from the delivery date
milestone: M9
type: backend
status: done
depends_on: [T-M9-48]
parallel_group: M9-R
touches: [backend-go/internal/cycle/engine]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... && make test-int PKG=./internal/cycle/...
---

# T-M9-48b — Engine v6: postpartum suppression only from the delivery date

## Why
Stage shadow (docs/engine-v6/rollout.md, Bugs found): `postpartumPeriods` in engine/limits.go never checks whether the day is before the delivery date. Persona 54 gets `postpartum_cycles_not_returned` + suppression on all days incl. before birth; finished pre-pregnancy cycles go blank.

## Scope
- In engine/limits.go apply E-14 postpartum suppression only for days on/after the delivery date.
- Golden/unit test: pre-delivery days of a postpartum fixture keep their normal phases and cycle days.
- Regenerate v6 goldens if needed (`UPDATE_GOLDEN_V6=1`) and explain each diff.

## Out of scope
- Other diff classes (all expected per rollout.md).

## Acceptance
- Days before delivery show no postpartum warning or suppression.
- Days on/after delivery unchanged.
- verify green. 
