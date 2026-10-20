---
id: T-M9-02
title: Family privacy hotfix: never-shared data and stale pregnancy grants
milestone: M9
type: fullstack
status: todo
depends_on: []
parallel_group: M9-A
touches: [backend-go/internal/companion, backend-go/internal/care/medication.go, backend-go/internal/care/appointment.go, backend-go/internal/pregnancy/handlers.go, backend-go/internal/pregnancy/v2, backend-go/internal/loss/service.go, backend-go/internal/postpartum/service.go, backend-go/internal/http/routes_shop_checklist.go, backend-go/internal/shop/checklist/handlers.go, backend-go/db/migrations, frontend/src/entities/companion, frontend/src/screens/companion-home, frontend/messages/fa/companion-home.json, frontend/messages/en/companion-home.json]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/companion/... ./internal/care/... ./internal/loss/... ./internal/postpartum/... && make test-int PKG="./internal/companion/... ./internal/care/... ./db/migrations/..." && cd ../frontend && npm run typecheck && npm run lint && npm run test
---

# T-M9-02 — Family privacy hotfix: never-shared data and stale pregnancy grants
**Model:** opus (health privacy). Run before the other M9 work; it does not wait for the engine.

## Why
Gap report 05 §3 found members can see data SPEC v6 §1.7 lists as **never shared**, and a stale grant leaks a new
pregnancy:
- FAM-H1: `mood` is in `shared.SymptomCategories` (`internal/companion/shared/shared.go:44-45`), so any member with
  the `symptoms` grant sees the owner's mood (also on the companion home, `home/home.go:205-210`).
- FAM-M2: medication `notes` and appointment `notes/topic/result_note/prep` reach members
  (`care/medication.go:142-170`, `care/appointment.go:203-237`).
- FAM-M3: postpartum counts as pregnancy for the pregnancy grant (`shared/mode.go:37-39,51-57`).
- FAM-M1 (part): the member cycle view carries `days_late` up to 14 days — a pregnancy signal (`shared.go:116-137`).
- FAM-H3: a `pregnancy: view` grant is never reset, so after a loss or delivery a member sees the *next* pregnancy
  from its first day (`shared.go:189-212`, `loss/service.go:263-286`, shop checklist `due_date/weeks_left`).
- FAM-L6: member responses have no `Cache-Control: no-store`.
The full family model (domains, `check()`, mode epoch) is M11; this is the minimal fix that closes the leaks now.

## Scope
1. Show the user (Persian) the six changes below and wait for OK (they are spec-mandated; the wider removal of the
   whole `symptoms` section is decision F-03 in M11).
2. Backend:
   1. Remove `mood` (and anything EPDS-derived) from what members receive: `SymptomCategories`, the companion home
      symptom summary and any section read that returns taxonomy entries.
   2. Member projections for meds and appointments become explicit allow-list structs (name/title, form/dose,
      schedule/time, location, doctor/"with", status). No `notes`, `topic`, `result_note`, `prep`, custom labels or
      any other free text. Never serialise the owner's JSON for a member.
   3. The pregnancy section is shown only when the owner's effective mode is pregnancy (not postpartum).
   4. The member cycle view drops `days_late` and every lateness-derived value or copy.
   5. One exported `companion.ResetModeScopedGrants(ctx, ownerID, reason)`: sets every `pregnancy` grant of the
      owner's links to `none`, writes a `grants_changed` audit row with the diff `{pregnancy: [old, none]}` and an
      owner notice through the existing owner-notice path (`companion/links.go`) — copy: «اشتراک بارداری با همراهت
      متوقف شد؛ اگر بخواهی، دوباره روشنش کن.» Call it when a pregnancy is activated (v1 and v2 setup), deactivated,
      on a loss (`loss/service.go`) and on postpartum activation. Run it inside the caller's transaction where one
      exists; otherwise right after commit.
   6. The shop checklist share (`routes_shop_checklist.go:88-94`, `shop/checklist/handlers.go:180-185`) uses the
      same pregnancy check, so a reset grant shows no due date or weeks left.
   7. `Cache-Control: private, no-store` on every member-facing companion response.
   8. Goose **data** migration (next free number; schema unchanged, so no Laravel twin): set to `none` every
      `pregnancy` grant whose `updated_at` is earlier than the start of the owner's current pregnancy episode, and
      every pregnancy grant of an owner who is not pregnant now; write the audit rows. `down` is a no-op (documented).
3. Frontend: the companion parsers and the companion home tolerate the missing mood category, free text and
   `days_late` (no crash, no empty card, no "late" copy on the member side). Keep the Laravel payload working.

## Tests
- `TestMember_MoodNeverShared`: owner logs mood + pain; a member with the symptoms grant gets pain and never mood
  (scan the whole JSON body for the mood key and value).
- `TestMember_MedsAppointments_NoFreeText`: sentinel strings in notes/topic/result_note/prep never appear in any
  member endpoint body.
- `TestMember_PostpartumIsNotPregnancy`; `TestMember_CycleView_NoLateness` (owner 9 days late → no `days_late`, no
  late flag).
- `TestPregnancyGrant_ResetOn{Activate,Deactivate,Loss,PostpartumStart}`: grant → none, audit diff, owner notice;
  `TestShopChecklist_StaleGrantDenied`.
- `TestMemberResponses_NoStore` (header on every member route).
- Migration integration test: a stale grant is reset, a grant given during the current pregnancy is kept.
- Frontend vitest: the companion-side parser with the new payload and with the old one.

## Out of scope
- Domains/levels/`check()`/mode epoch (M11), removing the whole symptoms section (F-03), invites, proposals.

## Acceptance
- User OK recorded in PROGRESS.md; every test above green; verify passes.
- PROGRESS.md cites D-133. Ask the user whether to deploy this fix to stage and beta now.
