---
id: B-N4-10b
title: N4 stage smoke follow-ups (share children UI, male ready copy, pregnancy half-switch, list comma, admin label)
milestone: N4
type: frontend
status: done
depends_on: [B-N4-10,B-N5-05]
parallel_group: N4-K2
touches: [frontend/src/features/invite-companion,frontend/src/screens/companion-detail,frontend/src/screens/onboarding-flow,frontend/src/screens/mode,frontend/src/screens/pregnancy-onboarding,frontend/src/screens/companion-home,admin-web/src/screens/companions]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build && cd ../admin-web && npm run typecheck && npm run lint
---

# B-N4-10b — N4 stage smoke follow-ups (share children UI, male ready copy, pregnancy half-switch, list comma, admin label)

## Why
Findings of the N4 stage e2e (`docs/qa/bloom/n4-stage.md`).

## Scope
- **B-1 (medium):** owners can't share children from the UI — replace the spouse wizard children step placeholder (`InviteCompanionFlow` `ChildrenStep`) with a real picker of the owner's children (`entities/child` `useChildren`, `child_ids` on create) and add a children editor on `/companions/[id]` (`PUT /companions/{id}/children`).
- **B-2:** male «ready» onboarding screen shows women's copy — male-specific copy.
- **B-3:** pregnancy setup step 1 already stores `stored_mode: pregnancy`; leaving the wizard leaves a half switch — only PUT the mode on the final step (or roll back on exit); reload resumes correctly.
- **B-4:** «به اشتراک گذاشته نشده» list joins with «، و» — use proper Persian list formatting (Intl.ListFormat or copy).
- **B-6:** after «بعداً وصل می‌شوم» a direct `/fa/companion` landed on `/onboarding/name` once — refresh onboarding/profile caches before navigating.
- **B-7:** admin calls partner «همراه» while the app says «پارتنر» — align admin label.
- I-1: invite expiry shows «۲۵ ساعت» — round down / use server-relative time.

## Out of scope
- 

## Acceptance
- Each item fixed (tests for B-3 flow and B-4 formatting); light + dark screenshots of touched screens in `docs/qa/bloom/B-N4-10b/`
- `verify` green
