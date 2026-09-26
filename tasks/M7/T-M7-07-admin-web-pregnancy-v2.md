---
id: T-M7-07
title: admin-web — pregnancy week details editor, care plan, alert rules, message create
milestone: M7
type: frontend
status: todo
depends_on: [T-M7-06, T-M2-23]
parallel_group: M7-C
touches: [admin-web/src/widgets/shell/model/nav.ts, admin-web/messages, admin-web/src/shared/ui/Icon.tsx, admin-web/src/screens/pregnancy-weeks, admin-web/src/screens/pregnancy-care-plan, admin-web/src/screens/pregnancy-alert-rules, admin-web/src/screens/messages, admin-web/src/app/(panel)/pregnancy-care-plan, admin-web/src/app/(panel)/pregnancy-alert-rules, admin-web/src/widgets/shell/ui/Sidebar.tsx, admin-web/src/shared/i18n]
skills: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M7-07 — admin-web — pregnancy v2 admin screens

## Scope
1. `pregnancy-weeks`: new «جزئیات ساختاریافته» tab — size label, illustration picker (preview), length/weight/heart
   rate, headline, repeatable highlights (icon, title, body), body symptom chips + text, tasks (key auto, text),
   warning, reviewer name/date, sources; fa/en tabs; preview card mimicking the app hero.
2. New `pregnancy-care-plan` screen (list + reorder + form).
3. New `pregnancy-alert-rules` screen: per rule enabled, level (4 with the app's colors), typed params, texts per
   locale, a sample rendering of the alert card.
4. `messages`: «ایجاد» for missing keys of registered groups (week tips 1–42 grid shows filled/missing).
5. Sidebar group «بارداری».

## Acceptance
- An editor can change a week's size/highlights, a care item window, a rule threshold and a week tip, and see them
  in the app without deploy; build green.
