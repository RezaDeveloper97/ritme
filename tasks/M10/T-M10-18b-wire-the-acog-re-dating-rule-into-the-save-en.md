---
id: T-M10-18b
title: Wire the ACOG re-dating rule into the save endpoints and remaining trimester callers
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-18]
parallel_group: M10-C
touches: [backend-go/internal/pregnancy/handlers.go,backend-go/internal/healthrecord/extract/dating.go,backend-go/internal/healthrecord/extract/extract_int_test.go,backend-go/internal/analysis/pregnancy.go,backend-go/internal/pregnancy/v2/service.go,backend-go/internal/pregnancy/v2/calendar/calendar.go,backend-go/internal/pregnancy/v2/preview.go,backend-go/internal/clinicalconfig/coverage_pin_test.go,backend-go/api/openapi.yaml,backend-go/contract/allowlist/pregnancy.yaml,backend-go/contract/allowlist/healthrecord-extract.yaml,backend-go/contract/allowlist/home.yaml,backend-go/contract/allowlist/messages.yaml,backend-go/contract/allowlist/loss.yaml,frontend/src/entities/pregnancy,frontend/src/screens/pregnancy-onboarding]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/pregnancy/... ./internal/healthrecord/... ./internal/analysis/... ./internal/clinicalconfig/... && make test-int PKG='./internal/pregnancy/... ./internal/healthrecord/...' && make contract ROUTES=all && cd ../frontend && npm run typecheck && npm run lint && npm run test
---

# T-M10-18b — Wire the ACOG re-dating rule into the save endpoints and remaining trimester callers
**Model:** opus. D-169.

## Why
T-M10-18 built the single dating function (`calc/dating.go`: `TrimesterOn`, `EDDFromLMP`, `Redate`), `pregnancy.ApplyScanRule` and `ApplyUltrasoundDatingByRule`. The callers outside its touches still use the old paths, so the re-dating rule never runs on a real save.

## Scope
1. `internal/pregnancy/handlers.go`:
   - Onboarding (`:191-243`) and Update (`:259-305`) call `ApplyScanRule(existing, validated, explicit)` before `saveProfile`; explicit = `age_source` sent in the request.
   - Use `cl.WithCycleLength(...)` for the optional cycle correction.
   - Return `dating` in the Onboarding response.
2. `internal/healthrecord/extract/dating.go:254`: use `ApplyUltrasoundDatingByRule`. Update `extract_int_test.go:395-404` (a 2-day difference at 12+3 keeps the LMP) and the copy «سن بارداری از همین سند به‌روز شد» for the kept case.
3. Switch the remaining callers to the dated functions, so they read the registry version of the day:
   - `internal/analysis/pregnancy.go:71` → `calc.TrimesterOn`;
   - `v2/service.go:146-152` → `d.TrimesterStartDays()`, `:195` → `d.TrimesterOfWeek(w)`;
   - `:271` and `v2/calendar/calendar.go:308` → `WeekLabelOn(..., today, ...)`;
   - `v2/preview.go`: the dating-preview returns the `dating` block.
4. Remove the v1 wrappers `calc.Trimester(weeks)` and `v2.WeekLabel` once nothing calls them.
5. `clinicalconfig/coverage_pin_test.go:97`: pin `pregnancy.edd` to calc reading the key, not to the `AddDays(280)` string in a comment.
6. Frontend: `entities/pregnancy/api/queries.ts:398-414` `useCompleteOnboarding` passes `dating` through. Show `DatingResultLine` (features/track-pregnancy) after a scan is saved on a profile with an LMP. Fix the comment `model/v2-types.ts:238` (0/35/70).

## Acceptance
- Saving a scan through onboarding/PUT profile applies the rule and returns `dating`.
- No caller of the v1 trimester/label wrappers remains.
- verify green; contract differences allow-listed under D-169 only.
