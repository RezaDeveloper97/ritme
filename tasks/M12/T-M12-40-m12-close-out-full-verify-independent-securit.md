---
id: T-M12-40
title: M12 close-out: full verify, independent security re-audit, stage deploy, PROGRESS summary
milestone: M12
type: release
status: todo
depends_on: [T-M12-01, T-M12-02, T-M12-03, T-M12-04, T-M12-05, T-M12-06, T-M12-07, T-M12-08, T-M12-11, T-M12-12, T-M12-13, T-M12-14, T-M12-15, T-M12-16, T-M12-17, T-M12-18, T-M12-19, T-M12-20, T-M12-21, T-M12-22, T-M12-23, T-M12-25, T-M12-27, T-M12-28, T-M12-29, T-M12-30, T-M12-31, T-M12-32, T-M12-33, T-M12-34, T-M12-35]
parallel_group: M12-REL
touches: [docs/quality-2026-11/findings.md, tasks/PROGRESS.md]
skills: [verify-all, deploy-stage]
verify: grep -q "## Resolution" docs/quality-2026-11/findings.md
---

# T-M12-40 — M12 close-out: full verify, independent security re-audit, stage deploy, PROGRESS summary
**Model:** main session; security-auditor (opus) for the re-audit. Evidence: `docs/quality-2026-11/findings.md`.

## Why
Close the loop: every finding has a resolution and nothing regressed.

## Scope
1. /verify-all green (with integration + race).
2. A fresh security-auditor run on the M12 diff (not anchored on our conclusions: give it the code, not the findings).
3. Add a `## Resolution` section to findings.md: each S-/P-/T- row → fixed (commit), accepted (user OK), or moved (task id).
4. Deploy to stage; beta only with the user's OK.

## Out of scope
- Production (T-M12-09 stays blocked).
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- All rows resolved; stage smoke green.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
