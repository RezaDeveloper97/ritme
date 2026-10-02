---
id: B-N3-14b
title: N3 stage smoke follow-ups (past-day bleeding period split, late-period hero, LH/BBT sync, small UI)
milestone: N3
type: fullstack
status: todo
depends_on: [B-N3-14]
parallel_group: N3-O2
touches: [backend-go/internal/healthlog,backend-go/internal/cycle,backend-go/internal/fertility,backend-go/internal/analysis,backend-go/internal/voicelog,frontend/src/screens/analysis,frontend/src/screens/analysis-ttc,frontend/src/screens/analysis-pregnancy,frontend/src/features/log-day]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-14b — N3 stage smoke follow-ups (past-day bleeding period split, late-period hero, LH/BBT sync, small UI)

## Why
Bugs from the N3 stage smoke (`docs/qa/bloom/n3-stage.md`).

## Scope
- **B-1 (high):** logging bleeding on a past day whose previous day has no bleeding always creates a new period — even inside a confirmed period — producing an open unconfirmed row with negative `cycle_length`, shortening the last period and moving the profile LMP back (`internal/healthlog/cyclehistory.go` `checkAndUpdatePeriodStart`, a Laravel port). Fix: a bleeding day inside or adjacent (±1–2 days) to an existing period extends/merges it; never negative lengths; regression tests with the repro (POST /cycle/period 06-01..05, then PUT /logs/days 01,02,04). Note any Laravel-parity deviation.
- **B-2 (medium):** late user logs bleeding in the log sheet → calendar/calculation «روز ۱» but home hero (`cycle_view`) still «تأخیر ۹ روز» because the resolver ignores the unconfirmed row (`internal/cycle/resolver` `anchorFor`). Make today's bleeding start count for the hero.
- **B-3 (medium):** LH/BBT logged in the log sheet don't reach `fertility_logs` → `/fertility/today` `lh:null`. Sync taxonomy v2 LH/BBT/mucus into the fertility path (or make fertility read v2) consistently with the TTC analysis.
- Lows: B-5 menopause hub asks to «log 3 more cycles»; B-6 pregnancy weight card week chip vs text off by one; B-7 `/analysis/fertility` subtitle bidi; B-8 voice labels Latin digits in fa; B-9 teen log sheet shows the Plus badge on the voice tab.
- (B-4 postpartum home = B-N5-04 scope.)

## Out of scope
- 

## Acceptance
- Each bug fixed with tests (B-1/B-2/B-3 integration tests), contract goldens updated only where behaviour intentionally changes
- `verify` green
