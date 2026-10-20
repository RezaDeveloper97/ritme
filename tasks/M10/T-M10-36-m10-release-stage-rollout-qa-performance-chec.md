---
id: T-M10-36
title: M10 release: stage rollout, QA, performance check and deviation reconciliation
milestone: M10
type: release
status: todo
depends_on: [T-M10-02, T-M10-03, T-M10-04, T-M10-05, T-M10-06, T-M10-07, T-M10-08, T-M10-09, T-M10-10, T-M10-11, T-M10-12, T-M10-13, T-M10-14, T-M10-15, T-M10-16, T-M10-17, T-M10-18, T-M10-19, T-M10-20, T-M10-21, T-M10-22, T-M10-23, T-M10-24, T-M10-25, T-M10-26, T-M10-27, T-M10-28, T-M10-29, T-M10-30, T-M10-31, T-M10-32, T-M10-33, T-M10-34, T-M10-35, T-M9-50]
parallel_group: M10-R
touches: [docs/engine-v6/rollout.md, docs/engine-v6/perf-baseline.md, docs/go-migration/deviations.md, docs/engine-v6/qa]
skills: [deploy-stage, verify-all]
verify: grep -q '## M10' docs/engine-v6/rollout.md && grep -q '^| D-160 .*shipped' docs/go-migration/deviations.md
---

# T-M10-36 — M10 release: stage rollout, QA, performance check and deviation reconciliation
**Model:** main session (opus); sonnet subagents may run the visual QA passes.

## Why
M10 changes what users see on the home, in alerts, in health modules and in analysis, and what admins can do. It has
to reach stage as one reviewed release, then beta only with the user's OK (beta has real users), with the deviation
rows reconciled.

## Scope
1. `verify-all` green on the merged state; `make contract ROUTES=all` green or every difference allow-listed with a
   D-160…D-183 id.
2. Deploy to stage (deploy-stage skill); run the migrations; smoke: feed, alerts inbox, a pregnancy persona's urgent
   alert path (kick session < 10), a diabetic persona's vitals, a lab with a low-confidence value, checkups for ages
   29/30/40, the consent sheet for an existing user, analysis hub cached/304, admin simulator with a health grant,
   dashboard alerts, user directory with filters.
3. Visual QA light and dark of every changed screen (user app and admin-web), saved under `docs/engine-v6/qa/m10/`;
   the main session reviews them.
4. Performance: feed, alerts and analysis budgets against perf-baseline.md (k6 scenarios from T-M9-14 extended with
   `feed` and `analysis_tab`), local run.
5. Reconcile D-160…D-183 (status `shipped on stage`), add rows in D-184…D-189 for any difference found; update
   `docs/engine-v6/rollout.md` with a "## M10" section.
6. Ask the user before deploying to beta; if OK: backup first (deploy-beta skill), deploy, smoke, record.

## Out of scope
- Production (T-M2-26/27 stay blocked).

## Acceptance
- Stage runs M10; QA and performance recorded; deviations reconciled; beta deployed only after the user's OK.
