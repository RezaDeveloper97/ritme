---
id: B-N1-12b
title: Admin — support reports inbox and support info group
milestone: N1
type: fullstack
status: done
depends_on: [B-N1-12]
parallel_group: N1-L2
touches: [admin-web/src,backend-go/internal/admin,backend-go/internal/profile,backend-go/api,backend-go/contract]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd ../admin-web && npm run typecheck && npm run lint && npm run build
---

# B-N1-12b — Admin — support reports inbox and support info group

## Why
B-N1-12 added `support_reports` (screenshot in private storage) and a `support` info-sections group, but admins cannot read reports and the admin-web label for the new group / Laravel `InfoSection::GROUPS` are missing; seeded boxes use fixed `key`s admins cannot set.

## Scope
- Admin API (role-gated): list support reports (paginated, newest first, status open/resolved), detail with the screenshot streamed from private storage (never public), mark resolved. No health payload in the list.
- admin-web: «گزارش‌های مشکل» page (list + detail + resolve), `support` group label in the info-sections editor, allow editing `key` on boxes (optional, validated slug).
- Contract case + OpenAPI.

## Out of scope
- 

## Acceptance
- Admin can list/view/resolve reports; screenshot served only to authenticated admins
- admin-web light + dark (current admin theme) screenshots in `docs/qa/bloom/B-N1-12b/`
- `verify` green
