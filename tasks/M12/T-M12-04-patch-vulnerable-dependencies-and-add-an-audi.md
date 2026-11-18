---
id: T-M12-04
title: Patch vulnerable dependencies and add an audit gate (next, zod, laravel-site composer)
milestone: M12
type: fullstack
status: todo
depends_on: []
parallel_group: M12-SEC
touches: [frontend/package.json, frontend/package-lock.json, frontend/next.config.ts, admin-web/package.json, admin-web/package-lock.json, admin-web/next.config.ts, laravel-site/composer.json, laravel-site/composer.lock, .github/workflows/claude-to-stage.yml]
skills: [verify-all]
verify: cd frontend && npm audit --omit=dev --audit-level=high && npm run typecheck && npm test && cd ../admin-web && npm audit --omit=dev --audit-level=high && npm run typecheck && npm test
---

# T-M12-04 — Patch vulnerable dependencies and add an audit gate (next, zod, laravel-site composer)
**Model:** sonnet (opus reviews the diff). Evidence: `docs/quality-2026-11/findings.md`.

## Why
S-07/S-08: next below the patched 15.5.x line (cache-poisoning and image-optimizer advisories), zod DoS, and an SVG-sanitizer stored-XSS advisory in laravel-site.

## Scope
1. Upgrade `next` (and `eslint-config-next`) in frontend and admin-web to the newest patched 15.5.x; `zod` ≥ 4.6.6; let `npm audit fix` handle the transitive highs without a major bump.
2. `images: { unoptimized: true }` in both `next.config.ts` (the optimizer is unused); check `/_next/image` returns 400/404.
3. laravel-site: `composer update enshrined/svg-sanitize symfony/mime symfony/html-sanitizer`, run its tests.
4. CI: `npm audit --omit=dev --audit-level=high` for both apps and `composer audit --locked` for laravel-site.

## Out of scope
- Next 16 / React upgrades.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Both audits report 0 high/critical; typecheck, lint, vitest, `next build` green in both apps; laravel-site tests green.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
