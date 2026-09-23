---
id: T-M2-01
title: Laravel test clock and deterministic contract fixtures
milestone: M2
type: backend
status: done
depends_on: []
parallel_group: M2-A
touches: [backend/app/Http/Middleware/TestClock.php, backend/app/Services/Sms/SmsService.php, backend/bootstrap/app.php, backend/app/Services/Sms/Providers/LogSmsProvider.php, backend/config/sms.php, backend/database/seeders/ContractFixtureSeeder.php, backend/tests/Feature/ContractFixtureTest.php, docker-compose.contract.yml, docs/go-migration/contract.md]
skills: [local-dev]
verify: cd backend && php artisan test --filter=ContractFixtureTest
---

# T-M2-01 — Laravel test clock and deterministic contract fixtures

## Why
The Go port is gated by golden responses recorded from Laravel (docs/go-migration/README.md → Contract). Most
Laravel responses depend on "today" (Asia/Tehran) and on user data, and only 4 PHP tests freeze time
(domain-inventory §7). We need a reproducible Laravel instance: MariaDB 11.4 + Redis, a fixed clock and a fixed
set of user personas, so the recorder (T-M2-05) produces the same goldens every run.

## Scope
1. `TestClock` middleware: if `APP_ENV` ∈ {local, testing, contract} **and** `TEST_CLOCK_ENABLED=true`, read header
   `X-Test-Now` (ISO-8601, Tehran) and call `Carbon::setTestNow()` for the request. Never active in production
   (fail closed; a test proves it). Register globally in `bootstrap/app.php`.
2. `LogSmsProvider` (`SMS_PROVIDER=log`): logs instead of sending; refuses to boot in production.
3. `ContractFixtureSeeder`: idempotent, deterministic (fixed ids / mobiles `0990000xxxx`, no Faker randomness). Seeds
   all content (runs the existing content seeders: pregnancy weeks, phase content, info, challenges, tasks,
   articles, affirmations, recommendations, message contents; languages fa/en + one extra active language `ar`
   to exercise "3rd locale" paths; 2–3 banners incl. one expired) and ~12 personas, e.g.:
   no-profile, profile-no-history, onboarding-declared only, regular 28d (6 confirmed cycles), irregular,
   short-outlier, open period (day 3 / day 11 past hard cap), overdue 10d / 20d, TTC goal, premium subscriber,
   pregnant (LMP / ultrasound / manual; week 8 / 26 with alerts, symptoms, weekly logs, fetal movements),
   blocked user, user with reminders + notifications + task/challenge completions.
   Relative dates are computed from a fixed anchor `CONTRACT_TODAY=2026-09-23` so data and clock line up.
4. `docker-compose.contract.yml`: `contract-mariadb` (11.4), `contract-redis`, `contract-laravel` (APP_ENV=contract,
   TEST_CLOCK_ENABLED=true, SMS_PROVIDER=log, QUEUE_CONNECTION=sync, CACHE_STORE=array), throwaway Passport keys in
   a throwaway volume, published on 127.0.0.1:8090. A Make/script target rebuilds the DB from scratch and runs
   `ContractFixtureSeeder` (this only ever touches the contract container's DB).
5. `docs/go-migration/contract.md`: personas table (id, mobile, what it exercises), how to boot, clock header.

## Out of scope
The recorder/differ (T-M2-05). Any Go code. Any production behaviour change.

## Acceptance
- `ContractFixtureTest`: seeder runs twice without error and yields identical row counts; TestClock is ignored when
  `APP_ENV=production` or the flag is off; `GET /api/v1/cycle/today` for the regular persona with
  `X-Test-Now: 2026-09-23T10:00:00+03:30` is identical across two calls.
- `docker compose -f docker-compose.contract.yml up` serves `/up` 200 on :8090 and an OTP login for a persona works
  (code read from the DB).
- No production config or compose file changed.
