---
id: T-M2-04
title: Platform core — civil dates, clock, PHP-compatible JSON, envelopes and errors
milestone: M2
type: backend
status: todo
depends_on: [T-M2-02]
parallel_group: M2-B
touches: [backend-go/internal/platform/civildate, backend-go/internal/platform/clock, backend-go/internal/platform/phpround, backend-go/internal/platform/jsonx, backend-go/internal/platform/httpx]
skills: []
verify: cd backend-go && go test ./internal/platform/civildate/... ./internal/platform/clock/... ./internal/platform/phpround/... ./internal/platform/jsonx/... ./internal/platform/httpx/...
---

# T-M2-04 — Platform core: civil dates, clock, PHP-compatible JSON, envelopes and errors

## Why
Every endpoint depends on Laravel/PHP output rules that Go handles differently (api-inventory §2, §5, §7;
domain-inventory §5, §8). Getting these right once, with exhaustive tests, makes the domain ports mechanical.

## Scope
1. `civildate`: a `Date` (y/m/d, no time) anchored to Asia/Tehran: `Today(clock)`, `AddDays`, `DiffDays` (civil,
   DST-safe; Iran had DST until 2022), Saturday-start week, ISO weekday (Mon=1..Sun=7), `DayOfYear`, `Y-m-d`
   parse/format, a `Carbon::parse`-compatible lenient parse for `/cycle/date/{date}`, SQL Scan/Value, age in years.
2. `clock`: injectable clock (real = now in Tehran; fixed for tests) + HTTP middleware honouring `X-Test-Now` only
   when `APP_ENV` ∈ {local,testing,contract} and `TEST_CLOCK_ENABLED=true` (same rule as T-M2-01).
3. `phpround`: PHP `round($x,$p)` (half away from zero with PHP's pre-rounding; golden values for 1.955, 2.5, -2.5,
   0.285, 1.005 …), PHP float→string (`12.5`, `12`) for interpolation, `json_encode` float form, Persian digits.
4. `jsonx`: encoder with `SetEscapeHTML(false)` plus types: `DecimalString` (`"65.50"`, per-column precision),
   `LaravelDateTime` (UTC `…T15:04:05.000000Z`), `LaravelDateCast` (plain `date` cast → Tehran midnight → UTC =
   previous day `T20:30:00.000000Z`, DST-era `T19:30:00.000000Z`), `ISO8601Tehran` (`+03:30`), `YMD`; a `PHPArray`
   map that marshals empty as `[]`; nil-slice → `[]` helpers; an ordered map for literal key order; an
   UNESCAPED_UNICODE option (the harness normalises escaping, but types must match).
5. `httpx`: controller envelopes `OK(data, msg?)`, `Fail(status, msg, extras)`; framework errors: 422
   `{message:"<first> (and N more errors)", errors}` (with Laravel's fa/en "and N more" strings), pretty-printed
   (4-space, unescaped slashes) 404 route / model-not-found / 405 / 429 / 500 / 503 with exact texts; the Fiber
   error handler; the 3 paginator shapes (raw LengthAwarePaginator with absolute URLs and `links[]` labels
   `&laquo; Previous` / `Next &raquo;`; `pagination{}`; `meta{}`).

## Out of scope
Validation rules and translated messages (T-M2-06). Auth errors (T-M2-08).

## Acceptance
- Table tests for every helper; expected values come from running the equivalent PHP (`php -r …`), with the snippet
  and output recorded in the test comments.
- `LaravelDateCast` covers 2021-06-01 (DST era) and 2026-09-23.
- The unknown-route 404 and the 405 bodies byte-match a Laravel capture committed as testdata.
