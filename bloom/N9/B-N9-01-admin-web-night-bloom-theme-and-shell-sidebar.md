---
id: B-N9-01
title: admin-web Night & Bloom theme and shell (sidebar, ⌘K, light/dark)
milestone: N9
type: frontend
status: todo
depends_on: [B-N8-10]
parallel_group: N9-A
touches: [admin-web/src]
skills: [verify-all]
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# B-N9-01 — admin-web Night & Bloom theme and shell (sidebar, ⌘K, light/dark)

## Why
Admin artboards are 1440px Night & Bloom.

## Design
- `docs/design/night-bloom/h-admin/nbl_Admin_Overview.dc.html` (+ `nbd_Admin_Overview`)

## Scope
- Tokens light/dark, sidebar groups per design (+ «سایر ماژول‌ها»), header with ⌘K command/search palette, user chip.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light + dark screenshots vs artboards
- `verify` green
