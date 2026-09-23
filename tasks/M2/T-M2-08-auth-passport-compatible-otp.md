---
id: T-M2-08
title: Auth — Passport-compatible tokens, OTP, SMS, rate limiting
milestone: M2
type: backend
status: done
depends_on: [T-M2-03, T-M2-04, T-M2-06]
parallel_group: M2-D
touches: [backend-go/internal/auth, backend-go/internal/platform/ratelimit, backend-go/internal/platform/queue, backend-go/internal/http/routes_auth.go, backend-go/db/queries/auth]
skills: []
verify: cd backend-go && go test ./internal/auth/... ./internal/platform/ratelimit/... && make test-int PKG=./internal/auth/... && make contract ROUTES=auth
---

# T-M2-08 — Auth: Passport-compatible tokens, OTP, SMS, rate limiting

## Why
Nobody may be logged out by the migration, and both stacks must accept each other's tokens during the strangler
period. The frontend drops the session on specific 401 `error_code`s, so the mapping must be exact.
Read infra-auth-admin-inventory §1–§2 and backend/CLAUDE.MD session rules first.

## Scope
1. `auth/passport`: load `STORAGE_PATH/oauth-public.key` (PKIX) and `oauth-private.key` (PKCS#8) — **never generate
   keys**; fail to start if either is missing or unreadable. Verify: golang-jwt v5, RS256 only, no leeway, float
   `iat/nbf/exp`, `aud` string or array → `oauth_access_tokens` row `id=jti AND revoked=0` (do **not** check DB
   `expires_at`) → `oauth_clients` by aud (exists, not revoked, provider NULL or `users`) → user by `sub`.
   Issue: same claim set (`aud` = latest non-revoked personal-access client, 80-hex `jti`, `sub` string, float
   µs `iat/nbf/exp`, `scopes: []`, header without `kid`), insert row (`name='auth_token'`, `scopes='[]'`,
   `expires_at = now + PASSPORT_TOKEN_LIFETIME_DAYS days` — a fixed day count, never "1 year").
2. Middleware `RequireUser`: Bearer header only; 401 JSON `{"message":"Unauthenticated.","error_code":…}` with
   `token_revoked` / `token_expired` (only when expiry is the sole failure) / `unauthenticated`. No 401 anywhere else.
3. `platform/ratelimit`: Redis fixed-window limiter matching Laravel `throttle:N,1` (key = user id if authenticated,
   else client IP from trusted XFF), headers `X-RateLimit-Limit/Remaining` on success, 429 pretty body
   `{"message":"Too Many Attempts."}` + `Retry-After`, `X-RateLimit-Reset`.
4. OTP: `POST /auth/send-otp` (5/min), `POST /auth/verify-otp` (10/min) exactly per api-inventory §1.1 (60s resend 429
   with `retry_after`, 2-min expiry, atomic attempt claim `attempts<5`, constant-time compare, 403 blocked with the
   same Persian text, firstOrCreate user + `mobile_verified_at`, `new_user`, `profile_completed` =
   filled(name) && profile exists). No bypass/test mode.
5. SMS: `sms.Provider` interface; Kavenegar (primary) and SMS.ir (fallback) HTTP clients per infra §2; `log` provider
   for non-prod. `platform/queue`: asynq on Redis (prefix `ritme-go`), in-process worker, job `send_otp_sms`
   (3 tries, backoff 5s/15s, fail when all providers fail).
6. `POST /auth/logout`, `GET /auth/user`, `POST /auth/refresh-session` (10/min; 30-day window; issue new before
   revoking old; `expires_at` ISO +03:30).

## Out of scope
Admin auth (T-M2-20). Routing auth traffic to Go in nginx (T-M2-25/26 — auth moves last).

## Acceptance
- Cross-stack test (contract env): a token issued by Laravel is accepted by Go; a token issued by Go is accepted by
  Laravel; a Laravel-revoked token gets `token_revoked` from Go; an expired token (test clock) gets `token_expired`;
  a garbage / wrong-signature token gets `unauthenticated`.
- `make contract ROUTES=auth` green (incl. 429 bodies/headers, 400/403/422 paths).
- Unit tests for the limiter window and header values; the SMS fallback order; the OTP attempt race (parallel verify).
- `backend-go/CLAUDE.md` session rules still hold (lifetime is a day count; refresh window; revoke only on
  logout/refresh here).
