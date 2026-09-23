---
id: T-M2-18
title: Home page sections, task/challenge toggles and notifications
milestone: M2
type: backend
status: todo
depends_on: [T-M2-10, T-M2-15, T-M2-16, T-M2-17]
parallel_group: M2-H
touches: [backend-go/internal/home, backend-go/internal/http/routes_home.go, backend-go/db/queries/home, backend-go/contract/allowlist/home.yaml, backend-go/contract/allowlist/notifications.yaml]
skills: []
verify: cd backend-go && go test ./internal/home/... && make test-int PKG=./internal/home/... && make contract ROUTES=home,notifications
---

# T-M2-18 — Home page sections, toggles and notifications

## Why
Home aggregates almost every domain (cycle, pregnancy, messages, articles, reminders, logs). It is the last API group
to port. Read api-inventory §1.10, domain-inventory §4.4, and `app/Services/HomePage/**`, `DailyChallengeService`.

## Scope
1. `home/context` = `HomeContext` (per-request memo, log window date−29 → end of Saturday-start week, `num()` Persian
   digits with PHP float strings, `t()` fa-or-English).
2. The 17 sections with their order and cycle-only flags; each section isolated (panic/error → logged and omitted,
   like the PHP try/catch); `HomeSection` JSON with `meta: null` when empty.
3. Support: `DailyChallengeService` (cycle-day filter, exclude last-14-day completions, signal-category preference,
   `crc32("{uid}:{Y-m-d}") % count`), affirmation `dayOfYear % count`, `CycleHistoryDigest`, `HealthMetricScorer`,
   `MoonPhase` (fmod/cos/round 2), `CyclePhasePalette`.
4. Endpoints: `GET /home` (framework 422 on `?date`), `GET /home/sections/{section}` (404 with `available_sections`),
   `POST /home/tasks/{task}/toggle` and `/home/challenges/{challenge}/toggle` (framework model-not-found 404 text
   `No query results for model [App\\Models\\TaskTemplate] 99` — keep the PHP class name), notifications list
   (per_page clamp 1–100, `pagination`, `+03:30`), read-all, read one (other user's → 404).

## Out of scope
New home sections or UX changes.

## Acceptance
- `make contract ROUTES=home,notifications` green for all personas × fa/en/ar, including toggle sequences
  (toggle → GET section → toggle back).
- A test proves a failing section is omitted without failing the request.
