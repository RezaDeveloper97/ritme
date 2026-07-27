---
name: verify-all
description: Run the full Ritme verification suite — backend tests+style, frontend typecheck+lint+FSD rules+style gate+unit tests — and report a pass/fail table. Use before commits and always before deploy.
---

# Verify everything

Run all of these (parallelize backend and frontend):

## Backend
```bash
cd backend && vendor/bin/pint --test && php artisan test
```

## Frontend
```bash
cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run test
```

`lint:styles` enforces CLAUDE.md §10.1 (no static `style` props, no hex colour
literals, no `var(--x)` for an undeclared x). It is a ratchet against
`frontend/scripts/styles-baseline.json`, so it only fails on a *regression*. If
a file legitimately improved, run `npm run lint:styles:accept` to re-baseline —
never re-baseline to silence a genuine new violation.

If this is pre-deploy, also run `npm run build` in frontend — the production build catches errors dev mode doesn't, and a failed build wastes a full deploy cycle.

## Reporting
Report a short pass/fail line per command. For failures, give the root cause with `file:line` and say whether it's from the current change or pre-existing. Fix failures caused by the current change; flag pre-existing ones to the user instead of silently fixing.

Known failure causes to check first:
- SQLite date-cast + `updateOrCreate` mismatch
- `Passport::actingAs` stale relation cache (needs `$user->refresh()`)
- Missing fa/en message keys (next-intl) causing lint/test failures
