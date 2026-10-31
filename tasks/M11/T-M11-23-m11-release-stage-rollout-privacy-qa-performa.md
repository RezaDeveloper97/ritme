---
id: T-M11-23
title: M11 release: stage rollout, privacy QA, performance check and deviation reconciliation
milestone: M11
type: release
status: blocked
depends_on: [T-M11-02, T-M11-03, T-M11-04, T-M11-05, T-M11-06, T-M11-07, T-M11-08, T-M11-09, T-M11-10, T-M11-11, T-M11-12, T-M11-13, T-M11-14, T-M11-15, T-M11-16, T-M11-17, T-M11-18, T-M11-19, T-M11-20, T-M11-21, T-M11-22, T-M10-36]
parallel_group: M11-R
touches: [docs/engine-v6/rollout.md, docs/engine-v6/perf-baseline.md, docs/go-migration/deviations.md, docs/engine-v6/qa]
skills: [deploy-stage, verify-all]
verify: grep -q '## M11' docs/engine-v6/rollout.md && grep -q '^| D-190 .*shipped' docs/go-migration/deviations.md
---

# T-M11-23 — M11 release: stage rollout, privacy QA, performance check and deviation reconciliation
**Model:** main session (opus); sonnet subagents may run the visual passes; a `security-auditor` pass is mandatory.

## Why
M11 changes who can see and change another person's health data. It ships to stage as one reviewed release with a
security audit, and to beta only with the user's OK (beta has real users and real family links).

## Scope
1. `verify-all` green; `make contract ROUTES=all` green or every difference allow-listed with a D-190…D-206 id.
2. `security-auditor` review of `internal/family`, `internal/companion`, the care, children and babylog changes and the
   new routes (IDOR, never-shared, TOCTOU, invite oracle, sealing keys); every finding fixed or recorded.
3. Deploy to stage (deploy-stage skill) with `FAMILY_SEAL_KEYS` and the invite pepper set; run the migrations; check
   the legacy-grant migration on a copy of stage data (counts before/after, symptoms grants gone, owner notices sent).
4. Smoke with personas: partner in cycle mode (phase only), pregnant owner with a carer (urgent alert relay end to
   end, SMS switch off), delivery → review card and child suggestion, co-parents logging feeds and undoing, a member
   proposal approved and one expired, nudges at the cap and in quiet hours, mutual links revoked one way.
5. Visual QA light and dark of every changed screen (`docs/engine-v6/qa/m11/`); the main session reviews them.
6. Performance: member home and member section budgets (k6 scenario `family_home` added), relay latency.
7. Reconcile D-190…D-206 (status `shipped on stage`); add D-207…D-209 rows for differences found; "## M11" section in
   `docs/engine-v6/rollout.md`.
8. Ask the user before beta. If OK: backup (deploy-beta skill), deploy, run the legacy-grant migration, smoke, record.

## Out of scope
- Production (T-M2-26/27 stay blocked).

## Acceptance
- Stage runs M11; security review, QA and performance recorded; deviations reconciled; beta only after the user's OK.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.
