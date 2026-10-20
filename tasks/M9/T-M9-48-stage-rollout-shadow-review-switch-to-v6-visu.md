---
id: T-M9-48
title: Stage rollout: shadow, review, switch to v6, visual QA
milestone: M9
type: investigate
status: todo
depends_on: [T-M9-01, T-M9-02, T-M9-03, T-M9-04, T-M9-05, T-M9-06, T-M9-07, T-M9-08, T-M9-09, T-M9-10, T-M9-11, T-M9-12, T-M9-13, T-M9-14, T-M9-22, T-M9-23, T-M9-27, T-M9-28, T-M9-29, T-M9-31, T-M9-33, T-M9-34, T-M9-35, T-M9-36, T-M9-37, T-M9-38, T-M9-39, T-M9-41, T-M9-45, T-M9-46, T-M9-47]
parallel_group: M9-R
touches: [docs/engine-v6/rollout.md, docs/engine-v6/qa, docker-compose.stage.yml]
skills: [deploy-stage, verify-all]
verify: test -f docs/engine-v6/rollout.md && grep -q 'Stage: v6' docs/engine-v6/rollout.md
---

# T-M9-48 — Stage rollout: shadow, review, switch to v6, visual QA
**Model:** main session (deploys and judgement); sonnet subagents may take screenshots.

## Why
Stage is where v6 meets real data shapes before beta users see it (X-03). Shadow mode measures the differences
against v1.1 without changing what anyone sees.

## Scope
1. `verify-all` locally; then deploy stage (`deploy-stage` skill) with `CYCLE_ENGINE=shadow`, the worker container,
   `ENGINE_CACHE_VERIFY=1` sampled at 10 %. Migrations run; check `/ready`.
2. Drive traffic: the QA personas on stage (seed them if missing — stage only) through Home, Calendar (12 months back),
   fertility, analysis, companion; let shadow run ≥ 24 h.
3. `engine-shadow-report`: every diff class explained in `rollout.md` — expected (cites the E/I decision) or a bug
   (create a follow-up task, fix before switching).
4. Switch stage to `CYCLE_ENGINE=v6`; visual QA light and dark of Home (8 personas: regular, late 8, late 15, pill,
   postpartum, onboarding-only, stale anchor, skip), Calendar, phase details, fertility, analysis cycle, companion home;
   screenshots in `docs/engine-v6/qa/`.
5. Check stage metrics: `engine_cache_verify_mismatch_total = 0`, no 5xx increase, query counts per route.
6. Record "Stage: v6 since <date>" in `rollout.md`.

## Out of scope
- Beta (T-M9-50), production (never).

## Acceptance
- rollout.md complete; zero unexplained diff classes; zero cache mismatches; the user has seen the screenshots.
