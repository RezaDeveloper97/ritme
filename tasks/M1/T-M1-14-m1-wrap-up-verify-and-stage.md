---
id: T-M1-14
title: M1 wrap-up — full verification, staging deploy, release notes
milestone: M1
type: release
status: done
depends_on: [T-M1-05, T-M1-11, T-M1-12, T-M1-13]
parallel_group: M1-G
touches: [frontend/package.json, tasks/PROGRESS.md]
skills: [verify-all, deploy-stage]
verify: cd frontend && npm run build
---

# T-M1-14 — M1 wrap-up — full verification, staging deploy, release notes

## Scope
- Run `verify-all` + `npm run build`; fix anything this milestone broke.
- Bump `frontend/package.json` `version` (minor). Bump `pwa.minSupportedVersion` **only** if the session/SW changes
  require every client to update (decide from T-M1-03/08 notes; say why).
- Deploy to staging with `deploy-stage`; smoke-test sign-in, home, log, calendar, offline page, update toast.
- Write the M1 summary in `tasks/PROGRESS.md`. **Do not deploy to production** — ask the user.

## Acceptance
- All gates green, staging deployed and smoke-tested, user asked about the production deploy.
