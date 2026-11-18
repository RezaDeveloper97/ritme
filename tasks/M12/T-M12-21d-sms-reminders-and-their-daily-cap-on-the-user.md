---
id: T-M12-21d
title: SMS reminders and their daily cap on the user's local time zone
milestone: M12
type: backend
status: todo
depends_on: [T-M12-21c]
parallel_group: M12-TEST
touches: [backend-go/internal/remindersms]
skills: []
verify: cd backend-go && CGO_ENABLED=0 go test ./internal/remindersms/... && make test-int PKG=./internal/remindersms/...
---

# T-M12-21d — SMS reminders and their daily cap on the user's local time zone

**Model:** opus. Follow-up of T-M12-21c (user decision 2026-11-18: care on the user's local time).

## Why
`internal/remindersms/worker.go` still calls `care.Occurrences` (Tehran) and builds prefs without a zone, and the
daily SMS cap (`CountReminderSMSSince`) counts from Tehran midnight. A Berlin user gets the push at 08:00 Berlin but
the SMS at 08:00 Tehran.

## Scope
1. Per user: `usertz.NewZones(prefs).For(ctx, uid)` → `care.OccurrencesIn(rows, from, to, loc)` and `prefs.InZone(loc)`.
2. Daily cap counted from the user's local midnight.
3. Tests: Berlin user across DST gets the SMS at local 08:00 once; cap resets at local midnight; Tehran unchanged.

## Acceptance
- Tests green, red on the old code; Tehran rows untouched.
