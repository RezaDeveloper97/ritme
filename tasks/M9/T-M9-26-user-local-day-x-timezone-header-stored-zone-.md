---
id: T-M9-26
title: User-local day: X-Timezone header, stored zone, LocalToday
milestone: M9
type: fullstack
status: done
depends_on: [T-M9-24]
parallel_group: M9-C
touches: [backend-go/internal/platform/civildate, backend-go/internal/platform/usertz, backend-go/cmd/api/app.go, backend-go/cmd/api/cors.go, backend-go/internal/cycle/service/handlers.go, backend-go/db/queries/profile/timezone.sql, backend-go/internal/profile/store, frontend/src/shared/api/apiClient.ts, frontend/src/shared/api/apiClient.test.ts]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/platform/... ./internal/cycle/... ./cmd/... && make test-int PKG=./internal/platform/usertz/... && cd ../frontend && npm run typecheck && npm run lint && npm run test
---

# T-M9-26 — User-local day: X-Timezone header, stored zone, LocalToday
**Model:** opus.

## Why
§2.3: "a day is the user's local day from the device timezone (default Asia/Tehran), not server UTC". Today every
"today" is Tehran (241 call sites); a user in Berlin at 23:30 local gets tomorrow's status (gap 02 IC-08, gap 07 B3).
X-07 scopes the switch to the engine path and its direct consumers.

## Scope
1. Frontend `apiClient` sends `X-Timezone: Intl.DateTimeFormat().resolvedOptions().timeZone` on every API call
   (omitted if unavailable). Display formatting stays as it is.
2. Middleware `usertz`: validate with `time.LoadLocation` (tzdata embedded), cache parsed locations; precedence header →
   `user_profiles.timezone` → `Asia/Tehran`. When an authed request carries a valid zone different from the stored one,
   store it (one `UPDATE … WHERE user_id = ? AND (timezone IS NULL OR timezone <> ?)`, after the response, never blocking
   it). No revision bump (the zone changes "today", which is in the ETag, not the engine state).
3. `civildate.LocalToday(c)` / `LocalNow(c)` helpers; `civildate.Today` stays for Tehran call sites.
4. Switch the cycle service handlers now; T-M9-32…38 use `LocalToday` in what they rewrite. Background jobs use the
   stored zone (`usertz.ZoneFor(profile)`). DB timestamps stay Tehran wall-clock.
5. CORS: confirm `X-Timezone` and `If-None-Match` pass the preflight (`cmd/api/cors.go` echoes requested headers) and add
   a test; check the stage/beta nginx config does not strip the header.
6. Document in `docs/engine-v6/README.md` §5 the list of call sites that intentionally stay on Tehran.

## Tests
- Instant 2026-07-17T21:00Z: header Asia/Tehran → 2026-07-18; Europe/Berlin → 2026-07-17; no header and no stored
  zone → Tehran (identical to today's behaviour); invalid header → ignored.
- 00:30 Tehran on a UTC server clock → the Tehran date (§6.1).
- Nowruz boundary: 2026-03-20T20:29Z → 2026-03-20 and 20:30Z → 2026-03-21 in Tehran (fixed: the original line was one minute off).
- Integration: a valid new zone is stored once; the same zone again → no UPDATE (query counter).
- Contract runs without the header stay byte-identical (`make contract ROUTES=cycle`).
- Frontend vitest: the header is present on requests.
- CORS preflight test with `Access-Control-Request-Headers: x-timezone, if-none-match`.

## Out of scope
- The ~100 other `civildate.Today` call sites (listed, not changed).

## Acceptance
- Tests green; PROGRESS.md cites D-126.
