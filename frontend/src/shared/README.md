# `shared/` — domain-agnostic foundation

Reusable, domain-free building blocks. Segment-organized (no slices):

- `ui/` — UI kit primitives (e.g. `Button`). RTL-safe, i18n'd by callers.
- `lib/date/` — the **only** place that touches the date library; exposes
  locale-aware calendar helpers (`formatLongDate`, `toParts`, `monthMatrix`,
  `addDays`, `diffInDays`). The calendar follows the locale: Jalali for `fa`,
  Gregorian for `en`.
- `api/` — the single shared HTTP client (`apiClient`).
- `config/` — validated environment/config (`env`).
- `i18n/` — next-intl routing, navigation, request config, message loading.

**Import rule:** `shared` may import **nothing else in `src`** — it is the
fully independent base every other layer builds on (CLAUDE.md §3.1).

Import per segment: `@/shared/ui`, `@/shared/lib/date`, `@/shared/api`,
`@/shared/config`, `@/shared/i18n`.
