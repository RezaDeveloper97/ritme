---
id: T-M12-15
title: Routine flow test: OTP signup → onboarding → log period → today/calendar prediction
milestone: M12
type: backend
status: todo
depends_on: [T-M12-11]
parallel_group: M12-TEST
touches: [backend-go/internal/http/routine_signup_int_test.go, backend-go/internal/http/testenv]
skills: []
verify: cd backend-go && make test-int PKG="./internal/http/..." RUN=TestRoutine
---

# T-M12-15 — Routine flow test: OTP signup → onboarding → log period → today/calendar prediction
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-05: each step has its own tests, but no test drives the whole first-day path through real routes; a contract drift between steps breaks the app's main loop without any red test.

## Scope
1. Shared `testenv` helper (reuse `cycle/revision/bump_int_test.go` env + test clock): OTP read from DB, real Fiber app, real MariaDB/Redis.
2. Flow: send-otp → verify → profile/onboarding (regular 28/5) → period start → period end → `/cycle/today` and `/cycle/month` show the expected day, phase and next-period prediction; edit the period date → prediction moves; ETag changes after each write and 304 before.
3. Table cases: irregular cycle, teen, user who skips onboarding, clock at 23:59 and 00:01 Tehran, 30 Esfand of a leap year.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Flow test green and asserts concrete dates (not just 200s).
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
