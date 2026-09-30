# Content catalog (CB-CORE-03)

One admin-editable table for the small clinical/content lists of the canvas-build epics — warning signs, missed-pill
rules, FAQs, kit items, score items, layette templates, tips — instead of one table per list (DECISIONS #7). Go only
(deviation D-31). Admin UI: CB-CORE-04.

## 1. Which mechanism?

| You need… | Use | Why |
|---|---|---|
| Something the **user logs** per day (a symptom, bleeding level, trigger, measurement) — even when the list is mode-scoped (menopause symptoms, pregnancy signs) | **bloom log taxonomy v2** (B-N3-01, `internal/healthlog`) | It owns the logging shape, enums, backfill and analysis. Never mirror log items into the catalog (CB-MENO-01 §1). |
| A **per-user record** tied to a list entry (records, settings, due dates) | the domain's own table (e.g. `checkup_types` for checkups, M4) | The catalog has no user data and no FKs; delete is hard. |
| **Engine copy**: smart/daily messages, BMI texts, one keyed text blob selected by rules, per-locale payload objects | **`message_contents`** (`messages/content.Repository`, admin «Messages») | Rows are per locale with a free payload and approval flag; the message engine reads them by `group`/`item_key`. |
| **UI strings** (button labels, headings, empty states) | `frontend/messages/<code>/*.json` (seed copy `backend-go/resources/translations`) | Not content; no admin workflow per item. |
| An **ordered list of display items** a screen renders — title (+ body) per item, optional per-item data, optional mode/audience filter, clinical review flag | **`catalog_items`** (this page) | Free-form groups, no schema change per list, one admin editor. |

Rule of thumb: if the server makes decisions from a row (rules, schedules, scores computed server-side), the fields the
logic reads belong in the domain's table or in `meta` with a documented shape per group (§4); if a user writes against
it, it is not a catalog item.

## 2. Schema — `catalog_items` (goose `00009_catalog_items.sql`, Laravel twin `2026_09_30_000001_…`)

| Column | Meaning |
|---|---|
| `group` varchar(64) | the list, lowercase snake case (`teen_faq`, `missed_pill_rules`, `meno_alerts`) |
| `code` varchar(64) | stable item id inside the group (unique per group); clients and seeds key on it; fixed after create |
| `sort_order` int | display order (then `id`) |
| `is_active` bool | inactive items are hidden from the public API |
| `audiences` JSON list \| NULL | mode/audience codes the item is for (`menopause`, `teen`, `pregnancy`, `ttc`…); NULL = everyone. Codes are free lowercase snake case — not validated against a mode enum |
| `title` JSON (required) | `{"fa": …, "en": …}` — keys grow with `languages`; only the default language is required |
| `body` JSON \| NULL | same, longer text (≤ 5000 chars per language) |
| `meta` JSON \| NULL | free-form per group (≤ 16 KB), see §4 |
| `needs_review` bool (default true) | copy is `[needs clinical review]` (DECISIONS #7); admins clear it after review; clients may show a badge |

## 3. API

**Public** — `GET /api/v1/catalog/{group}[?audience=<code>]` (auth:api, `Accept-Language` / `?locale=`; OpenAPI tag
`Catalog`):

```json
{"success": true, "data": {"group": "teen_faq", "locale": "fa", "items": [
  {"code": "first_period", "title": "…", "body": "…", "meta": null, "audiences": ["teen"], "needs_review": true}
]}}
```

- Active items only, `sort_order, id`. Unknown group → empty `items`; malformed group name → 404.
- `title`/`body` picked: request locale → default language → first non-empty translation; `null` when all empty.
- `meta`: every nested object whose keys all look like language codes and include at least one active language is
  replaced by its picked value; everything else is returned as stored.
- `?audience=x` returns the items with `audiences` NULL plus those listing `x`. Without it, all active items.
- Cached in Redis per group (`ritme-go:catalog.group.<group>`, 10 min TTL), flushed by every admin write.

**Admin** — `/api/admin/v1/catalog` (httpadmin session + CSRF, editor and super):

| Method + path | |
|---|---|
| `GET /catalog` | groups that have items: `{items: [{group, items_count, active_count}]}` |
| `GET /catalog/{group}?q=&status=all\|active\|inactive&page=&per_page=` | paginated items (`filters` echoed); `q` searches code + title |
| `POST /catalog/{group}` | create (201 `{catalog_item}`); `code` required + unique in group; `is_active`/`needs_review` default true, `sort_order` defaults to the end |
| `GET /catalog/{group}/{id}` | `{catalog_item}`; an id of another group is 404 |
| `PUT /catalog/{group}/{id}` | update; `code` ignored; absent optional fields keep their value, `null` clears |
| `DELETE /catalog/{group}/{id}` | hard delete (`{id}`) — prefer `is_active: false` for items clients may have referenced |

`catalog_item` = `id, group, code, sort_order, is_active, audiences, title, body, meta, needs_review, created_at,
updated_at` (JSON columns decoded, all languages). Validation errors are the admin 422 bag (`title.fa`, `code`,
`audiences.0`, …). Writes are audit-logged (`catalog_item.create|update|delete`).

## 4. `meta` conventions

- Keep it flat and document each group's shape in the epic's doc (e.g. `docs/canvas-build/menopause.md`), plus in the
  admin editor's per-group hints (CB-CORE-04).
- Translatable values inside meta are objects keyed by language code (`{"cta": {"fa": "تماس", "en": "Call"}}`) — the
  public API picks them; lists of them work too (`{"steps": [{"fa": …, "en": …}]}`).
- Numbers/enums the frontend branches on (`severity: "urgent"`, `score_max: 4`, `hours: 24`) stay plain values.

## 5. Seed convention (each epic appends)

Never edit `00009`. Each epic that ships a group adds **one** data migration pair in its own task:

1. goose `backend-go/db/migrations/000NN_catalog_<group>.sql` (next free number):

   ```sql
   -- +goose Up
   INSERT IGNORE INTO `catalog_items` (`group`, `code`, `sort_order`, `is_active`, `audiences`, `title`, `body`, `meta`, `needs_review`, `created_at`, `updated_at`) VALUES
     ('teen_faq', 'first_period', 1, 1, '["teen"]', '{"fa":"…","en":"…"}', '{"fa":"…","en":"…"}', NULL, 1,
      CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'), CONVERT_TZ(UTC_TIMESTAMP(), '+00:00', '+03:30'));
   -- +goose Down
   DELETE FROM `catalog_items` WHERE `group` = 'teen_faq' AND `code` IN ('first_period');
   ```

2. a data-only Laravel twin in `backend/database/migrations/` using `DB::table('catalog_items')->insertOrIgnore([...])`
   with the same rows (see `2026_09_29_000001_seed_pregnancy_calendar_note.php`), so `make schema-diff` row counts match.

`INSERT IGNORE` on the (`group`, `code`) unique key never overwrites an admin edit. Seeded clinical copy keeps
`needs_review = 1`. Store text as raw UTF-8 JSON (not `\uXXXX`).
