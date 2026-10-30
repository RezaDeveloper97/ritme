---
id: T-M11-06
title: Member projections v6: allow-list DTOs per domain, symptoms section removed, effective grants only
milestone: M11
type: backend
status: done
depends_on: [T-M11-05]
parallel_group: M11-C
touches: [backend-go/internal/companion/member_privacy_int_test.go,backend-go/internal/http/companion_security_int_test.go,backend-go/internal/http/routes_companion_home_int_test.go,backend-go/internal/http/routes_companion_int_test.go,backend-go/internal/http/routes_loss_int_test.go,backend-go/internal/http/routes_teen_int_test.go,backend-go/internal/family/projections, backend-go/internal/companion/shared, backend-go/internal/companion/home, backend-go/internal/companion/handlers.go, backend-go/internal/shop/checklist/handlers.go, backend-go/internal/http/routes_family.go, backend-go/internal/http/routes_shop_checklist.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/family/... ./internal/companion/... && make test-int PKG="./internal/family/... ./internal/companion/... ./internal/http/..." && make contract ROUTES=companion,companion-home,shop-checklist
---

# T-M11-06 — Member projections v6: allow-list DTOs per domain, symptoms section removed, effective grants only
**Model:** opus (health privacy).

## Why
SPEC §1.7: members get summaries only, and private notes, mood/EPDS, sex, labs, chats, full cycle history and
fertility data are never shared. T-M9-02 removed mood, free text and lateness; the v6 domains (vitals, fetal
activity, TTC switch) do not exist, the `symptoms` section has no v6 domain, and the member grant list still shows
every section (gap 05 §2.2, §2.4, §2.10 item 5; FAM-M1). DECISIONS.md F-01, F-02, F-03, F-21; D-194.

## Scope
1. `GET /api/v1/family/owners/{ownerId}/domains/{domain}` (member), each domain an explicit allow-list struct:
   - cycle_phase: `main_phase` from the engine's public phase (T-M9-36), `period_soon` (≤ 3 days) for the partner role
     only (F-01); the fertile label hidden when the engine reports fertility suppressed (hormonal contraception); in
     TTC mode window, ovulation day and best days only through `ttc_fertility` for partners (F-02);
   - pregnancy: week and approximate due date (pregnancy mode, current epoch);
   - appointments: title, time, place, "with", status;
   - meds: name, form and dose, schedule times; today's dose status only at `nudge` or above;
   - vitals: latest weight, BP and glucose each as value + category + date (no history);
   - fetal_activity: today's kick-count summary and the last contraction session's 5-1-1 status (pregnancy only).
2. Remove the `symptoms` section, its home card and every reader (F-03); `/companions/links/:id/sections/symptoms` →
   404.
3. `/companions/links/:id/sections/:section` maps legacy names onto the new DTOs (differences allow-listed, D-194).
4. Shop checklist share: check() on `pregnancy` (epoch-bound).
5. The member's link view lists effective domains only.
6. `Cache-Control: private, no-store` on every `/family/*` response.

## Tests
- `TestMember_NeverSharedNotReachable`: seed sentinel values for notes, mood, EPDS, sex, labs, chats, cycle length and
  history, TTC data; walk every member endpoint (family and companion) → no sentinel in any body.
- `TestCyclePhase_NoNumbers`, `TestCyclePhase_PeriodSoonPartnerOnly`,
  `TestFertileLabel_HiddenUnderHormonalContraception`, `TestTTCSwitch_PartnerOnly`.
- `TestPregnancyProjection_ModeAndEpochOnly`, `TestShopChecklist_StaleGrantDenied`, `TestVitalsProjection_LatestOnly`,
  `TestFetalActivity_PregnancyOnly`, `TestMeds_DoseStatusOnlyAtNudge`, `TestMemberLinkView_EffectiveDomainsOnly`.

## Out of scope
- Member home (T-M11-13), household (T-M11-14), Plus for members (T-M11-16).

## Acceptance
- Verify passes; PROGRESS.md cites D-194.
