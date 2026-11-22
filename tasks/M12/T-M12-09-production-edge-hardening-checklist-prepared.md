---
id: T-M12-09
title: Production edge hardening checklist (prepared now, applied only at cutover with the user)
milestone: M12
type: release
status: blocked
depends_on: [T-M12-02]
parallel_group: M12-SEC
touches: [docs/quality-2026-11/prod-edge.md, tasks/M2/T-M2-26-production-cutover-to-go-per-route-group-with-r.md]
skills: []
verify: test -f docs/quality-2026-11/prod-edge.md
---

# T-M12-09 — Production edge hardening checklist (prepared now, applied only at cutover with the user)
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
S-15/S-16/S-17: prod reads the client-controlled XFF, Laravel sees the CDN edge as the client, `/oauth/*` is exposed, OTPs are stored in plain text. All touch production, which is frozen until T-M2-26.

## Scope
1. Write `prod-edge.md`: exact nginx diff (CDN real_ip include from T-M12-02, `location ^~ /oauth/ { return 404; }`), Laravel TrustProxies change, OTP HMAC storage plan + deviation row, and the smoke checks.
2. Add a line to T-M2-26 Scope pointing to this checklist as a prerequisite.

## Out of scope
- Applying anything to production. This task is blocked on purpose (no prod cutover rule); unblock only when the user asks.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Checklist reviewed by the user.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
