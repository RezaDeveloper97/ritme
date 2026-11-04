---
id: T-M11-17
title: Frontend people hub, add-member flow, access matrix and review card (owner side)
milestone: M11
type: frontend
status: todo
depends_on: [T-M11-07, T-M11-08]
parallel_group: M11-W
touches: [frontend/src/screens/companion-list, frontend/src/screens/companion-invite, frontend/src/screens/companion-detail, frontend/src/features/invite-companion, frontend/src/entities/companion, frontend/src/entities/family, frontend/src/app/[locale]/companions, frontend/src/app/message-scopes.ts, frontend/messages/fa/companions.json, frontend/messages/en/companions.json, backend-go/resources/translations/fa/companions.json, backend-go/resources/translations/en/companions.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- companions
---

# T-M11-17 — Frontend people hub, add-member flow, access matrix and review card (owner side)
**Model:** sonnet; the main session reviews every screenshot and the privacy copy.

## Why
SPEC §1.8 Me_People, Fam_Hub, Fam_AddMember and Fam_Access: one list for everyone, relation → quick-start template →
editable rows → optional "ask them too" → review, consent and invite; a domain × level matrix with disallowed levels
disabled, the urgent switch, duration, the "never shared" list and removal; plus the review card after a mode change.
The current screens follow the Hamdam artboards for the legacy five sections. DECISIONS.md F-11, F-13, F-22, F-23.

## Scope
1. `entities/family`: types and API for `/api/v1/family/*` (me, members, review, reapprove, confirm, share requests);
   when the API answers 404 (Laravel production) the screens keep today's `/companions/*` behaviour.
2. Hub (`/companions`): members with role, access summary and status (`awaiting_owner` → «تأیید کن»), incoming share
   requests, the review card with «مثل قبل بماند» (one tap) and per-domain choices, children with their guardians.
3. Add member (`/companions/new`): relation → template → matrix → "ask them too" (domains of their own meds and
   appointments) → review with the consent text version → code or phone; carer duration defaults (F-13).
4. Access (`/companions/[id]`): matrix with disabled levels per domain and role, urgent switch and SMS switch,
   duration, «هرگز به اشتراک گذاشته نمی‌شود» list (static, from family.md), remove member.
5. Hamdam artboards (`docs/design/night-bloom/companion-family/`) for existing parts; new parts in the same language
   (F-22). Light and dark; RTL; fa/en through `copy:sync` and the backend translation copy.

## Tests
- Vitest: matrix rules (disabled levels, template defaults, male-owner domains), request builders, the 404 fallback,
  the review card's keep-as-before set.
- Headless screenshots light and dark of hub, add flow, access and review, saved under `docs/engine-v6/qa/m11/`.

## Out of scope
- Member side (T-M11-18, T-M11-20).

## Acceptance
- Verify passes; screenshots reviewed by the main session.
