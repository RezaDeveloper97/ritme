---
id: T-M12-11c
title: Fix the 6 integration tests red on HEAD in care, loss, messages, notifications
milestone: M12
type: backend
status: todo
depends_on: [T-M12-11]
parallel_group: M12-TEST
touches: [backend-go/internal/care,backend-go/internal/loss,backend-go/internal/messages,backend-go/internal/notifications]
skills: []
verify: cd backend-go && make test-int PKG="./internal/care/... ./internal/loss/... ./internal/messages/... ./internal/notifications/..."
---

# T-M12-11c — Fix the 6 integration tests red on HEAD in care, loss, messages, notifications

**Model:** opus. Found by the first full integration + race run (T-M12-11, 2026-11-18); red natively too.

## Why
CI now runs integration tests and blocks the claude/** auto-merge; these six are red on `stage` HEAD and nobody owns
them. Two of them (loss) guard privacy: pregnancy content must stop and erase must be complete after a loss record.

- `care` `TestUpcomingListsDosesAndAppointmentReminders` — 422 validation on `kind` (push_notices_int_test.go:210)
- `care` `TestToday_Fixture` — response gained `amount`/`meal` (today_int_test.go:84)
- `loss` `TestLoss_RecordStopsPregnancyContentDurably` (loss_int_test.go:198) and `TestLoss_EraseAndConcurrency` (:414) — expect 0, get 2
- `messages` `TestFeedInt_BannerSeed` — seed copy changed by the M8 copy pass (feed_int_test.go:384)
- `notifications` `TestShow_DefaultsWithoutRow` — default groups payload changed (notifications_int_test.go:118)

## Scope
1. Per test: find the commit that turned it red; decide regression (fix code) vs intended change (update test +
   fixture/contract golden, citing the commit/decision).
2. The two loss tests: treat as a privacy regression until proven otherwise; fix the code if rows survive.
3. Contract goldens touched → `make contract ROUTES=<group>` green.

## Out of scope
- The `internal/http` companion tests (T-M12-11b).

## Acceptance
- Verify green; PROGRESS.md lists each test → regression/intended + commit.
