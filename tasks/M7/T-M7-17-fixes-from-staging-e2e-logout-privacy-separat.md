---
id: T-M7-17
title: Fixes from staging e2e — logout privacy, separators, NT prefill, daily message after mode switch
milestone: M7
type: frontend
status: todo
depends_on: [T-M7-15,T-M4-10]
parallel_group: M7-E
touches: [frontend/src/features/auth,frontend/src/entities/user,frontend/src/shared/session,frontend/src/screens/pregnancy-calendar,frontend/src/screens/pregnancy-log,frontend/src/screens/home,frontend/src/entities/message,frontend/src/widgets/checkups-card,frontend/messages,backend-go/resources/translations,docs/pregnancy-v2,docs/checkups]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go test ./resources/...
---

# T-M7-17 — Fixes from staging e2e — logout privacy, separators, NT prefill, daily message after mode switch

## Why
Staging UI e2e (2026-09-28) for T-M7-15 and T-M4-10 found a privacy bug and a few display bugs (details in
docs/pregnancy-v2/README.md § Staging UI e2e and docs/checkups/README.md § Staging UI e2e).

## Scope
1. **Privacy:** the persisted `ritme-onboarding` store (name, mobile, birth date, weight, height, intention, pregnancy
   basis) survives logout. Every session end (explicit logout, account deletion, `endsSession()` path) must reset
   and remove it — and any other persisted per-user store (audit `persist(` usages; keep device prefs like theme and
   locale). Don't use `localStorage.clear()` (CLAUDE.md §11.1). Unit test.
2. **Separator:** the fa middle dot `·` in counts lines reads like a Persian zero («۳ · ۱» → «۳۰ ۱۰») — replace the
   separator in fa messages (checkups.json separator and the other `·` uses in fa copy where it sits next to digits,
   incl. the pregnancy PDF «· ۴ لیوان») with a form that can't be confused (e.g. «،» or a spaced «|» / bullet «•» —
   choose one and use it consistently). Keep en unchanged. Sync the backend translation seed (a Go test enforces it).
3. **NT prefill category (9a):** booking a care-plan item prefills the appointment kind from the care item's kind
   (scan/test/vaccine/visit → the matching appointment category) instead of always `in_person` (`bookHref`). Unit test.
4. **Daily message after switching back to cycle (9h):** `/messages/daily` returns 400 for a cycle user with no
   period data; the home must handle that as an empty/“log your period” state, not a broken card full of «—».
   Frontend only (backend behaviour is Laravel-parity).
5. **Log weight field** shows «۶۲.۵» with ASCII dot — use the locale decimal separator like «آخرین ثبت».

## Out of scope
- 9b/9c/9d/9i-content/9j, checkups minor design items — design/clinical decisions.

## Acceptance
- 1–5 fixed with unit tests; verify green; after logout localStorage contains no `ritme-onboarding` (checked in a
  local headless run).
