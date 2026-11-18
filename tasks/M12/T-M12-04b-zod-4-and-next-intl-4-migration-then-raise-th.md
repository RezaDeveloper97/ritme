---
id: T-M12-04b
title: Zod 4 and next-intl 4 migration, then raise the npm audit gate to high
milestone: M12
type: frontend
status: todo
depends_on: [T-M12-04]
parallel_group: M12-SEC
touches: [frontend/package.json,frontend/package-lock.json,frontend/src,admin-web/package.json,admin-web/package-lock.json,admin-web/src,.github/workflows/claude-to-stage.yml]
skills: [verify-all]
verify: cd frontend && npm audit --omit=dev --audit-level=high && npm run typecheck && npm test && cd ../admin-web && npm audit --omit=dev --audit-level=high && npm run typecheck && npm test
---

# T-M12-04b — Zod 4 and next-intl 4 migration, then raise the npm audit gate to high

**Model:** sonnet (opus reviews). Evidence: `docs/quality-2026-11/findings.md` S-07; follow-up of T-M12-04.

## Why
T-M12-04 patched next/postcss/laravel-site, but three advisories need majors: zod 3→4 (GHSA-mrhx-c7xc-m3w5, memory
DoS, high, both apps), next-intl 3→4 (open redirect, moderate, frontend; pulls picomatch high in admin-web). The CI
audit gate runs at `--audit-level=critical` until this lands.

## Scope
1. zod 4 in frontend (~292 type errors, 113 files import zod; e.g. `.default({})` in `shared/autopilot/config.ts`)
   and admin-web (`src/shared/api/client.ts:151`). Use the official zod 4 migration notes; behaviour must not change
   (API response schemas still tolerant where they were).
2. next-intl 4 in frontend (and admin-web if used); keep the custom locale routing (memory: multi-language — the
   next-intl routing middleware was deliberately replaced).
3. If picomatch is still flagged with no upstream fix, document the exposure (build-time only?) and accept it with
   the user, or override.
4. Raise both CI audit steps to `--audit-level=high`.

## Out of scope
- Next 16 / React upgrades. Android.

## Acceptance
- Both audits 0 high/critical (or a user-accepted, documented exception); typecheck, lint, vitest, build green in both apps.
- Spot-check schemas: existing API schema tests pass unchanged; a malformed payload is still rejected where it was.
