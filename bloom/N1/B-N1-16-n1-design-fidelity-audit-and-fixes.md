---
id: B-N1-16
title: N1 design-fidelity audit and fixes
milestone: N1
type: frontend
status: done
depends_on: [B-N1-05,B-N1-06,B-N1-07,B-N1-08,B-N1-09,B-N1-10,B-N1-11,B-N1-12,B-N1-13,B-N1-14,B-N1-15]
parallel_group: N1-P
touches: [docs/night-bloom/audit-n1.md,frontend/src,frontend/messages,backend-go/resources/translations]
skills: [verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-16 — N1 design-fidelity audit and fixes

## Why
Every milestone ends with a side-by-side audit, as in M3–M7.

## Scope
- Screenshot every N1 screen light + dark (headless, see memory «Headless UI verification»), compare with the artboards, list deviations by severity with file:line in `docs/night-bloom/audit-n1.md`, fix all high/med, write a Resolution per row.

- Leftover from B-N1-02: components still using `--on-accent` on a brand fill (white on light-lavender in dark) — onboarding checks, DayTasks, TodayChallenge, PeriodButton, DailyStatusCard, IntroIllustration, DayLogPage — switch to `--on-brand` if their screen task did not.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Audit table complete with Resolution column
- verify green
- `verify` green
