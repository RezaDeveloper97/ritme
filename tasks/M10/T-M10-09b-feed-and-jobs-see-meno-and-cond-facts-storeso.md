---
id: T-M10-09b
title: Feed and jobs see meno.* and cond.* facts (StoreSource implements Menopause/Condition sources)
milestone: M10
type: backend
status: done
depends_on: [T-M10-09]
parallel_group: M10-B
touches: [backend-go/internal/http/routes_messages.go, backend-go/internal/messages/modulefacts_int_test.go, backend-go/internal/messages/feed_int_test.go, backend-go/internal/messages/source.go,backend-go/internal/messages/facts]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/messages/... && make test-int PKG="./internal/messages/..."
---

# T-M10-09b — Feed and jobs see meno.* and cond.* facts (StoreSource implements Menopause/Condition sources)

## Why
Split from T-M10-09: meno.* / cond.* rules exist, but the home feed and push/sweep jobs don't load those facts, so stage tips never fire in the feed and inbox rows are written only when the two GETs are called.

## Scope
- `StoreSource` (internal/messages/source.go) implements `facts.MenopauseSource` and `facts.ConditionSource`.
- Int test: a menopause user's feed shows the daily_advice stage tip without calling the menopause GET.

## Out of scope
- New menopause content (hot-flash counts, PMDD pattern).

## Acceptance
- Feed + sweep evaluate meno.*/cond.* rules.
- verify green. 
