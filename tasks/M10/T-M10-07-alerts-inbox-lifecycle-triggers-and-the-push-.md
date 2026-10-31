---
id: T-M10-07
title: Alerts inbox: lifecycle, triggers and the push slot
milestone: M10
type: backend
status: done
depends_on: [T-M10-06]
parallel_group: M10-B
touches: [backend-go/internal/messages/inbox, backend-go/internal/messages/feed, backend-go/internal/http/routes_messages.go, backend-go/db/queries/messages, backend-go/api/openapi.yaml, docs/engine-v6/jobs.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test -race ./internal/messages/... && make test-int PKG=./internal/messages/...
---

# T-M10-07 — Alerts inbox: lifecycle, triggers and the push slot
**Model:** opus (health alerts).

## Why
SPEC §5.4.1: pregnancy alerts (and every other mode's alerts) are messages in slot `alerts_inbox`, listed with a
lifecycle (active → acknowledged / added to visit note / resolved → archived after 30 days), merged per message,
evidence required from follow_up up, urgent not closable until acknowledged, safety also shown in the urgent modal and
engine items in the home banner. Today alerts live in a separate pregnancy engine with one row per reading, no
`resolved`, no archive and no generic API (gap 03 row 6, §2.6). SPEC §5.4 also has a push slot; there is no push
channel, so M-09 delivers it to the in-app notifications inbox. DECISIONS.md M-09, M-16; D-164.

## Scope
1. `inbox.Service.Evaluate(ctx, userID, facts, now)`: run the `alerts_inbox` slot (all eligible) and diff with the
   user's open rows (loaded with the history query):
   - new match → `active` row; level ≥ follow_up without evidence → no row and `inbox_missing_evidence_total` + 1;
   - open row, same `evidence_key` → nothing; new `evidence_key` → `hit_count + 1`, `last_at`, new evidence; an
     acknowledged `urgent` row returns to `active` (M-16);
   - open row whose rule no longer matches and whose facts were all available → `resolved`;
   - rows are written only when something changed (an app open with no change writes nothing).
2. Evaluation runs inside the feed request (T-M10-06) and through `inbox.EvaluateNow(ctx, userID)` for write paths that
   need the alert in their own response (used by T-M10-08).
3. API (Go-only): `GET /api/v1/alerts?state=open|history&cursor=` ordered urgent → follow_up → suggestion → info, then
   `last_at` desc; `POST /api/v1/alerts/{id}/ack`; `POST /api/v1/alerts/{id}/visit-note` (generalise what
   `pregnancyalerts/engine.go:681-726` does today); `POST /api/v1/alerts/{id}/dismiss` → 409 `ack_required` for an
   urgent row not yet acknowledged. Foreign ids → 404. `Cache-Control: private, no-store`.
4. Fan-out: open safety rows appear in the feed's `urgent_modal` until acknowledged; open engine rows in
   `status_banner` (SPEC §5.4.1).
5. Daily job `messages.inbox_sweep` (worker, queue `low`, unique per day): re-evaluate users with open rows (resolve)
   and archive resolved/acknowledged/noted rows whose `last_at` is older than 30 days.
6. Push slot (M-09): hourly job `messages.push_slot` picks users whose local time is 09:00–09:59 and who were active in
   the last 30 days (A-06), evaluates the `push` slot once per local day, and writes `user_notifications` rows through
   the existing notifications service and its user preferences. Lock-screen text stays neutral («یادآور ریتمی») and
   never names a health detail.
7. Catalogue entries in `docs/engine-v6/jobs.md`; OpenAPI.

## Tests
- Merge: same evidence key → unchanged; new key → count 2 and new `last_at`; acknowledged urgent + new key → active.
- Resolve when the condition drops; archive at day 30 (fake clock), not at day 29.
- Urgent dismiss → 409 until acknowledged; ack then dismiss → 200.
- follow_up rule without evidence → no row, metric + 1.
- Ordering; cursor paging; foreign id → 404.
- Fan-out: a safety row is in `urgent_modal` until acknowledged.
- Push: a user at 23:30 local is skipped; 1 per day and 3 per week caps; the notification text contains none of the
  words of a health-term deny-list (fa and en); the job is idempotent within a local day.

## Out of scope
- Pregnancy alert rules and their data migration (T-M10-08), menopause and condition rules (T-M10-09), the UI
  (T-M10-15), real web push.

## Acceptance
- Verify passes; PROGRESS.md cites D-164.
