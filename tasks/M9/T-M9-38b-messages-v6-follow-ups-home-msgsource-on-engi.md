---
id: T-M9-38b
title: Messages v6 follow-ups: home msgSource on engine, no-last_period 400 under v6, admin preview user timezone, query budget
milestone: M9
type: backend
status: done
depends_on: [T-M9-38]
parallel_group: M9-C
touches: [backend-go/internal/home/context.go,backend-go/internal/messages/handlers.go,backend-go/internal/admin/messages/rules.go,backend-go/internal/querybudget,backend-go/contract/golden-v6/home/index.overdue_20.fa.json,docs/engine-v6/perf-baseline.md]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/messages/... ./internal/home/... && make test-int PKG="./internal/messages/... ./internal/home/... ./internal/querybudget/..."
---

# T-M9-38b — Messages v6 follow-ups: home msgSource on engine, no-last_period 400 under v6, admin preview user timezone, query budget

## Why
Open items left by T-M9-38 (messages read v6 facts).

## Scope
- `home/context.go` `msgSource` still goes through `v6Calculation` + legacy display window: attach the engine like
  `/messages/daily` so Home's message equals the daily message.
- Under v6, `/messages/daily` returns 400 when the profile has no `last_period_start` even if periods are logged:
  use the engine's data instead of the profile field.
- Admin preview resolves the target user's day in Tehran time: use the user's own timezone.
- Add query-budget entries for messages daily/selected on v6.
- Product review of T-M9-38 calls: late (`period_expected`) day gets luteal content (I-11); `fertility_suppressed`
  true for teen/menopause; pill user with goal TTC gets non-TTC copy. Ask the user if any is wrong.

## Out of scope
- New message content.

## Acceptance
- Home message == daily message for the same user/day under v6 (int test).
- No 400 for a user with logged periods but empty `last_period_start` under v6.
- Query budget test covers both endpoints; verify green.
