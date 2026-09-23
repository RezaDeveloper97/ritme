# Contract stack: reproducible Laravel for golden recording

The Go port is checked against golden responses recorded from Laravel (see [README.md](README.md) → Contract).
This page describes the Laravel side: a throwaway stack with a fixed clock and fixed personas, so every recording
run produces the same bytes. The recorder/differ itself is T-M2-05.

| Piece | Where |
|---|---|
| Stack | `docker-compose.contract.yml` (project `ritme-contract`): `contract-mariadb` (11.4, tmpfs), `contract-redis`, `contract-laravel` |
| Fixed clock | `backend/app/Http/Middleware/TestClock.php` (global middleware) |
| No-op SMS | `backend/app/Services/Sms/Providers/LogSmsProvider.php` (`SMS_PROVIDER=log`) |
| Fixtures | `backend/database/seeders/ContractFixtureSeeder.php` |
| Tests | `backend/tests/Feature/ContractFixtureTest.php` |

## Boot

```bash
docker compose -f docker-compose.contract.yml up -d --build         # http://127.0.0.1:8090
docker compose -f docker-compose.contract.yml run --rm contract-reset  # fresh DB + fixtures (run after every `up`)
curl -s http://127.0.0.1:8090/up                                     # 200
docker compose -f docker-compose.contract.yml down -v               # throw everything away
```

- If 8090 is taken on your machine, prefix every command with `CONTRACT_PORT=18090` (or another port).
- `contract-laravel` runs with `APP_ENV=contract`, `APP_DEBUG=false`, `TEST_CLOCK_ENABLED=true`,
  `SMS_PROVIDER=log`, `QUEUE_CONNECTION=sync`, `CACHE_STORE=array`, `SESSION_DRIVER=array`, admin panel off.
- The entrypoint migrates, generates a **throwaway** Passport key pair on the `contract-storage` volume and creates the
  personal access client, exactly as in production.
- `contract-reset` (profile `tools`) runs `migrate:fresh`, recreates the personal access client (fresh drops
  `oauth_clients`), seeds the admin (`admin@contract.test` / `contract-admin`) and then `ContractFixtureSeeder`.
  It only ever talks to `contract-mariadb`. Existing tokens die with the reset — log in again afterwards.
- `APP_KEY` is a fixed contract-only value (override with `CONTRACT_APP_KEY`); it protects nothing real.

## Clock header

```
X-Test-Now: 2026-09-23T10:00:00+03:30
```

- Active only when `APP_ENV` ∈ {`local`, `testing`, `contract`} **and** `TEST_CLOCK_ENABLED=true`. Otherwise the header
  is ignored (production can never match; a test proves it).
- ISO-8601; a value without an offset is read as Asia/Tehran wall-clock. It sets `Carbon::setTestNow()` for that
  request only and restores the previous value afterwards.
- When active, an unparseable value returns **400** `{"message":"Invalid X-Test-Now header."}` instead of silently
  falling back to the real clock.
- Always send it on recording runs, including for "clock-free" endpoints. Use a time on `CONTRACT_TODAY`
  (**2026-09-23**) so the clock and the fixture data line up. Passport's own token expiry check uses the real clock.

## Logging in as a persona

There is no OTP bypass (see `OtpNoBypassTest`). Send the OTP, read the code from the contract DB, verify:

```bash
B=http://127.0.0.1:8090/api/v1
curl -s -X POST $B/auth/send-otp -H 'Content-Type: application/json' -d '{"mobile":"09900000004"}'
CODE=$(docker compose -f docker-compose.contract.yml exec -T contract-mariadb \
  mariadb -uritme -pcontract ritme_contract -N -e \
  "select code from otp_verifications where mobile='09900000004' order by id desc limit 1")
curl -s -X POST $B/auth/verify-otp -H 'Content-Type: application/json' \
  -d "{\"mobile\":\"09900000004\",\"code\":\"$CODE\"}" | jq -r .data.access_token
```

`send-otp` has a 60 s resend window per mobile and a 5/min per-IP throttle; the recorder should log each persona
in once and reuse the token. A never-seeded mobile (e.g. `09900009999`) exercises the new-user signup path.

## Fixture data

All relative dates are computed from `ContractFixtureSeeder::CONTRACT_TODAY = 2026-09-23`; all
`created_at`/`updated_at` (and the content seeders' `now()`) are `2026-09-23 09:00:00` Tehran. The seeder is
idempotent: content seeders are `firstOrCreate`/`updateOrCreate`, and every persona's rows are deleted and re-inserted
with the same ids on each run (so it also undoes writes a recording made). Row ids: users `1001…`, a persona's child
rows `userId * 100 + n` (e.g. the regular persona's periods are `100401…100406`).

Content: languages `fa` (default), `en` and an extra active `ar` (row only — no translation files or
`message_contents`, so it exercises fallback paths); task templates, articles, affirmations, challenges, message
contents, recommendations, 40 pregnancy weeks, phase content, info sections; banners `1` (home_top, internal link,
active window), `2` (home_middle, external link, no window) and `3` (home_top, **expired** yesterday). Banner images
are not on disk; only the URLs are generated. The seeded admin is not part of `ContractFixtureSeeder`.

Profile defaults unless noted: birthday 1996-04-12, 60.5 kg, 165 cm, period 5 d, cycle 28 d, `non_ttc`,
`avoiding`, `free`. Cycle history rows are confirmed `user_logged` 5-day periods.

| id | mobile | persona key | what it exercises |
|---|---|---|---|
| 1001 | 09900000001 | `no_profile` | verified user, no name, no profile → `profile_completed=false`, empty cycle/home paths |
| 1002 | 09900000002 | `profile_no_history` | profile LMP 2026-09-13, no `cycle_histories` → resolver falls back to profile LMP |
| 1003 | 09900000003 | `onboarding_declared` | single `is_estimated` / unconfirmed `onboarding_estimate` row starting 2026-09-11 |
| 1004 | 09900000004 | `regular` | 6 confirmed 28-day cycles, last start 2026-09-14 (cycle day 10, fertile window); 4 daily logs (2 bleeding, 1 today) |
| 1005 | 09900000005 | `irregular` | lengths 24/35/29/41/26 (latest three span > 7 d), last start 2026-09-09, profile cycle 30 |
| 1006 | 09900000006 | `short_outlier` | a 17-day outlier among 28/29-day cycles, last start 2026-09-03 |
| 1007 | 09900000007 | `open_period_day3` | open period (no end) started 2026-09-21 → day 3 |
| 1008 | 09900000008 | `open_period_day11` | open period started 2026-09-13 → day 11 (past warning cap 8) |
| 1009 | 09900000009 | `open_period_day13` | open period started 2026-09-11 → day 13 (past hard cap 12) |
| 1010 | 09900000010 | `overdue_10` | last start 2026-08-16, ECL 28 → period 10 days late |
| 1011 | 09900000011 | `overdue_20` | last start 2026-08-06 → 20 days late (beyond the 14-day overdue window) |
| 1012 | 09900000012 | `ttc` | `user_goal=ttc`, `pregnancy_intention=trying`, 28/29/27-day cycles, last start 2026-09-11 |
| 1013 | 09900000013 | `premium` | `subscription_type=premium`, chronic `pcos`, 20 consecutive low-energy/fatigue/bad-sleep logs (premium pattern layer, `chronic_fatigue`) |
| 1014 | 09900000014 | `pregnant_lmp_w8` | pregnancy via LMP 2026-08-02 (7w3d → week 8), medium confidence, 1 symptom log |
| 1015 | 09900000015 | `pregnant_ultrasound_w26` | ultrasound 23w2d on 2026-09-09 (→ week 26), Rh−, hypertension, miscarriage history; 3 symptom logs (incl. spotting + 3 severe), weekly logs weeks 24–26 (week 26 BP 142/92, fasting sugar 97.5), fetal movements (today `reduced`), 3 alerts (emergency unread, warning read, info dismissed) |
| 1016 | 09900000016 | `pregnant_manual_w14` | manual 12w0d entered 2026-09-16 (→ week 14), low confidence |
| 1017 | 09900000017 | `blocked` | `blocked_at` set → `verify-otp` 403 |
| 1018 | 09900000018 | `engaged` | 4 reminders (doctor future, daily medication, past appointment, inactive weekly custom), 3 notifications (1 read), 2 task completions (today, yesterday), 2 challenge completions |

## Open items

- `SmsService::resolveProvider()` (`backend/app/Services/Sms/SmsService.php`) does not yet map `log` to
  `LogSmsProvider`, so with `SMS_PROVIDER=log` the inline OTP job logs `Unknown SMS provider: log`. The controller
  swallows that and `send-otp` still returns 200 with the code in the DB, so login works; wiring it is a one-line
  `'log' => new LogSmsProvider` arm outside this task's file list.
