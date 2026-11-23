---
id: T-M12-01b
title: OTP caps per number+network so a stranger cannot lock a user out
milestone: M12
type: backend
status: done
depends_on: [T-M12-01]
parallel_group: M12-SEC
touches: [backend-go/internal/auth,backend-go/internal/http/routes_auth.go,backend-go/internal/platform/config/config.go,docs/quality-2026-11/findings.md]
skills: []
verify: cd backend-go && CGO_ENABLED=0 go vet ./internal/auth/... && make test-int PKG=./internal/auth/...
---

# T-M12-01b — OTP caps per number+network so a stranger cannot lock a user out

## Why
R-02 (independent re-audit): the T-M12-01 send caps (3 / 15 min, 10 / 24 h) and the failure lockout (10 wrong codes →
60 min) are keyed on sha256(mobile) only. Anyone who knows a number can use them up from one IP and block her login
and step-up codes for up to 24 h (works for unknown numbers too).

## Scope
- User decision: key the send caps and the failure lockout per (mobile, client network = `ratelimit.ClientNet`:
  IPv4 /32, IPv6 /64); same thresholds as T-M12-01.
- Keep a much higher global per-number ceiling (all networks) as the anti-bombing / distributed-guessing backstop:
  30 sends / 24 h, 50 failures / 24 h → number locked 1 h everywhere; env `OTP_SEND_MAX_DAILY_GLOBAL`,
  `OTP_FAILURE_MAX_GLOBAL`, `OTP_FAILURE_GLOBAL_WINDOW_HOURS`.
- Keys never carry a user id (delete + re-signup resets nothing); Redis keys via `cache.Client.Key()`, hashed number
  and network; no full number or code in logs.
- Same 429 envelopes (framework throttle, `Retry-After`, `X-RateLimit-*`); no new codes, no 401.
- Mark R-02 fixed in `docs/quality-2026-11/findings.md`.

## Out of scope
- Per-code attempts (`OTPMaxAttempts`), code length (D-211), the per-number send lock and resend window.
- Real client IP behind the edge (T-M12-02 / T-M12-09); frontend; Android.

## Acceptance
- Attacker on net A caps sends / trips the lock → owner on net B still gets and verifies a code; net A still capped
  / locked as before (incl. after deleting the account).
- Global ceilings trip across many networks; boundaries cap−1 / cap tested.
- Every new test fails when the fix is reverted (mutation); existing T-M12-01 tests stay green.
