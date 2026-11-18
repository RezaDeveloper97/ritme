---
id: T-M12-01
title: OTP brute-force and SMS-bombing hardening (per-number caps, failure budget, atomic send)
milestone: M12
type: backend
status: done
depends_on: []
parallel_group: M12-SEC
touches: [backend-go/internal/auth, backend-go/internal/http/routes_auth.go, backend-go/internal/platform/ratelimit, backend-go/db/queries/auth, backend-go/internal/platform/config/config.go, docs/go-migration/deviations.md, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && CGO_ENABLED=0 go test ./internal/auth/... ./internal/platform/ratelimit/... && make test-int PKG="./internal/auth/... ./internal/http/..."
---

# T-M12-01 — OTP brute-force and SMS-bombing hardening (per-number caps, failure budget, atomic send)
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
S-01/S-02: a 4-digit code with 5 tries that resets on every resend lets one IP brute-force any number in about a day and take a 365-day token to her health data. Parallel sends can bomb a number with SMS.

## Scope
1. Per-number send caps in Redis: at most 3 codes / 15 min and 10 / 24 h (values in config, not hard-coded). Over the cap → 429 with the existing throttle envelope.
2. Per-number failure budget across codes: 10 wrong codes / 24 h → the number is locked 1 h (even the right code fails) + a `slog.Warn` with the masked number and a metric.
3. Atomic send lock: Redis `SET NX EX 60` per number before the delete/insert, so N parallel sends dispatch exactly 1 SMS job.
4. Split the shared per-IP counter into named limiters (`otp_send`, `otp_verify`) and key IPv6 clients on their /64.
5. Code length stays 4 digits (user decision 2026-11-18) — no deviation row needed.
6. Success responses stay identical for existing and unknown numbers (no enumeration).

## Out of scope
- HMAC-stored OTPs (T-M12-09).
- Laravel `backend/` changes.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Integration tests with the test clock: (a) 3 send+5-wrong cycles 61 s apart → the 4th send is 429; (b) the 11th failure in 24 h → right code refused until +1 h; (c) 5 parallel sends → 1 job; (d) IPv6 addresses in one /64 share a bucket; (e) unknown and known numbers get the same body.
- All existing auth tests stay green; contract cases for `/send-otp`/`/verify-otp` unchanged except the 429 paths.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
