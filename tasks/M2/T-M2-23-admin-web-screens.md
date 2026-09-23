---
id: T-M2-23
title: admin-web — all admin screens (parity with the Blade panel)
milestone: M2
type: frontend
status: done
depends_on: [T-M2-21, T-M2-22]
parallel_group: M2-G
touches: [admin-web/src, admin-web/messages]
skills: [frontend-design:frontend-design]
verify: cd admin-web && npm run typecheck && npm run lint && npm run test && npm run build
---

# T-M2-23 — admin-web: all admin screens

## Why
Every page the Blade panel offers today must exist in admin-web before the admin host is switched (infra inventory
§3 lists them). Editors must not lose any capability.

## Scope
Screens (list + create/edit + toggle/delete where applicable), all against the Go admin API:
users (list/search/filter, detail with stats, edit, block/unblock, delete), articles (rich text + cover upload +
phases), affirmations, challenges (+ completions report), recommendations (subphase picker), banners (upload,
schedule window, position, link type), task templates, info sections (by group), pregnancy weeks (10 JSON sections
per week), phase contents (9 sections), smart messages (group/locale filter, payload editor, approve, toggle),
languages (super: CRUD, default, toggle, regenerate), translation editor (per namespace), admins (super), own
password. Role-aware navigation; empty/error/loading states; Persian copy.

## Out of scope
New admin features beyond Blade parity (note ideas in PROGRESS instead).

## Acceptance
- A parity checklist in PROGRESS: every Blade route in `backend/routes/admin.php` ↔ an admin-web screen, all ✔.
- Headless screenshot pass of each screen (see memory "Headless UI verification") in RTL light + dark.
- `verify:` green.
