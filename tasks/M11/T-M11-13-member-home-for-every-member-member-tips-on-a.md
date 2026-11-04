---
id: T-M11-13
title: Member home for every member, member tips on a member-safe fact set, member notifications
milestone: M11
type: backend
status: todo
depends_on: [T-M11-06, T-M11-10, T-M11-12, T-M11-14, T-M10-04, T-M9-39]
parallel_group: M11-C
touches: [backend-go/internal/family/home, backend-go/internal/family/notices, backend-go/internal/companion/home, backend-go/internal/messages/facts/member.go, backend-go/internal/messages/rules, backend-go/internal/http/routes_family.go, backend-go/internal/http/routes_companion_home.go, backend-go/db/queries/companion, backend-go/api/openapi.yaml, backend-go/contract/allowlist, docs/engine-v6/jobs.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/family/... ./internal/messages/... ./internal/companion/... && make test-int PKG="./internal/family/... ./internal/companion/home/... ./internal/http/..." && make contract ROUTES=companion-home
---

# T-M11-13 — Member home for every member, member tips on a member-safe fact set, member notifications
**Model:** opus (what a member learns about the owner).

## Why
SPEC §1.8 Fam_PHome: a home per owner mode for every member (pregnancy summary, today's family tasks, appointments
with "I'm coming", support suggestions from `member_tip`, household list) and member notifications (phase or week
change, appointment tomorrow, missed dose at nudge level, urgent), all neutral on the lock screen. Today the companion
home is male-only, tips come from a fixed content group and there are no member notifications (gap 05 §2.13, FAM-M8).
DECISIONS.md F-01, F-10; D-201.

## Scope
1. `GET /api/v1/family/home` (any member, no gender gate): per owner — mode-aware summary from the projections
   (phase or pregnancy week), today's family tasks (household items due today, doses at nudge level, appointments today
   and tomorrow), appointments with attendance, support tips, shared-children cards (guardianship), unseen alert count.
   One authorization statement for all owners; projections batched.
2. `POST` / `DELETE /api/v1/family/appointments/{id}/attend` (appointments ≥ view; foreign id → 404); her appointment
   shows who is coming.
3. Member tips: the `member_tip` slot (T-M10-04) evaluated on a member fact set built only from facts registered
   `MemberSafe` **and** derived from domains this member holds now (e.g. `owner_main_phase` needs cycle_phase,
   `owner_pregnancy_week` needs pregnancy). The rule validator rejects a member_tip rule that names any other fact; the
   current `companion_tip` contents become seeded content-kind member_tip rules.
4. Member notifications (outbox + dispatcher jobs): phase change and pregnancy week change from `engine_daily_snapshots`
   (T-M9-39; no engine run), appointment tomorrow, missed dose (nudge level only), urgent (T-M11-12). Neutral title
   «پیامی از خانواده در ریتمی»; the member's `companion` category and quiet hours respected (deferred, dropped when
   stale); dedupe per (member, kind, subject, local day).
5. `/companion/home` becomes an alias of the new payload's compatible subset (allow-listed, D-201).

## Tests
- `TestFamilyHome_FemaleMemberAllowed`, `TestFamilyHome_QueryBudget` (3 owners ≤ 6 statements),
  `TestMemberTip_NoHiddenFieldConditions`, `TestMemberTip_ValidatorRejectsUnsafeFact`,
  `TestMemberNotices_NeutralAndQuietHours`, `TestMemberNotices_PhaseChangeFromSnapshots`,
  `TestMemberNotices_MissedDoseNudgeLevelOnly`, `TestAttend_IDOR`, `TestCompanionHome_AliasSubset`.

## Budgets
- Member home with 3 owners: p95 ≤ 120 ms cold locally, ≤ 6 statements.

## Out of scope
- Screens (T-M11-20).

## Acceptance
- Verify passes; PROGRESS.md cites D-201.
