# admin-web — the Next.js admin (`admin-web/`)

Replaces the Blade panel (`backend/resources/views/admin`) on the admin host (`adpanell.ritme.app`, stage: the
stage host). It is a separate Next.js app so the user PWA bundle stays untouched, and it talks only to the Go admin
API ([admin-api.md](admin-api.md)) plus the public `GET /api/v1/languages`. T-M2-22 built the scaffold (shell,
auth, shared UI, dashboard); T-M2-23 adds the screens.

## 1. Stack

Next.js 15 (same major as `frontend/`), App Router, React 19, TypeScript strict (+ `noUncheckedIndexedAccess`),
Tailwind 4, TanStack Query, Zustand (UI state only), zod at the API boundary, next-intl, TipTap 3, vitest, eslint
(`next/core-web-vitals` + `next/typescript`), steiger (FSD).

```bash
cd admin-web
npm run dev          # http://localhost:3001 (proxies /api/* to ADMIN_API_PROXY_TARGET)
npm run typecheck && npm run lint && npm run test && npm run build   # the task's verify
npm run fsd:lint     # FSD boundaries
```

## 2. Runtime topology

```
browser ──► adpanell.ritme.app (nginx, T-M2-25)
              ├── /api/admin/  → backend-go   (session cookie + CSRF, admin-api.md)
              ├── /api/v1/languages → backend-go (public content-language registry)
              └── /            → admin-web:3000 (Next standalone server)
```

- **Base path (build time):** `NEXT_PUBLIC_ADMIN_BASE_PATH` (Docker build arg, compose var `ADMIN_WEB_BASE_PATH`),
  default empty = served at `/` (production's own host). Staging shares `stage.ritmeapp.ir` with the user app and
  the Blade panel, so it builds with `/panel` (`docker-compose.stage.yml`); nginx sends `/panel` and `/panel/…` to
  admin-web. Next's `Link`/router/middleware handle the prefix; the raw `window.location` redirects use
  `withBasePath`/`stripBasePath` (`shared/config/base-path.ts`). The API bases are **not** prefixed.
- **Middleware redirects** are built on the public origin (`X-Forwarded-Host`/`Host` + `X-Forwarded-Proto`,
  `shared/lib/public-origin.ts`): behind nginx the standalone server only knows `localhost:3000`.
- **One origin.** The session cookie is `__Host-ritme_admin_session` (host-only, `Secure`), so the admin UI and the
  admin API must share the host. Nothing is cross-origin in production and no CORS is involved.
- **Env (build time, public):** `NEXT_PUBLIC_ADMIN_API_BASE_URL` (default `/api/admin/v1`) and
  `NEXT_PUBLIC_API_BASE_URL` (default `/api/v1`, only for `/languages`). Absolute URLs work too; their origins are
  added to the CSP `connect-src`.
- **Local dev:** run backend-go with `APP_ENV=local ADMIN_COOKIE_SECURE=false` (plain http: cookies lose the
  `__Host-` prefix, both names are accepted), then `ADMIN_API_PROXY_TARGET=http://127.0.0.1:8020 npm run dev`.
  `next dev` rewrites `/api/*` to backend-go, so the browser stays same-origin. You need an `admins` row (bcrypt
  hash) and the `languages` rows.
- **Docker:** `admin-web/Dockerfile` (standalone output, uid 1001, port 3000, healthcheck on `/login`). Compose
  service `admin-web` is internal only (`expose`, no published port). Stage alias `stage-admin-web`, container
  `ritme-stage-admin-web-1`. nginx wiring and the prod container-name pin are T-M2-25.
- **Headers:** enforced CSP (same-origin + `img-src https:` for uploaded images), `X-Frame-Options: DENY`,
  `X-Robots-Tag: noindex`. `middleware.ts` redirects to `/login` when there is no session cookie at all; the API
  decides whether a present cookie is valid.

## 3. Structure (FSD-lite)

```
src/
  app/          routes only: layout (font, theme bootstrap, intl), providers (QueryClient, 401 handler, toasts,
                confirm), login/, (panel)/ (AuthGate + shell) with one folder per screen
  screens/      one slice per route: ui/<Name>Screen.tsx, api/ (query hooks + zod schemas), index.ts
  widgets/shell sidebar (role-aware NAV), header
  features/auth login form, logout, useMe/useCurrentAdmin, AuthGate, safeNext
  entities/admin Admin schema/type, RoleBadge
  shared/
    api/        fetch client, envelopes, ApiError, CSRF, QueryClient
    ui/         Button, Panel, Field/TextInput/TextArea/Select/Switch, DataTable, Pagination, SearchInput,
                toast, confirm, ImageUpload, RichTextEditor, TranslatableField, Badge, Icon, ErrorState, …
    lib/        date (Jalali/Gregorian), number, list params (URL state)
    i18n/       UI locales, content languages, error-code translation
    theme/      light/dark store + no-flash script
    config/     env, cookie names
```

Import rules are `frontend/CLAUDE.md` §3: downward only, no sibling slices, only through `index.ts`.

## 4. Conventions (from `frontend/CLAUDE.md`, adapted)

- **Colours:** tokens only (`src/app/globals.css`, both themes). Tailwind utilities map to them (`bg-surface`,
  `text-ink-3`, `border-line`, `text-danger-deep`, …). eslint fails on hex literals and static `style={{}}` in
  `src/`. The brand gradient is only on the brand mark and the login CTA; everyday primary = `variant="primary"`
  (solid `--brand-fill`); turquoise only for data; red only for danger.
- **Logical CSS only** (`ms-`, `pe-`, `start-`, `border-s`, `text-start`). Icons that point somewhere flip with
  `rtl:-scale-x-100`.
- **Two kinds of language.**
  - *Admin UI language* (the chrome): bundled `messages/<code>.json`, picked by the `ritme_admin_locale` cookie, no
    URL prefix. Direction comes from the bundle's `meta.direction`. Adding one = a new JSON (a test asserts key
    parity with `fa.json`) + one line in `shared/i18n/ui-locales.ts`.
  - *Content languages*: **data**, from `GET /api/v1/languages` (`useContentLanguages()`). Never list them in code.
- **Every visible string** goes through next-intl (`useTranslations('<namespace>')`), with ICU plurals. The API's
  English `message` is never shown: errors go through `useErrorMessage()` → `errors.<error_code>`.
- **Dates/numbers** only through `shared/lib` (`formatDateTime`, `formatDate`, `formatNumber`): `fa` → Jalali +
  Persian digits, anything else → Gregorian, always in Asia/Tehran.
- **Server state** in TanStack Query with a key factory per screen/entity; invalidate through it after mutations.
  Zustand only for UI state (toasts, confirm, theme).
- **Theme:** `<html data-theme>`; `ThemeToggle` in the header. No component branches on the theme.
- **Sign-out** is a full document replace (`/login?signed_out=1`); a JSON 401 anywhere does the same to
  `/login?next=…`. Never call `router.back()` for navigation.

## 5. The API client (`shared/api`)

```ts
api.get('/users', { query: list.query, schema: usersListSchema, signal })   // → validated `data`
api.post('/admins', body, { schema: z.object({ admin: adminSchema }) })     // JSON + X-CSRF-Token
api.post('/banners', formData)                                               // multipart, no Content-Type
api.get('/languages', { api: 'public', schema })                             // /api/v1
```

- `credentials: 'include'` always. `X-CSRF-Token` on POST/PUT/PATCH/DELETE from the last login / `GET /auth/me`
  (fallback: the readable CSRF cookie). A 419 refreshes the token via `/auth/me` and retries once.
- Every failure is an `ApiError { status, code, fieldErrors, retryAfter }` (`network_error`, `invalid_response`,
  `http_error` are client codes). `error.field('title.fa')` reads the 422 bag.
- Only a **JSON** 401 triggers the unauthorized handler (a proxy / Basic-auth 401 has no JSON body).
- `listSchema(item, filters)` for `{items, meta, filters}`; `optionSchema` for `{value, label}`.

## 6. Shared UI you will use in every screen

| Need | Use |
|---|---|
| Section container | `<Panel title actions bodyClassName="">` (`""` for edge-to-edge tables) |
| Server list | `useListParams({status:'all'})` → `useQuery({queryKey: keys.list(list.query), placeholderData: keepPreviousData})` → `<SearchInput>` + `<Select>` + `<DataTable>` + `<Pagination>` (see `screens/ui-kit`) |
| Inputs | `<TextInput>`, `<TextArea>`, `<Select options>`, `<Switch>`, all with `label`/`hint`/`error` |
| Translatable column | `<TranslatableField name="title" label value onChange required kind="text\|textarea\|rich" errors={error?.fieldErrors}>` — one input per active content language, only the default `required`, errors read as `title.<code>` |
| Rich text | `kind="rich"` or `<RichTextEditor>` (lazy-loaded TipTap). Its schema only emits tags/attributes the backend sanitizer keeps; `rich-text.test.ts` enforces it. Empty document = `''`. |
| Image | `<ImageUpload label value={file} currentUrl={row.image_url} onChange maxMb={4}>` → `fd.append('image', file)` |
| Feedback | `toast.success(t('saved'))`, `toast.error(describe(err))`; `if (await confirm({message, tone:'danger'}))` |
| Errors | `useErrorMessage()(error)`; `<ErrorState error onRetry>` |

`/ui-kit` (dev builds only in the sidebar) renders all of them live, including a real server list on `GET /users`.

## 7. How to add a CRUD screen (T-M2-23)

Example: articles.

1. **API + schemas** — `src/screens/articles/api/articles.ts`:
   ```ts
   export const articleSchema = z.object({ id: z.number(), title: z.record(z.string(), z.string()), … });
   const listSchemaA = listSchema(articleSchema, z.object({ q: z.string(), status: z.string() }).partial());
   export const articleKeys = {
     all: ['articles'] as const,
     list: (q: Record<string, string | number>) => [...articleKeys.all, 'list', q] as const,
     detail: (id: number) => [...articleKeys.all, 'detail', id] as const,
   };
   export const useArticles = (q) => useQuery({ queryKey: articleKeys.list(q), queryFn: ({signal}) =>
     api.get('/articles', { query: q, schema: listSchemaA, signal }), placeholderData: keepPreviousData });
   export const useSaveArticle = () => { const qc = useQueryClient(); return useMutation({ mutationFn: …,
     onSuccess: () => qc.invalidateQueries({ queryKey: articleKeys.all }) }); };
   ```
   If two screens need the same schema/hooks, move them down to `entities/article` instead.
2. **List screen** — `ui/ArticlesScreen.tsx`: `useListParams`, `<Panel actions={<Link className="btn btn-primary">}>`,
   `<SearchInput>`, filters, `<DataTable onRowClick>`, `<Pagination>`. Delete: `confirm({tone:'danger'})` →
   mutation → `toast.success`.
3. **Form screen** — `ui/ArticleFormScreen.tsx`: controlled `useState` per field (seed from the detail query), zod
   check on submit, `TranslatableField` for translatable columns, `ImageUpload` for files (send `FormData` with
   `title[fa]`-style keys when a file is involved, JSON otherwise). On 422 pass `error.fieldErrors` to fields
   (`error.field('slug')`); show `useErrorMessage()` for everything else. Super-only screens: check
   `useCurrentAdmin()?.role` and still expect a 403 from the API.
4. **Routes** — `src/app/(panel)/articles/page.tsx` (list), `articles/new/page.tsx`, `articles/[id]/page.tsx`:
   each is a server component that exports `generateMetadata` (title via `getTranslations`) and renders the
   screen. Screens reading `useSearchParams` are already inside the panel layout's `<Suspense>`.
5. **Nav** — set `ready: true` on the item in `src/widgets/shell/model/nav.ts` (role gating is already there).
6. **Strings** — add a namespace (`articles`) to **every** `messages/*.json` (the parity test fails otherwise).
   New error codes go under `errors`.
7. **Tests** — pure logic (mappers, validators) as `*.test.ts`; components that matter can be rendered with
   `react-dom/server` inside `NextIntlClientProvider` (see `TranslatableField.test.tsx`).
8. Run `npm run typecheck && npm run lint && npm run test && npm run build && npm run fsd:lint`.
