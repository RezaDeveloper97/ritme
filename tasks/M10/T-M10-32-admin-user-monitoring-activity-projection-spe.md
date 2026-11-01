---
id: T-M10-32
title: Admin user monitoring: activity projection, spec churn, segments, reference codes, engine card
milestone: M10
type: backend
status: done
depends_on: [T-M10-25, T-M10-30, T-M10-03, T-M9-39]
parallel_group: M10-D
touches: [backend-go/internal/admin/users, backend-go/internal/admin/segments, backend-go/internal/messages/facts/account.go, backend-go/internal/auth/handlers.go, backend-go/db/queries/admin, backend-go/db/queries/auth, docs/engine-v6/jobs.md, backend-go/internal/http/routes_admin_core.go, backend-go/internal/http/routes_auth.go, backend-go/internal/http/routes_messages.go, backend-go/internal/messages/source.go, backend-go/internal/messages/handlers.go, backend-go/internal/messages/v2.go, backend-go/internal/admin/messages/testdata/facts_registry.json, docs/go-migration/admin-api.md]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/admin/... ./internal/messages/... && make test-int PKG="./internal/admin/users/... ./internal/admin/segments/..."
---

# T-M10-32 — Admin user monitoring: activity projection, spec churn, segments, reference codes, engine card
**Model:** opus (admin access to user data; privacy).

## Why
SPEC §5.2 (gap 06 §2.5): the directory needs mode, data_quality and the engine's main warning; churn must follow the
spec (high if logged days in the last 14 ≤ 50 % of the 14 before, or 7 days without activity; medium if ≤ 75 %), today
< 30 % / < 70 % with no inactivity rule (`v2.go:119-140`); filters must be saveable as segments (for messages and
analysis); user detail must show today's engine output, cycle-length history and the messages shown with the rule
that triggered them; the RT code is the hex of the id (enumerable); every page aggregates 30 days of logs for all
users (`v2.go:75-86`); raw SQL breaks the sqlc rule (`v2.go:276-282`). DECISIONS.md A-08, A-09; D-181.

## Scope
1. Job `admin.user_activity` (nightly, worker): fills `admin_user_activity` from `user_activity_days` (last activity,
   log days 30 / recent 14 / prior 14), churn (A-08; new accounts younger than 14 days → `new`), mode, plan, and
   data_quality + main warning from today's `engine_daily_snapshots` row.
2. Directory and count read the projection with indexed filters (mode, plan, churn, data_quality, main warning, last
   activity); no per-request aggregate; raw SQL moved to sqlc.
3. Reference code (A-09): `RT-` + first 6 base32 chars of HMAC-SHA256(user_id, `ADMIN_REF_SECRET`) — on a unique
   collision extend to 7, then 8 chars; set at signup and by a one-off backfill job; search by code; the secret is
   required outside local/testing (fail closed).
4. Segments: `/api/admin/v1/segments` CRUD (users manage capability; marketing per T-M10-33), filters in the directory
   vocabulary; the `segment` message fact (T-M10-03) evaluates segment filters against the user's projection row.
5. User detail: today's engine summary (from the snapshot) and cycle-length history (lengths only), recent messages
   and challenges with rule title and version; symptoms, labs, notes and assistant chats stay behind the clinical
   support grant (unchanged).

## Tests
- Churn boundaries: recent/prior = 5/10 → high, 6/10 → medium, 7.5/10 → medium, 8/10 → low; 7 days inactive → high;
  account 10 days old → new.
- Filters and segment save/apply; the `segment` fact true for a matching user.
- Reference code: deterministic, not derivable from the id without the secret, collision path extends the length.
- Directory with 10 000 seeded users ≤ 3 queries per page.
- User detail without a health grant shows no health values.

## Out of scope
- Admin-web screens (T-M10-35).

## Acceptance
- Verify passes; PROGRESS.md cites D-181.

## Note from T-M10-33 (2026-10-30)
- SPEC §5.8 says support/analyst see "masked users", but the legacy admin `/users` list and detail (`backend-go/internal/admin/users/users.go` ~L114, L209) still return the full `mobile` to anyone with `users:view`. Mask it here (or add a `users:unmask` capability) and update the admin-web users screens.
