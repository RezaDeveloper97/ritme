---
id: T-M9-48c
title: Engine v6: data_quality stays good on deliberately suppressed days
milestone: M9
type: backend
status: done
depends_on: [T-M9-48]
parallel_group: M9-R
touches: [backend-go/internal/cycle/engine, backend-go/internal/messages/manager/testdata/golden-v6]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... && make test-int PKG=./internal/cycle/...
---

# T-M9-48c — Engine v6: data_quality stays good on deliberately suppressed days

## Why
Stage shadow (rollout.md, Bugs found): quality.go requires phase != unknown for `good`, but E-13/E-14 make unknown deliberate on suppressed days. Pill persona flips good→partial on 144 days.

## Scope
- In engine quality.go, don't downgrade data_quality for days whose phase is unknown because of deliberate suppression (hormonal/postpartum).
- Unit test for pill and postpartum fixtures.
- Regenerate goldens if needed and explain each diff.

## Out of scope
- Other diff classes (all expected per rollout.md).

## Acceptance
- Suppressed days keep the data_quality they would have without suppression.
- verify green. 
