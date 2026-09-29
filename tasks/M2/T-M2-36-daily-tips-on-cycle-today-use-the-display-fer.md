---
id: T-M2-36
title: Daily tips on /cycle/today use the display fertile window (F-1)
milestone: M2
type: backend
status: done
depends_on: [T-M2-35]
parallel_group: M2-J
touches: [backend-go/internal/cycle,backend-go/internal/messages,backend-go/api/openapi.yaml,backend-go/contract/allowlist,docs/go-migration/deviations.md,docs/qa,frontend/src/screens/home]
skills: [verify-all]
verify: cd backend-go && go vet ./... && go test ./... && golangci-lint run && make contract ROUTES=all && cd ../frontend && npm run typecheck && npm run test
---

# T-M2-36 — Daily tips on /cycle/today use the display fertile window (F-1)

## Why
Final stage smoke F-1 (docs/qa/stage-final-smoke-2026-09-29.md): on O+1 an avoiding user's home «توصیه‌های امروز»
still says "peak fertility / best time" and shows the ovulation energy tip, because `GET /cycle/today`
`calculation.daily_tips` (and `calculation.is_fertile_window`/phase) come from the legacy engine
(`internal/cycle/legacy/engine.go`), while D-28 only fixed `/messages/daily`.

## Scope
- Apply the task.md §19 display window (end on ovulation day; O+1 = luteal, not fertile) to the tips and flags the
  home reads from `/cycle/today` — reuse the D-28 helper, don't duplicate it. Extend D-28 (or add D-nn) in
  deviations.md + contract allowlist for the affected `/cycle/*` cases; golden sweep test applies the same rewrite.
- Unit + int test for the O+1 case (avoiding and trying users).

## Acceptance
- On O+1 no fertile/peak copy anywhere on the home; verify green (incl. contract).
