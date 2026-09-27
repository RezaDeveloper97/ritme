---
id: T-M4-11
title: Checkups UI polish from local e2e
milestone: M4
type: frontend
status: todo
depends_on: [T-M4-07,T-M4-08,T-M4-09]
parallel_group: M4-D
touches: [frontend/src/screens/checkups,frontend/src/screens/checkup-detail,frontend/src/screens/checkup-history,frontend/src/screens/checkup-self-exam,frontend/src/screens/pregnancy-calendar,frontend/src/shared/lib/pdf,backend-go/internal/checkups,docs/checkups]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go test ./internal/checkups/...
---

# T-M4-11 — Checkups UI polish from local e2e

## Why
The local e2e for T-M4-10 (docs/checkups/README.md § Local e2e) found bugs on the checkup screens.

## Scope
1. PDF footer prints the raw key `checkups.history.pdf.footer` (IntlError): `CheckupHistoryPage.tsx` calls
   `t('history.pdf.footer')` without values while `shared/lib/pdf/render.ts` fills `{page}`/`{pages}` itself — pass the
   raw template (`t.raw`). Same pattern in `PregnancyCalendarPage.tsx`. Page numbers in render.ts must use locale
   digits (fa → Persian).
2. Checkup cards have no inner padding (global `.card` has none): add padding in CheckupsPage, CheckupDetailPage,
   CheckupHistoryPage, SelfExamPage cards (don't change the global `.card`), same approach as T-M5-10 (`p-4`).
3. Latin digits in fa copy: history «2 مورد ثبت‌شده» and self-exam «19 روز دیگر» → `formatNumber`.
4. Section order: the overdue/«نیاز به اقدام» section must come right after the first section as the design
   specifies (check docs/checkups/README.md and the design refs) — make the order deterministic. Prefer fixing it on
   the server (engine groups/sorts sections) if the API contract documents an order; otherwise order in the frontend
   view model. Add a test either way.

## Out of scope
- Minor design questions (attachment chip placement, detail hero icon, admin-web digit mixing, self-exam «موعدش رسیده»
  wording).

## Acceptance
- Items 1–4 fixed with unit tests for logic changes; verify green; light/dark screenshots of list, detail, history
  and self-exam re-taken locally into `docs/checkups/screenshots/` (same names) plus page 1 of the PDF.
