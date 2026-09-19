---
id: T-M1-05
title: Session lifetime regression coverage and docs
milestone: M1
type: fullstack
status: todo
depends_on: [T-M1-02, T-M1-03, T-M1-04]
parallel_group: M1-C
touches: [docs/investigations/session-logout.md, frontend/CLAUDE.md, backend/CLAUDE.MD]
skills: [verify-all]
verify: cd backend && php artisan test && cd ../frontend && npm run test
---

# T-M1-05 — Session lifetime regression coverage and docs

## Scope
- End-to-end check against local dev (`local-dev` skill): sign in, travel the clock (Carbon::setTestNow / JWT
  near-expiry token) and verify the sliding refresh; verify an explicitly revoked token signs out cleanly.
- Close out `docs/investigations/session-logout.md` with "Fixed in" commit refs.
- Add the session rules to `frontend/CLAUDE.md` (§11 auth) and `backend/CLAUDE.MD`: token lifetime, what may clear
  a session, refresh window. Keep it short.
- Deploy to staging (`deploy-stage` skill) and sign in there so the 1-year behaviour can be observed.

## Acceptance
- `verify-all` green; docs updated; staging deployed and signed in.
