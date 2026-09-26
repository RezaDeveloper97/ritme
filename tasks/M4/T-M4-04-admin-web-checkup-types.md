---
id: T-M4-04
title: admin-web — checkup types screens (list, form, stats)
milestone: M4
type: frontend
status: in_progress
depends_on: [T-M4-03, T-M2-23]
parallel_group: M4-C
touches: [admin-web/src/widgets/shell/model/nav.ts, admin-web/messages, admin-web/src/shared/ui/Icon.tsx, admin-web/src/screens/checkup-types, admin-web/src/app/(panel)/checkup-types, admin-web/src/widgets/shell/ui/Sidebar.tsx, admin-web/src/shared/i18n]
skills: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M4-04 — admin-web — checkup types screens (list, form, stats)

## Why
Admin UI for T-M4-03, built with the shared admin kit from T-M2-23 (`resource.ts`, `FormPage`, `RowActions`,
`LinkTabs`, `translations` helpers).

## Scope
1. Sidebar item «چکاپ‌های دوره‌ای» (content group).
2. List: drag/arrow reorder, title (fa), category, interval, age window, active toggle, records count from `stats`,
   edit/delete row actions (delete disabled when in use → suggest deactivate).
3. Form: per-language tabs for title/subtitle/why/source note; repeatable editors for prep steps, guide steps
   (title+body) and finding options; category, performed_by, icon picker (same names as the app), tone swatches,
   interval (+ optional max), age min/max, cycle day range, remind lead, hide-in-pregnancy, active.
4. A read-only preview card that mimics the app row (icon tile + title + interval + timing) for both tones.
5. Stats panel on the list: users with records, records last 30 days, overdue users per type.

## Acceptance
- Create → edit → reorder → deactivate works against the Go admin API locally; build green; fa RTL correct.
