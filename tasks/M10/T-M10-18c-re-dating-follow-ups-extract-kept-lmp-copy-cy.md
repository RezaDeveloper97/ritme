---
id: T-M10-18c
title: Re-dating follow-ups: extract kept-LMP copy, cycle-length source, dating schema dedupe, goldens
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-18b]
parallel_group: M10-C
touches: [backend-go/internal/healthrecord/extract/handlers.go,backend-go/internal/healthrecord/extract/lang,backend-go/internal/http/routes_pregnancy.go,backend-go/internal/pregnancy/handlers.go,backend-go/contract/golden/pregnancy,backend-go/contract/golden/pregnancy-v2,backend-go/contract/allowlist/pregnancy.yaml,backend-go/contract/allowlist/healthrecord-extract.yaml,frontend/src/features/track-pregnancy,frontend/src/entities/pregnancy]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/pregnancy/... ./internal/healthrecord/... && make test-int PKG='./internal/pregnancy/... ./internal/healthrecord/...' && make contract ROUTES=all && cd ../frontend && npm run typecheck && npm run lint && npm run test
---

# T-M10-18c — Re-dating follow-ups: extract kept-LMP copy, cycle-length source, dating schema dedupe, goldens
**Model:** opus (sonnet is fine for the copy). D-169.

## Why
These T-M10-18b items lived outside its touches.

## Scope
1. Health-record extract "kept LMP" copy:
   - `extract/handlers.go:412` always sends `messages.dating_applied`, and `offerJSON` (`:343-366`, `offer.applied`) does too. When `o.Redating != nil && !o.Redating.Replaced`, use a new kept key instead, and send `o.Redating.JSON()` as `dating`.
   - New fa/en keys in `extract/lang`. fa proposal: `messages.dating_kept` «سونوگرافی ثبت شد. اختلافش با تاریخ آخرین پریودت کم بود، برای همین سن بارداری همان‌طور ماند.»
   - `offer.same_dating` in the kept case: «این سونوگرافی را قبلاً ثبت کرده‌ای».
   - Write the copy with the persian-writing skill.
2. `internal/http/routes_pregnancy.go:22`: `h.SetCycleLength(...)` with the effective cycle length from the cycle engine (no effect until `pregnancy.edd.cycle_correction` is enabled).
3. Frontend: `features/track-pregnancy` `pregnancyDatingSchema` duplicates `entities/pregnancy` `onboardingDatingSchema`; the feature re-exports the entity's.
4. Re-record the pregnancy and pregnancy-v2 goldens from Go so they carry `dating`. Then make `dating` `required` in `openapi.yaml` and drop the D-169 `data.dating` allow-list entries.
5. Product question to raise: onboarding compares against the profile's previous LMP even when it belongs to an earlier pregnancy.

## Note (parallel batch with T-M10-22)
`backend-go/api/openapi.yaml` is owned by T-M10-22 in this batch: write the openapi change for step 4 as a unified diff
under the session scratchpad; the main session applies it.

## Acceptance
- A kept-LMP extract shows the kept copy.
- `dating` is required and recorded.
- verify green.
