---
id: T-M2-24
title: Parity gate — full contract diff, client smoke tests, load comparison
milestone: M2
type: investigate
status: done
depends_on: [T-M2-10, T-M2-11, T-M2-12, T-M2-15, T-M2-16, T-M2-17, T-M2-18]
parallel_group: M2-I
touches: [docs/go-migration/parity-report.md]
skills: [verify-all]
verify: test -s docs/go-migration/parity-report.md && cd backend-go && make contract ROUTES=all
---

# T-M2-24 — Parity gate

## Why
Before any traffic moves, prove the Go API is a drop-in replacement: every route, persona and locale matches, both
clients work end to end, and it is not slower. This is an investigate task: it changes no code; failures become
follow-up tasks (`T-M2-NNb`) on the owning domain task.

## Scope
1. `make contract ROUTES=all` on a clean checkout (record the run output); list every allow-list entry with its
   deviation id and user approval.
2. Web frontend smoke: run `frontend/` against Go (`.env.local` → Go on the contract DB) and click through signup
   (OTP from DB), onboarding, home, calendar, period start/end, daily log, pregnancy onboarding/tracker, articles,
   settings/export/delete; headless screenshots vs the same flow on Laravel.
3. ~~Android smoke~~ — dropped 2026-09-23: the user excluded Android from every /next-task task.
4. Token interop re-check (T-M2-08 acceptance) on the contract env.
5. Load comparison with `hey` or k6 on identical hardware/DB: `/cycle/today`, `/cycle/month` calendar, `/home`,
   `/messages/daily`, `/banners` — p50/p95/p99, RPS, memory; Laravel vs Go.
6. Write `docs/go-migration/parity-report.md`: results, screenshots, numbers, open issues → follow-up tasks.

## Out of scope
Fixing issues inside this task (create follow-ups instead).

## Acceptance
- Report exists with evidence; `make contract ROUTES=all` green; no unexplained allow-list entries.
- Every smoke step ✔ on web, or a follow-up task exists and blocks T-M2-25.
