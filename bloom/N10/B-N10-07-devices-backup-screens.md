---
id: B-N10-07
title: Devices & backup screens
milestone: N10
type: frontend
status: todo
depends_on: [B-N10-01]
parallel_group: N10-G
touches: [frontend/src/screens/devices,frontend/src/screens/backup]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N10-07 — Devices & backup screens

## Why
«ساعت و گجت‌ها» and «پشتیبان و خروجی داده».

## Design
- `docs/design/night-bloom/g-me-settings/nbl_Me_Hub.dc.html` (+ `nbd_Me_Hub`)

## Scope
- Backup: last server sync time, manual export. Devices: honest «به‌زودی» state + interest capture (web can't pair most wearables); document what a real integration would need.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
