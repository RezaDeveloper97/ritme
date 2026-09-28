---
id: T-M2-32
title: Small web fixes from staging smoke — reminder form buttons, slide a11y digits
milestone: M2
type: frontend
status: todo
depends_on: [T-M2-25]
parallel_group: M2-J
touches: [frontend/src/app/globals.css,frontend/src/screens/profile-reminders,frontend/messages,frontend/src/widgets/intro-carousel,frontend/src/widgets/banner-slideshow,backend-go/resources/translations,backend-go/internal/i18n/testdata]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go test ./... && make contract ROUTES=all
---

# T-M2-32 — Small web fixes from staging smoke — reminder form buttons, slide a11y digits

## Why
Staging smoke (docs/go-migration/stage-rollout-log.md § Staging smoke + soak) found two small frontend bugs.

## Scope
1. Old reminder form (`?sheet=reminders` → «یادآور جدید»): «بی‌خیال» overflows the card and «ذخیره یادآور» is
   squeezed — `.btn { width: 100% }` + `.rem-form-cancel { flex-shrink: 0 }` (globals.css), RemindersSheet. Make the
   two buttons share the row properly (scoped class change; don't change global `.btn` behaviour elsewhere).
2. fa screen-reader text «اسلاید 1 از 3» / goToSlide use Latin digits (`messages/fa/welcome.json`, IntroCarousel,
   BannerSlideshow) → ICU `{n, number}` or pass formatted numbers; keep en working. Sync the backend translation
   seed and i18n goldens (tests enforce it).

## Acceptance
- Both fixed; verify green (incl. contract).
