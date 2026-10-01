---
id: B-N2-02
title: Onboarding flow v2 (women path)
milestone: N2
type: frontend
status: done
depends_on: [B-N2-01,B-N1-03]
parallel_group: N2-B
touches: [frontend/src/screens/onboarding-*,frontend/src/screens/auth-signup,frontend/src/screens/auth-otp,frontend/src/app/[locale]/onboarding,frontend/src/features/auth]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N2-02 — Onboarding flow v2 (women path)

## Why
Replace the 11-step legacy onboarding with the designed flow.

## Design
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Phone.dc.html` (+ `nbd_Onb_Phone`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_OTP.dc.html` (+ `nbd_Onb_OTP`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Name.dc.html` (+ `nbd_Onb_Name`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Gender.dc.html` (+ `nbd_Onb_Gender`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Goal.dc.html` (+ `nbd_Onb_Goal`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Cycle.dc.html` (+ `nbd_Onb_Cycle`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Preg.dc.html` (+ `nbd_Onb_Preg`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Meno.dc.html` (+ `nbd_Onb_Meno`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Conditions.dc.html` (+ `nbd_Onb_Conditions`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Health.dc.html` (+ `nbd_Onb_Health`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Ready.dc.html` (+ `nbd_Onb_Ready`)

## Scope
- Phone (+98, terms + health-data consent checkboxes), OTP (5 digits, WebOTP autofill, resend timer), Name, Gender, Goal (cycle/ttc/pregnant/menopause) → branch: Cycle (calendar + period/cycle steppers, «دقیق یادم نیست»), Preg (dating basis: LMP/ultrasound/due date/current week), Meno (stage + questions) → Conditions → Health → Ready.
- Male choice routes to the partner-code screen (stub until B-N4-05).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- Each goal branch completes and lands on the right home
- `verify` green
