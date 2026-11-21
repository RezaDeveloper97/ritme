---
id: T-M12-28
title: Lazy-load the autopilot catalog, voice-log stack and mode-specific home branches
milestone: M12
type: frontend
status: done
depends_on: [T-M12-27]
parallel_group: M12-PERF
touches: [frontend/src/app/autopilot/AutopilotMount.tsx, docs/quality-2026-11/perf.md, frontend/src/shared/autopilot, frontend/src/screens/home/ui/HomePage.tsx, frontend/src/screens/log, frontend/src/widgets/log-sheet, frontend/scripts/check-bundle-size.mjs]
skills: []
verify: cd frontend && npm run typecheck && npm test && npm run build && node scripts/check-bundle-size.mjs
---

# T-M12-28 — Lazy-load the autopilot catalog, voice-log stack and mode-specific home branches
**Model:** sonnet. Evidence: `docs/quality-2026-11/findings.md`.

## Why
P-02/P-03: ~300 KB of autopilot code and the voice/AI stack load on every visit though rarely used; every user downloads menopause, teen and editor code on home.

## Scope
1. AutopilotMount: tiny idle listener; `import()` catalog + engine when a script starts.
2. `VoiceLogGated` loaded on orb press; `MenopauseHome`, `TeenHome`, `LinkedTeenCards`, `HomeLabCards`, `QuickEditSheet`, `PeriodDateEditor`, `SkipPeriodSheet` via `next/dynamic` with skeletons that keep layout stable.
3. Autopilot catalog test still sees every name (catalog is still statically checkable in tests).
4. Lower the first-load JS budget to the new number.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- First-load JS of /fa/home and /fa/log reduced (numbers in perf.md); voice log and autopilot still work (manual check).
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
