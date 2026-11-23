---
id: T-M12-30c
title: Pregnancy alert evaluation off the hot read path without missing time-based alerts
milestone: M12
type: backend
status: done
depends_on: [T-M12-30b]
parallel_group: M12-PERF
touches: [backend-go/internal/messages/pregnancyalerts,backend-go/internal/messages/inboxalerts,backend-go/internal/http/routes_pregnancy_v2.go,backend-go/internal/pregnancy,backend-go/internal/querybudget]
skills: []
verify: cd backend-go && make test-int PKG="./internal/messages/pregnancyalerts/... ./internal/messages/inboxalerts/... ./internal/pregnancy/... ./internal/querybudget/..."
---

# T-M12-30c — Pregnancy alert evaluation off the hot read path without missing time-based alerts

**Model:** opus (safety). Follow-up of T-M12-30b.

## Why
`/pregnancy/v2/today` = 27 cold / 21 warm queries; 15 of them are `al.EvaluateFirst` (alerts) on every request.
Caching "once a day" or "only when inputs changed" would miss time-based urgent alerts (H-05 kick window ends with no
write; 5-1-1 contractions, week_entered, next_visit).

## Scope
1. The alert engine returns a per-user cache key `(input revision, next_due_at)` — next_due_at = earliest instant a
   time-based rule may flip (kick window end, day/week change, visit due).
2. `EvaluateFirst` skips only when revision unchanged AND now < next_due_at.
3. Revision bumps on every relevant save (day log, v1 saves, kicks, contractions, vitals, ack).
4. Tests: urgent alert right after a qualifying log; kick alert after the 2 h window expires with no write; budget
   lowered (target ≤ 8 warm on a repeat read).

## Acceptance
- Safety tests green and red if the skip ignores next_due_at; budget tightened.
