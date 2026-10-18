---
id: T-M8-14
title: Hardcoded Persian strings in frontend code — move to messages or rewrite
milestone: M8
type: frontend
status: todo
depends_on: [T-M8-03]
parallel_group: M8-C
touches: [frontend/src,frontend/public/manifest.webmanifest,frontend/messages/fa/common.json,frontend/messages/en/common.json,backend-go/resources/translations/fa/common.json,backend-go/resources/translations/en/common.json]
skills: [persian-writing:persian-writing,new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test
---

# T-M8-14 — Hardcoded Persian strings in frontend code — move to messages or rewrite
**Model:** sonnet subagent; main session reviews.

## Why
~340 non-comment lines in `frontend/src` contain Persian literals (e.g. `aria-label={label ?? 'برگرد'}` in
`shared/ui/NavBack.tsx`, metadata in `app/[locale]/layout.tsx`). CLAUDE.md forbids hardcoded user-facing text,
and they escaped the batch rewrites.

## Scope
1. List every Persian literal in `frontend/src` (excluding comments, tests, digit tables like `FA = ['۰',…]`,
   and Jalali month/weekday tables owned by `shared/lib/date`). Classify: user-facing UI / metadata / fallback /
   mock-dev data.
2. User-facing UI + fallbacks → translator keys in the owning slice's namespace (or `common` for shared/ui),
   with an `en` value; metadata (`layout.tsx`, manifest) → STYLE.md-correct copy (manifest stays static).
3. Every new/changed fa string follows STYLE.md + GLOSSARY.md; run `copy-fa/clean`, `sync`, `check` on touched
   namespaces.

## Out of scope
- Comments, tests, the Jalali date layer.

## Acceptance
- Re-running the step-1 grep shows only the allowed categories (paste the command + count).
- verify green; `check.mjs` 0 errors on touched namespaces.
