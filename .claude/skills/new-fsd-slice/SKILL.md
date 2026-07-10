---
name: new-fsd-slice
description: Scaffold a new frontend FSD slice (entity/feature/widget/screen) with correct layer placement, public API barrel, TanStack Query, and fa/en i18n. Use when adding any new frontend functionality.
---

# New FSD slice (frontend)

Stack: Next.js 15 App Router, React 19, TS, Tailwind 4, TanStack Query, Zustand, react-hook-form+zod, next-intl (fa/en), Jalali dates (dayjs+jalaliday).

## Pick the layer first
- **entities/** — a domain concept: API calls, types, query hooks, dumb display components (e.g. `entities/pregnancy`).
- **features/** — a user interaction that changes something (forms, toggles, mutations).
- **widgets/** — composition of entities+features into a self-contained block (e.g. `banner-slideshow`).
- **screens/** — full pages, composed of widgets; wired into `app/` routes.
- **shared/** — pure utilities/UI primitives, zero domain knowledge.

## Slice structure
```
src/<layer>/<slice>/
  api/       # axios calls + TanStack Query hooks (entities)
  model/     # types, zod schemas, zustand store if truly client state
  ui/        # components
  index.ts   # PUBLIC API — everything cross-slice goes through here
```

## Hard rules
- Imports only flow downward: app → screens → widgets → features → entities → shared. Same-layer slices never import each other.
- Cross-slice imports ONLY via `index.ts` barrels — no deep paths.
- Server state lives in TanStack Query (proper query keys, invalidate after mutations) — never duplicated into Zustand.
- `'use client'` only where actually needed (hooks, events).
- **Every user-facing string** goes in `messages/fa.json` AND `messages/en.json` via next-intl — no hardcoded text.
- RTL-safe Tailwind: logical utilities (`ms-`, `me-`, `ps-`, `pe-`, `start-`, `end-`) wherever direction matters.
- Dates shown to users: Jalali via the existing dayjs+jalaliday helpers in shared.

## Before finishing
```bash
cd frontend && npm run typecheck && npm run lint && npm run fsd:lint
```
`fsd:lint` (steiger) enforces the layer rules — it must pass. If the app has mode-aware UI concerns (period vs pregnancy mode), check whether the bottom nav / screen needs mode handling like the existing pregnancy screens.
