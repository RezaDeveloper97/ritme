---
id: T-M1-13
title: Backend dead code and dead storage cleanup
milestone: M1
type: backend
status: todo
depends_on: [T-M1-12]
parallel_group: M1-F
touches: [backend/app, backend/database/migrations, backend/routes, backend/tests]
skills: []
verify: cd backend && vendor/bin/pint --test && php artisan test
---

# T-M1-13 — Backend dead code and dead storage cleanup

## Scope
Remove what the baseline proved unused: dead services/controllers/routes, the `cycle_calculations` dead storage
(stop writing to it; drop the table in a new migration only after confirming nothing reads it — grep backend,
frontend and admin panel), unused config. Each removal needs a grep proving no caller.

## Out of scope
Refactors of code that is in use.

## Acceptance
- List of removed items with the proof-of-no-caller in the task report.
- Full backend gate green.
