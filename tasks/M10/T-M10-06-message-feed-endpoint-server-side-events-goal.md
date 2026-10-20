---
id: T-M10-06
title: Message feed endpoint, server-side events, goals, rollup and retention
milestone: M10
type: backend
status: todo
depends_on: [T-M10-05, T-M9-11, T-M9-42]
parallel_group: M10-B
touches: [backend-go/internal/messages/feed, backend-go/internal/messages/events, backend-go/internal/messages/v2.go, backend-go/internal/messages/handlers.go, backend-go/internal/http/routes_messages.go, backend-go/db/queries/messages, backend-go/db/migrations, backend-go/internal/admin/messages/rules.go, backend-go/internal/cycle/periods, backend-go/internal/healthlog, backend-go/internal/checkups/service.go, backend-go/internal/plus/service.go, backend-go/internal/challenges/service.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist/messages.yaml, docs/engine-v6/jobs.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test -race ./internal/messages/... ./internal/admin/messages/... && make test-int PKG="./internal/messages/... ./internal/admin/messages/..." && make contract ROUTES=messages
---

# T-M10-06 — Message feed endpoint, server-side events, goals, rollup and retention
**Model:** opus.

## Why
Each `/messages/selected` call rebuilds facts and history for one placement, so a home with six slots would cost
about 6 × (6 + 2N) queries; the server logs nothing (`Selected` has "no side effects", `v2.go:79-119`), so caps and
CTR depend on the client; event inserts are synchronous; `message_events` grows forever and the admin CTR query scans
it (gap 03 §2.9, rows 1j, 3g, 9). SPEC §5.4 step 7 says the server logs `eligible` and `shown`; §5.6 wants CTR per
version. DECISIONS.md M-12, M-13, M-14, M-15, M-17, M-19; D-163.

## Scope
1. `GET /api/v1/messages/feed?slots=...` (default: urgent_modal, status_banner, smart_tip, daily_advice,
   daily_challenge, modal) → `{data: {slots: {<slot>: [{id, rule_key, version, variant, kind, level, title, body,
   cta_text, cta_deeplink, dismissible, evidence?}]}, alerts: {open, urgent}, rules_rev}}`. One fact prefetch for the
   union of the requested slots, one history query, `Cache-Control: private, no-store`. A slot whose module is switched
   off (app modules) returns `[]`.
2. Server-side `shown`: logged once per (user, message, slot, user-local day) (M-19) — Redis `SET NX` key
   `ritme-go:msg:shown:{uid}:{rule}:{slot}:{day}` (26 h TTL) with a DB check when Redis is down. Sampled `eligible`
   (M-12): the evaluated top 5 per slot; product/marketing sampled 10 % by a stable per-user hash.
3. `events.Writer`: buffered channel (10 000), multi-row INSERT every 1 s or 500 rows, drain on shutdown through the
   lifecycle hooks; a full buffer falls back to a synchronous insert (never drops) and counts it; metrics for queue
   depth, flush latency and fallbacks.
4. `POST /api/v1/messages/events`: accepts `clicked`, `dismissed`, `goal` and the old names (`click`, `dismiss`,
   `impression` — the last one is ignored when the server already logged `shown` that day); validates rule, version
   and slot; rate-limited per user.
5. Goals (M-17): `messages.Goal(ctx, userID, event)` called after commit from the catalogue's write paths
   (`period_logged` — cycle/periods; `log_saved` — healthlog; `checkup_done` — checkups; `plus_purchased` — plus;
   `challenge_joined` — challenges; one call each, nothing else changes in those packages). It logs `goal` for every
   rule with that `goal_event` shown to the user in the last 7 days, once per shown window.
6. `/messages/selected?placement=` becomes a wrapper over the feed for one slot (same response shape).
   `/messages/daily` unchanged (M-15).
7. Status banner copy (M-14): data migration seeding one approved `engine` rule in `status_banner` per
   `today_ui.banner` value, `rule_key = banner.<value>`, condition `today_ui.banner = <value>`, fa/en copy and CTA
   deep links identical to T-M9-42's built-in copy (`frontend/messages/{fa,en}/home.json`).
8. Jobs (T-M9-11 scheduler, queue `low`, unique per day, worker): `messages.rollup` (yesterday's raw rows →
   `message_event_daily`, idempotent upsert) and `messages.prune` (raw rows older than 180 days in batches of 5 000,
   M-13); catalogue entries in `docs/engine-v6/jobs.md`.
9. Admin rules list: CTR per version and variant from the rollup plus today's raw rows (replaces the 7-day scan in
   `admin/messages/rules.go:386-406`).
10. OpenAPI for the feed and events; contract case for the feed (Go-only route, D-163).

## Tests
- Integration: the feed returns per-slot winners and writes `shown` rows after a flush; a second fetch the same local
  day adds no `shown`; the next local day adds one; `eligible` sampling is stable for a user.
- Shutdown drains the buffer (no lost rows); a full buffer takes the synchronous path.
- Events API: aliases, bad version → 422, rate limit.
- Goal: matched 6 days 23 h after the last shown; not matched at 7 days + 1 s.
- A switched-off module empties its slot; `/messages/selected` contract unchanged.
- Rollup is idempotent (run twice → same counts); prune leaves the rollup intact.
- Banner seed: each `today_ui.banner` value of the engine has exactly one approved rule.

## Budgets
- Feed with a warm engine state and 200 active rules ≤ 20 ms p95 and ≤ 3 queries plus auth.

## Out of scope
- The alerts inbox (T-M10-07), the home UI (T-M10-14), CTR alerts (T-M10-31).

## Acceptance
- Verify passes; feed budget recorded in perf-baseline.md; PROGRESS.md cites D-163.
