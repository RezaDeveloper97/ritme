# Ritme (ریتمی)

A women's health app — menstrual **cycle** tracking and **pregnancy** mode.
Persian-first (RTL, Jalali calendar), with English as a secondary locale.

> Architecture, conventions, and non-negotiable rules live in
> [`CLAUDE.md`](./CLAUDE.md). Read it before contributing. This README only
> covers running the project.

## Stack

Next.js (App Router + RSC) · TypeScript (strict) · Tailwind CSS v4 (logical
utilities) · next-intl · TanStack Query · Zustand · react-hook-form + zod ·
Jalali dates via a centralized date layer.

## Getting started

```bash
npm install
cp .env.example .env.local   # then fill in values
npm run dev                  # http://localhost:3000  → redirects to /fa
```

## Scripts

| Script             | What it does                                  |
| ------------------ | --------------------------------------------- |
| `npm run dev`      | Start the dev server                          |
| `npm run build`    | Production build                              |
| `npm run start`    | Run the production build                      |
| `npm run lint`     | ESLint (next/core-web-vitals + TS)            |
| `npm run typecheck`| `tsc --noEmit`                                |
| `npm run test`     | Unit tests (domain logic, date layer)         |
| `npm run fsd:lint` | Feature-Sliced Design boundary check (steiger)|

**Definition of done:** `typecheck`, `lint`, and `fsd:lint` pass, and new
domain logic has tests.

## Project structure (Feature-Sliced Design)

Layers, highest to lowest. A layer may import **only from layers below it**, and
cross-slice imports always go through a slice's `index.ts` (CLAUDE.md §3).

| Layer        | Role                                                | May import                 |
| ------------ | --------------------------------------------------- | -------------------------- |
| `app/`       | init + Next.js App Router (providers, i18n, routes) | everything below           |
| `views/` \*  | page-composition (one screen per slice)             | widgets, features, …, shared |
| `widgets/` \*| large self-contained blocks                         | features, entities, shared |
| `features/` \*| user actions (log-period, switch-mode, …)          | entities, shared           |
| `entities/` \*| domain nouns (cycle, symptom, pregnancy, user, …)  | shared                     |
| `shared/`    | domain-agnostic foundation                          | nothing else in `src`      |

\* Created on demand — only `app/` and `shared/` exist today (FSD discourages
empty layers, and `fsd:lint` enforces it). The **`fsd-slice` skill** scaffolds a
new slice and its layer with the right segments and public API.

> `views/` is the FSD "pages" layer, renamed to avoid colliding with the
> Next.js Pages Router (a `src/pages/` directory would be auto-routed).

```
src/
├── app/      # App Router: providers.tsx, globals.css, [locale]/{layout,page}.tsx
├── shared/   # ui/ · lib/date/ · api/ · config/ · i18n/
└── middleware.ts   # next-intl locale routing
messages/     # i18n messages — messages/<locale>/<namespace>.json
```
