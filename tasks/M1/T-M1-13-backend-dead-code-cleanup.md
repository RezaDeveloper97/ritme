---
id: T-M1-13
title: Backend dead code and dead storage cleanup
milestone: M1
type: backend
status: todo
depends_on: [T-M1-12]
parallel_group: M1-F
touches: [backend/app, backend/database/migrations, backend/routes, backend/resources/views, backend/tests]
skills: []
verify: cd backend && vendor/bin/pint --test && php artisan test
---

# T-M1-13 — Backend dead code and dead storage cleanup

## Scope
Remove what the baseline (`docs/investigations/perf-baseline.md` §2.3, §3 #1 and #10) proved unused. Each removal
needs a grep proving there is no caller in backend, frontend, the admin panel **and** `application/` (Android).
1. **`cycle_calculations` dead storage (baseline #1, the biggest win)**: stop dispatching `CalculateCycleDataJob`
   from `ProfileController`, `PeriodLogController`, `DailyHealthLogController` and `CycleCalculationController::recalculate`.
   Delete the job, the `CycleCalculation` model, `User::cycleCalculations()` and the admin `UserController` row count.
   Then drop the table in a new migration. On prod it holds 89,304 rows / 238 MB, ~99 % of the DB, and every write
   currently costs 366 rows + 376 queries of worker CPU.
   **Keep the contract**: `/cycle/status`, `/cycle/recalculate` and `month.is_recalculating` are called by the
   frontend (`useCycleStatus` polls every 4 s while `processing`) and by Android. Keep incrementing
   `calculation_version` on every engine-input write, because T-M1-12's cache key depends on it. Set status to
   `completed` synchronously so clients stop polling.
2. MatrixEngine (`app/Services/MatrixEngine/*`, 2,107 lines) and its routes `/cycle/matrix-messages` and
   `/cycle/matrix-enums`: 0 callers in frontend and Android. The same applies to `/cycle/enums` and `/messages/enums`.
   Confirm with the owner before removing documented API routes.
3. Local-only debug tooling: `Test{Matrix,Pregnancy,Message,Page}Controller` (1,534 lines), their
   `resources/views/test-*.blade.php` and the `local`-only block in `routes/web.php`. Owner decision: keep or remove.
4. `app/Services/MessageSystem/Contracts/ContextProviderInterface.php` has 0 references.

## Out of scope
Refactors of code that is in use.

## Acceptance
- List of removed items with the proof-of-no-caller in the task report.
- Full backend gate green.
