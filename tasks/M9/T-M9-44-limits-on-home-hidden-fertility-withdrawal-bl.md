---
id: T-M9-44
title: Limits on Home: hidden fertility, withdrawal bleed, postpartum, 45+, pregnancy prompt
milestone: M9
type: frontend
status: todo
depends_on: [T-M9-43]
parallel_group: M9-C
touches: [frontend/src/screens/home, frontend/src/screens/onboarding-flow/ui/ReadyStep.tsx, frontend/src/screens/notification-settings, frontend/messages/fa/home.json, frontend/messages/en/home.json, backend-go/resources/translations/fa/home.json, backend-go/resources/translations/en/home.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- home
---

# T-M9-44 — Limits on Home: hidden fertility, withdrawal bleed, postpartum, 45+, pregnancy prompt
**Model:** sonnet; the main session writes the explanation lines (medical tone).

## Why
§2.40 forbids showing the fertile window or a fertility level under hormonal contraception, but Home, the onboarding
ReadyStep and the fertile-window reminder still show or compute them; there is no withdrawal-bleed wording, no
postpartum/45+ explanation and no "switch to pregnancy mode" prompt after a positive test (gap 02c D.6–D.7; E-13, E-14,
E-24, E-26).

## Scope
1. `isFertilitySuppressed` (T-M9-40) drives the existing `hideFertility` paths on Home: prediction rows, TTC card,
   ring/week dots, fertility chance card.
2. Under hormonal contraception the next-period row reads «خون‌ریزی قطع مصرف» and a one-line explanation appears;
   postpartum-not-returned shows «پیش‌بینی بعد از برگشت دو پریود شروع می‌شود»-style copy (final wording by the main
   session); age ≥ 45 (`perimenopause_age_range`) shows the calm irregularity line.
3. `limits.prompt_pregnancy_mode` → a card «رفتن به حالت بارداری» linking to the mode screen; dismissible for the day.
4. Onboarding ReadyStep hides the fertile window when the user picked a hormonal method.
5. The fertile-window reminder toggle is hidden/disabled while suppressed (server-side skip belongs to the reminder
   sender — record a follow-up if a sender exists).
6. Teen/menopause keep today's no-fertility copy (E-29).

## Tests
- Home view-model: pill, copper IUD, postpartum not returned, age 46, positive test prompt, Laravel payload.
- ReadyStep: hormonal vs non-hormonal choice.
- Visual QA light/dark for pill and postpartum.

## Out of scope
- Calendar (T-M9-45), phase details/companion bar (T-M9-46).

## Acceptance
- Verify passes; no fertile UI on Home for the pill persona in either theme.
