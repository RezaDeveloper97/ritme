---
id: T-M12-25
title: Browser smoke harness (Playwright) for the main routine against the local stack
milestone: M12
type: frontend
status: todo
depends_on: [T-M12-15, T-M12-12]
parallel_group: M12-TEST
touches: [frontend/e2e, frontend/playwright.config.ts, frontend/package.json, .github/workflows/ci.yml]
skills: [local-dev, run]
verify: cd frontend && npx playwright test
---

# T-M12-25 — Browser smoke harness (Playwright) for the main routine against the local stack
**Model:** sonnet. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-05: there is no browser-level test at all; the API flows (T-M12-15) do not catch a screen that renders raw keys, a broken sheet or a dead button.

## Scope
1. Playwright with the local stack (backend-go on :8020 with `SMS_PROVIDER=log`, OTP read from the DB).
2. Specs: login → onboarding → log period via the log sheet → home ring shows the new day → calendar shows the period; light and dark; fa locale; assert no raw `namespace.key` text on each page.
3. Use the autopilot `pilot()` names as selectors where they exist.
4. CI job optional (nightly) — start it only if the stack boots in under ~5 minutes.

## Out of scope
- Android. Payment and AI submit (covered by API flows).
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- `npx playwright test` green locally; how to run it documented in frontend/CLAUDE.md testing section.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
