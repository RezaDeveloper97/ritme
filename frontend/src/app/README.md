# `app/` — initialization layer (+ Next.js App Router)

Top FSD layer. Wires the application together: providers, global styles, i18n
and TanStack Query setup, and the localized route tree. In this project it
doubles as the **Next.js App Router** directory.

- `globals.css` — Tailwind entry + base styles.
- `providers.tsx` — client providers (TanStack Query).
- `[locale]/layout.tsx` — root layout: `<html lang dir>`, i18n + query providers.
- `[locale]/page.tsx` — `/` screen.

**Import rule:** may use every lower layer (`views`, `widgets`, `features`,
`entities`, `shared`). Nothing imports from `app`.

Locale lives in the URL (`/fa`, `/en`); the redirect to the default locale is
handled by `src/middleware.ts`.
