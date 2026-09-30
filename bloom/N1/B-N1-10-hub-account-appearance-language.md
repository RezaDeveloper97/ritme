---
id: B-N1-10
title: «من» hub, account, appearance, language
milestone: N1
type: frontend
status: todo
depends_on: [B-N1-04]
parallel_group: N1-J
touches: [frontend/src/screens/profile,frontend/src/screens/profile-info,frontend/src/screens/appearance,frontend/src/shared/theme]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-10 — «من» hub, account, appearance, language

## Why
The Me tab is restructured into grouped sections.

## Design
- `docs/design/night-bloom/g-me-settings/nbl_Me_Hub.dc.html` (+ `nbd_Me_Hub`)
- `docs/design/night-bloom/g-me-settings/nbl_Me_Profile.dc.html` (+ `nbd_Me_Profile`)
- `docs/design/night-bloom/g-me-settings/nbl_Me_Appearance.dc.html` (+ `nbd_Me_Appearance`)

## Scope
- Me hub groups: profile header (masked phone, mode, Plus status), من و خانواده, کارها و خریدها, داده و دستگاه, تنظیمات, پشتیبانی, خروج. Entries whose feature ships later show «به‌زودی» until their task lands.
- Account screen (name, family name, phone change, birth date, email, devices, logout, delete account).
- Appearance: theme dark/light/system, text size (root font scale), reduce motion (overrides OS), keep haptics row hidden on web.
- Language & calendar (fa/en, Jalali).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
