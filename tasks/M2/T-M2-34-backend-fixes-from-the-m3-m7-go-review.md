---
id: T-M2-34
title: Backend fixes from the M3-M7 Go review
milestone: M2
type: backend
status: done
depends_on: [T-M2-31]
parallel_group: M2-J
touches: [backend-go/db/queries,backend-go/internal/store,backend-go/internal/pregnancy,backend-go/internal/profile,backend-go/internal/cycle,backend-go/internal/enums,backend-go/internal/care,backend-go/internal/account,backend-go/internal/auth,backend-go/internal/messages/pregnancyalerts,backend-go/db/migrations,backend-go/api/openapi.yaml,backend-go/contract/allowlist,docs/go-migration]
skills: [verify-all,new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && golangci-lint run && make contract ROUTES=all
---

# T-M2-34 — Backend fixes from the M3-M7 Go review

## Why
`docs/go-migration/backend-review-m3-m7.md` (backend-reviewer, 2026-09-29) found 2 high, 3 med, 9 low issues.

## Scope
Fix every finding in that doc EXCEPT the checkups engine section bug (it is in T-M4-12). In particular:
1. (high) Pregnancy v2 calendar/today drop appointments with `is_active = 0` (the reminder bell toggle) — only
   cancelled appointments may be excluded. Integration test.
2. (high) `POST /profile` without `last_period_start` fabricates a confirmed period dated today. Stop inventing data
   in Go (leave cycle history empty; the frontend already has a "log your period" empty state since T-M7-17).
   Record as a decided deviation D-nn in `docs/go-migration/deviations.md`, allowlist the affected contract cases.
3. (med) `daily_card` fertility level uses the legacy `FertilityLevel()`; switch to `FertilityLevelV11()` so one
   response carries one answer. Deviation D-nn + allowlist.
4. (med) Pregnancy v1 log handlers return 500 when the v2 alert hook fails after the log committed — make the hook
   best-effort (`slog.Warn`), per D-21.
5. All low items (legacy medication PUT/toggle in /care, subtitle language, day-log lower date bound, day-log Save
   transaction, alerts GET not writing, weight_missing_week start, DELETE /account token cleanup atomic, care-item
   delete guard, 00005 Down preserving admin edits — for a migration already applied, only fix Down if safe; otherwise
   document). For any low item you decide not to fix, write why in the review doc.
Keep OpenAPI in sync with any behaviour change.

## Out of scope
- `backend/` (Laravel, frozen) — prod parity handled via deviations.

## Acceptance
- Review doc gets a Resolution column; verify green (incl. contract); integration tests for 1, 2, 4.
