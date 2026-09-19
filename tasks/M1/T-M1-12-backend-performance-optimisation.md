---
id: T-M1-12
title: Backend performance optimisation (queries, indexes, caching)
milestone: M1
type: backend
status: done
depends_on: [T-M1-10]
parallel_group: M1-E
touches: [backend/app/Services, backend/app/Repositories, backend/app/Http/Resources, backend/app/Http/Controllers/Api/V1, backend/database/migrations, backend/tests, backend/Dockerfile, backend/docker/entrypoint.sh]
skills: [new-endpoint]
verify: cd backend && vendor/bin/pint --test && php artisan test
---

# T-M1-12 — Backend performance optimisation (queries, indexes, caching)

## Scope
Implement the backend items from `docs/investigations/perf-baseline.md` §3 (ranked). Baseline numbers are in §2
(seeded user, in-process harness §6.1), in the baseline's order:
1. **(#2) Framework caches at container start**: `config:cache`, `route:cache`, `event:cache` in
   `docker/entrypoint.sh` (env is runtime, so not at image build). Prod `bootstrap/cache` has none today; the
   measured win is −14 ms p50 (−21 %) per request locally. Opcache is already tuned, so leave it alone.
2. **(#3, backend half) Slim `/cycle/month`**: add an opt-in calendar view (e.g. `?view=calendar`) that returns only
   the fields `frontend/src/entities/cycle/api/schema.ts` reads, **without** per-day `daily_tips`/`text_flags`/`source_*`,
   and skip the work that produces them. The default response must stay unchanged because the Android app reads it.
   Encode with `JSON_UNESCAPED_UNICODE`. Baseline: 118,104 B, 46.9 ms p50; target ≈ 8.4 KB.
3. **(#6) Per-user cache of cycle-engine results** (`/cycle/month`, `/cycle/today`, `/cycle/date/*`), keyed by
   `user_id` + `user_profiles.calculation_version` + locale + **Tehran today** + args. It must be invalidated on every
   write that changes engine inputs, which the version bump already covers. Coordinate with T-M1-13: the version must
   keep bumping after the recalculation job is deleted.
4. **(#7) Query noise**: preload `daily_health_logs` for the range in the `/home` sections (9× per-day
   `whereDate` N+1 in `/home`, 7× in `week_calendar`; the fix is the `preloadDailyLogs` pattern), memoise
   `cycle_histories` per request (dup in `/cycle/today` and `/cycle/date/*`), and cache `message_contents` lookups
   per `(group, locale)` with invalidation on admin save (4 queries/request in `/messages/daily` and `smart_tip`).
5. Drop the redundant plain index `cycle_histories_user_id_period_start_date_index` (a UNIQUE covers the same
   columns) in a new migration.

Not needed, per the baseline: new indexes (prod `EXPLAIN` found none missing). Don't touch `CalculateCycleDataJob`
here; removing it is T-M1-13.

Migrations: new files only, never edit old ones. They must work on MariaDB **and** SQLite.

## Acceptance
- Query count per hot endpoint asserted in a feature test, so N+1s can't come back. Baseline counts (including 3
  Passport auth queries): `/home` 37, `week_calendar` 14, `/messages/daily` 14, `/cycle/today` 8, `/cycle/month` 7.
- Before/after p95 appended to `docs/investigations/perf-baseline.md`.
- `pint --test` + `php artisan test` green (mind memory `ritme-backend-testing-gotchas`).
