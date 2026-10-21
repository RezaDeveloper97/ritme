---
id: T-M9-13
title: Config caches with pub/sub invalidation and one query for rule history
milestone: M9
type: backend
status: done
depends_on: [T-M9-07]
parallel_group: M9-B
touches: [backend-go/internal/platform/configcache, backend-go/internal/messages/content, backend-go/internal/messages/rules, backend-go/internal/cycle/recommendation, backend-go/internal/challenges, backend-go/internal/plus, backend-go/internal/modules, backend-go/internal/home/articles.go, backend-go/internal/admin, backend-go/db/queries/messages]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/platform/configcache/... ./internal/messages/... ./internal/cycle/recommendation/... ./internal/challenges/... ./internal/plus/... ./internal/modules/... ./internal/admin/... && make test-int PKG="./internal/platform/configcache/... ./internal/messages/..."
---

# T-M9-13 — Config caches with pub/sub invalidation and one query for rule history
**Model:** opus.

## Why
Small admin-edited tables are re-read on almost every request: `message_contents` (whole table per locale),
`recommendations`, `message_rules`, task templates, challenges, affirmations, home articles, plus plans, languages,
app modules (gap 07 §1 item 3, §7.2). `/home` costs ≈ 19 queries and `/messages/selected` 8 + 2 per rule (the rule
history N+1 in `rules.go:262,271`).

## Scope
1. `internal/platform/configcache`: generic `Cache[T]` = atomic snapshot + loader + content fingerprint; `Get(ctx)`
   is lock-free; reload on (a) local invalidation after an admin write commits, (b) Redis pub/sub message on
   `ritme-go:config-changed` naming the table, (c) a 60 s TTL safety net. Singleflight on reload. Redis down → TTL only.
2. Apply it to: message contents (per locale), recommendations, message rules (raw rows; compiling them is
   T-M10-05), task templates, challenges, affirmations, home articles, plus plans, app modules. Every admin writer of
   these tables publishes after commit (find them under `internal/admin/**`).
3. Language registry and modules: replace the per-request Redis GET + JSON decode with the in-process snapshot.
4. `ListUserMessageEventStats(uid, rule_ids)`: one grouped query (last shown, last dismissed, counts per rule and
   event within the largest cap window) replacing the per-rule probes in `rules.Select`.
5. Expose `configcache.Fingerprint(tables…)` for ETags (T-M9-31).

## Tests
- Unit: concurrent `Get` during reload (race detector); TTL reload; fingerprint changes only when content changes.
- Integration: an admin edit is visible on the next request of the same process (local) and of a second process
  (pub/sub), and after 60 s with pub/sub disabled.
- `/messages/selected` with 50 active rules → constant query count (query budget updated); `/home` budget lowered.
- `rules.Select` bench with prefetched stats ≤ 200 µs.

## Out of scope
- Compiled rules (T-M10-05), the cycle engine cache (T-M9-31).

## Acceptance
- Budgets in `internal/querybudget` lowered and green; perf-baseline.md updated.
