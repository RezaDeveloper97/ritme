---
id: T-M12-02
title: Trust the client IP only from the CDN on beta/stage; close the stage gate private-address bypass
milestone: M12
type: release
status: blocked
depends_on: []
parallel_group: M12-SEC
touches: [deploy/beta-ssl.conf, deploy/vhost-beta.inc, deploy/vhost-beta-admin.inc, deploy/vhost-stage.inc, deploy/stage-gate.sh, deploy/cdn-real-ip.inc, backend-go/cmd/api/app.go, deploy/smoke]
skills: [deploy-stage]
verify: bash deploy/smoke/forged-xff.sh https://stage.ritmeapp.ir
---

# T-M12-02 — Trust the client IP only from the CDN on beta/stage; close the stage gate private-address bypass
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
S-03/S-04/S-05: a request straight to the origin with a forged X-Forwarded-For defeats every per-IP limit (OTP, share lookups, admin login) on beta, where real users are; the stage password gate opens for any private source address.

## Scope
1. `deploy/cdn-real-ip.inc`: `set_real_ip_from` for the CDN (ArvanCloud) ranges + `real_ip_header X-Forwarded-For; real_ip_recursive on;`; vhosts forward `$remote_addr` only. Keep the range list in one file with its source URL and date.
2. Go: `TrustedProxies` = the nginx container network only, not `0.0.0.0/0`.
3. Stage gate: exempt only the exact compose subnet of the frontend container (or a shared-secret header from SSR); never all of 10/8 and 172.16/12. Add `Secure` to the `ritme_stage` cookie.
4. `deploy/smoke/forged-xff.sh`: sends requests with forged XFF and checks the rate-limit key does not change; checks an external IPv4 and IPv6 request without the cookie gets 401 on stage.
5. Deploy to stage with /deploy-stage. Beta deploy only after the user says yes in this session (AskUserQuestion).

## Out of scope
- Production vhosts (T-M12-09).
- Server firewall changes — list them for the user instead.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Smoke script green on stage (and on beta once the user OKs it).
- Access log on stage shows real client IPs, not CDN edges or 172.x.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).

## Blocked (2026-11-18)
Files done and tested offline (nginx -t, docker smoke, Go tests). Waiting for the user's OK to (1) sync `deploy/` +
compose to the shared prod proxy (`NO_BUILD=1 ./deploy.sh` proxy reload — touches prod hosts), (2) `/deploy-stage`,
(3) beta deploy, then run `deploy/smoke/forged-xff.sh` on stage/beta. Firewall/IPv6 items for the user are in PROGRESS.
