---
id: T-M9-48d
title: Engine v6: hormonal suppression only from the method start date
milestone: M9
type: backend
status: done
depends_on: [T-M9-48]
parallel_group: M9-R
touches: [backend-go/db/queries/cycle, backend-go/internal/cycle/store, backend-go/internal/cycle/inputs, backend-go/internal/labs/engine_test.go, backend-go/internal/cycle/engine,backend-go/internal/cycle/state,backend-go/internal/cycle/service,docs/engine-v6/DECISIONS.md]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... && make test-int PKG=./internal/cycle/...
---

# T-M9-48d — Engine v6: hormonal suppression only from the method start date

## Why
Stage shadow (rollout.md): `Input.Hormonal` is a single bool, so pill suppression covers all history (persona 53 pack started 2026-10-23, yet days back to her first period are suppressed). User decision 2026-11-02: suppress only from the method's start date.

## Scope
- Carry the active hormonal method's start date into the engine input (replace/extend `Input.Hormonal`).
- E-13 suppression applies only to days on/after that date; earlier cycles resolve normally.
- No start date known → current behaviour (whole history), note it.
- Record the decision in docs/engine-v6/DECISIONS.md (amend E-13).
- Tests + goldens (explain each diff).

## Out of scope
- Ending suppression when a method stops (already handled or separate).

## Acceptance
- Pill persona: days before 2026-10-23 show normal phases; from that date suppressed.
- DECISIONS.md updated.
- verify green. 

## Decision (2026-11-02)
Start date = min(contraception_methods.created_at, the method's own date column if set: pill pack_started_on / hormonal_iud inserted_on / injection injected_on). Onboarding-medications path has no date → whole history (unchanged). Snapshot L2 format version 1 → 2.
