---
id: T-M12-34
title: ETag/304 for slow-changing endpoints (modules, banners, taxonomy, enums, courses, checkups home)
milestone: M12
type: backend
status: done
depends_on: [T-M12-30]
parallel_group: M12-PERF
touches: [backend-go/internal/platform/etag, backend-go/internal/http, backend-go/loadtest/web_home.js]
skills: []
verify: cd backend-go && go vet ./... && make test-int PKG="./internal/http/..."
---

# T-M12-34 — ETag/304 for slow-changing endpoints (modules, banners, taxonomy, enums, courses, checkups home)
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
P-10: these responses rarely change but are re-sent in full on every open.

## Scope
1. Reuse `platform/etag`; the ETag must vary by user where the body does (modules per user, banners per segment, locale) — never serve one user's body to another.
2. Int tests: same body → 304 with `If-None-Match`; admin edit → new ETag; two users with different bodies → different ETags.
3. k6 web_home sends `If-None-Match` on the second iteration; record the 304 ratio.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Tests green; 304 ratio and bytes saved in perf.md.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
