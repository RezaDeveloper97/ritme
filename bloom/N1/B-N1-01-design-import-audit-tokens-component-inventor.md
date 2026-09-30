---
id: B-N1-01
title: Design import audit — tokens, component inventory, screen→route map
milestone: N1
type: investigate
status: todo
depends_on: []
parallel_group: N1-A
touches: [docs/night-bloom]
skills: []
verify: test -s docs/night-bloom/README.md
---

# B-N1-01 — Design import audit — tokens, component inventory, screen→route map

## Why
The Night & Bloom canvas (363 artboards, 10 pages) replaces the current look and adds ~75% new product surface. Every later task needs one shared reference for tokens, components and where each screen lives.

## Scope
- Read every artboard under `docs/design/night-bloom/` (TEXT.md per folder has the extracted copy).
- Extract the **light** (`nbl_`) and **dark** (`nbd_`) palettes into a token table: canvas, surface, border, text 1/2/3, primary (#6E54F0 light / #B9A6FF dark), data accent (#0FA3C9 / #4CE0C3), warm accent (#F5A623 / #FFB86B), rose (#B82A52 / #FF6B8B), success, danger; plus the dark starfield + radial glow background.
- Typography scale (Vazirmatn weights, Lalezar display numerals), radii (14/18/20/24/27), shadows, spacing.
- Component inventory (header w/ 44px round back button, date strip, pill chips, segmented tabs, stepper input, cards, status pills, accordion, bottom sheet, FAB, bottom nav, charts) with the artboards that use each.
- Screen → route table for all 181 screens: existing route to restyle, or NEW, with the owning bloom task id.
- Bottom-nav spec per mode (امروز · <mode tab> · + · خدمات · من; mode tab: تقویم / باروری / بارداری / کودک / علائم) and list the artboards whose nav deviates (e.g. «تحلیل» instead of «خدمات»).
- Design gaps list: empty `nb2_Preg_Timeline`, dark-only `nbd_User_Library`/`nbd_User_Player`, no menopause/teen home artboards, no doctor-side UI — each with a proposed default.
- If a later task's scope is wrong after this audit, edit that task file and note it in PROGRESS.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/night-bloom/README.md` with the sections above; every later N1 task can cite it
- `verify` green
