---
id: T-M9-03
title: Quick fixes: pregnancy_week fact, cycle cache key, language registry TTL
milestone: M9
type: backend
status: todo
depends_on: []
parallel_group: M9-A
touches: [backend-go/internal/messages/v2.go, backend-go/internal/cycle/service/service.go, backend-go/internal/cycle/cache, backend-go/internal/i18n/registry.go, backend-go/internal/content/content.go]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/messages/... ./internal/cycle/... ./internal/i18n/... ./internal/content/...
---

# T-M9-03 — Quick fixes: pregnancy_week fact, cycle cache key, language registry TTL
**Model:** opus (small, but `pregnancy_week` feeds pregnancy messages).

## Why
Three independent bugs from the gap reports, each a few lines:
- `pregnancy_week = today.DiffDays(lmp)/7 + 1` (`internal/messages/v2.go:73`); `DiffDays` returns `other − d`, so the
  week is negative for every pregnant user in `/messages/selected` and the admin rule preview (gap 02 IC-13).
- B1: the cycle cache key omits the clinical-config fingerprint, so an approved `engine.*` change is served stale
  until Tehran midnight (`cycle/service/service.go:191-215`; `clinicalconfig.Fingerprint` exists, `store.go:168`).
- B5: `languages.registry` is written to Redis with no TTL (`i18n/registry.go:166`), so a direct DB edit never refreshes.

## Scope
1. `pregnancy_week`: read the gestational week from the pregnancy dating code already used by the pregnancy screens
   (`internal/pregnancy/calc` / v2 dating), so the message facts and the pregnancy UI can never disagree. Same for
   any trimester fact if present.
2. Cache key: add the fingerprint of the `engine.*` clinical-config keys valid on the target day; bump the cache schema
   constant (3 → 4). This is an interim fix; T-M9-31 replaces the cache.
3. Registry: set a TTL (10 min) on the Redis write; keep `Flush` on admin writes; drop the per-route `EXPIRENX`
   workaround in `content/content.go:117-120` only if it becomes redundant.

## Tests
- LMP 70 days before today → week 11; LMP today → week 1; a v2 rule `pregnancy_week between 10 and 12` matches; the
  admin preview path returns the same week.
- Changing an `engine.*` value in the config test store changes the cycle cache key; changing a vitals key does not.
- miniredis: the registry key has a TTL > 0 after a load.

## Out of scope
- The message fact registry (T-M10-03), the engine cache redesign (T-M9-31).

## Acceptance
- Tests green; verify passes; no contract golden changes (the fixed fact is not in any Laravel golden — confirm with
  `grep -r pregnancy_week backend-go/contract/golden`).
