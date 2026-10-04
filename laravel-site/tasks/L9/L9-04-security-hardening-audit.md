---
id: L9-04
title: Security hardening + audit
milestone: L9
type: quality
status: todo
depends_on: [L6-06,L5-06,L4-05,L7-04]
parallel_group: L9-C
touches: [app,config,public/.htaccess,docs/SECURITY.md]
skills: []
verify: composer verify
---

# L9-04 — Security hardening + audit

## Scope
- Run the `security-auditor` agent over `laravel-site/` and fix findings: CSP verified on every template (no
  `unsafe-inline`), upload validation (mime sniff, image re-encode, SVG sanitise), HTML sanitiser on every rich field,
  mass-assignment (`$fillable`/DTOs), authorization policies on every Filament resource, rate limits on all public
  POSTs, signed/unguessable codes for booked/order pages, session/cookie flags (secure, httponly, samesite=lax),
  `.env`/`storage`/`vendor` not web-reachable (cPanel `public_html` layouts too), admin path + MFA, `composer audit`
  and `npm audit` clean, error pages leak nothing with `APP_DEBUG=false`.
- `docs/SECURITY.md` checklist.

## Acceptance
- No high/critical findings open; `composer audit` / `npm audit --omit=dev` clean.
