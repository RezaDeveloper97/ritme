# Mother & child directory — schema (CB-DIR-01)

The canvas directory (boards `nbl_Dir_*`: map, list, place, book, my bookings, 4-step business join) is built **on top
of bloom's city services** (B-N10-05, goose `00072_city_services.sql`, Go package `internal/cityservices`). Nothing is
re-implemented: a directory **place** is a `city_service_listings` row, a directory **booking** is a
`city_service_bookings` row, categories stay in bloom's admin-editable `city_service_categories`. There is no parallel
`/places` route and no second admin section — CB-DIR-02..05 extend `/api/v1/city-services/*` and the existing
`/api/admin/v1/city-services/*` / `city-service-bookings/*` admin.

Migration: goose `00081_directory.sql`, Laravel twin `2026_10_11_000081_extend_city_services_for_directory.php`
(schema + the same seed rows; `make schema-diff` green). Integration test: `db/migrations/directory_int_test.go`.

## 1. Decisions that shape the schema

- **MVP, admin-managed, no online payment** (DECISIONS #13/#14). No payment / invoice / refund column anywhere. Prices
  are what the venue declared (`price_toman`, display only); the board's «پرداخت و رزرو» becomes «ارسال درخواست».
- **`online` booking mode** = an extra value of `city_service_listings.booking_mode` next to bloom's
  `link | request | phone` (the varchar(12) already fits — no DDL). It means "pick a slot from the Ritme calendar",
  but it still creates a booking in status `requested` that the Ritme team confirms in admin. **Never auto-confirm**,
  even though the JoinDocs board says «رزرو خودکار تأیید می‌شود».
- **«پیام به مجموعه»** has no messaging backend: hide it or route it to support. No messages table.
- **Legacy fallbacks.** bloom's `booking_mode`, `hours_text`, `price_text` and `photos` (public-disk paths) keep
  rendering until the structured fields (`weekly_hours`, services' `price_toman`, `photo_file_ids`) are filled. A
  reader prefers the structured field and falls back to the bloom one.
- **Privacy.** No health / medical field, no cycle or pregnancy state in any directory table. A booking may reference a
  bloom `children` row (`child_id`) only so the venue learns the child's first name and age (Dir_Book: «فقط نام تو،
  نام و سن کودک و شماره تماست»). Reviews / reports carry the author's user id for moderation only; a venue never sees
  an account id. Logs carry ids only.

## 2. Tables

### `city_service_listings` (bloom, extended)

| Added column | Meaning |
|---|---|
| `status` varchar(16) default `published` | `draft` · `pending` · `published` · `suspended`. Existing rows are `published`; `is_active` keeps its bloom meaning (admin on/off). Public reads need `is_active = 1 AND status = 'published'`. |
| `verified_at` timestamp NULL | «مجوز و مدارک بررسی شد» badge — set by the team after the document review / visit. Not a medical endorsement. |
| `owner_user_id` → users, NULL, on delete set null | the account whose approved application created / claimed the place («صاحب این مجموعه هستی؟»). |
| `lat`, `lng` decimal(9,6) NULL | map pin; index (`lat`,`lng`) for the bbox query (CB-DIR-02). |
| `about` text NULL | the long «درباره مجموعه» copy; bloom's `description` stays the short teaser. |
| `age_ranges` JSON list NULL | codes of catalog group `dir_age_ranges`. |
| `amenities` JSON list NULL | codes of catalog group `dir_amenities`. |
| `weekly_hours` JSON NULL | `[{"day":0..6,"open":"09:00","close":"20:00"}, …]`, day 0 = Saturday (Jalali week); a missing day = closed. Fallback `hours_text`. |
| `photo_file_ids` JSON list NULL | `files.id` of purpose `place_photo` (public, re-encoded WebP, CB-CORE-05). Fallback `photos`. |
| `website_url` varchar(1000) NULL | the «وب‌سایت» action (http/https only, validated in Go). |
| `cancel_free_hours` smallint NULL, `cancel_policy` varchar(300) NULL | «لغو رایگان تا ۲۴ ساعت قبل» + optional venue text. |
| `rules` text NULL | «قوانین مجموعه». |
| `rating_avg` decimal(3,2) NULL, `rating_count` int default 0 | denormalised from published reviews, recomputed by the review service (CB-DIR-03 / admin moderation). |

Index `(status, city_id, category_id)` for the list query.

### `city_service_services` (new)

A place's services and packages. `kind` `session` | `package`; `name` varchar(160) (venue copy, like the listing name —
not translatable); `description`; `format` `group` | `private` | NULL; `duration_minutes`; `age_min_months` /
`age_max_months` (pregnancy = NULL/NULL with the place's `pregnancy` age range); `capacity`; `price_toman` (display
only); `package_sessions`, `package_validity_days` (packages); `sort_order`, `is_active`. Cascade on listing delete.

### `city_service_slots` (new)

Open times. `listing_id` (cascade), `service_id` NULL (cascade; NULL = any service), `starts_at` / `ends_at` **datetime**
in Tehran wall-clock, `capacity`, `booked_count`, `status` `open` | `closed` | `cancelled`. «پر شد» =
`booked_count >= capacity`. The booking service (CB-DIR-03) increments `booked_count` in the same transaction as the
booking insert (`SELECT … FOR UPDATE` on the slot); a rejected / cancelled booking releases its seat.

### `city_service_bookings` (bloom, extended)

| Added column | Meaning |
|---|---|
| `code` varchar(16) NULL, unique | user-visible booking code; NULL on bloom-era rows. |
| `service_id` → services, `slot_id` → slots | NULL for bloom request-mode bookings; on delete set null (the booking survives). |
| `child_id` → children, NULL, on delete set null | which child (Dir_Book «برای کدام کودک؟»); bloom's `child_age_range` stays for typed ages. |
| `price_toman` int NULL | price snapshot shown at request time (display only). |
| `confirmed_at`, `cancelled_at`, `completed_at` timestamp NULL; `cancelled_by` `user` \| `admin` | status history. |

Kept: `contact_name`, `contact_phone`, `preferred_date` (= the slot's day when a slot is chosen), `note`, `status`
(`requested` → `confirmed` | `rejected` | `cancelled`; `confirmed` → `completed` | `cancelled` — bloom's
`cityservices.CanMove`).

### `city_service_reviews` (new)

One per **completed** booking (`booking_id` unique; the service checks `status = completed` and that the booking is
the author's). `rating` 1–5, sub-scores `score_cleanliness`, `score_instructor`, `score_amenities` 1–5 NULL, `body`
≤ 1000, `status` `published` | `hidden` (admin moderation). Cascade with booking / listing / user. The board's
«مادر رها» label is derived from the user's first name at read time — nothing else about the author is shown.

### `city_service_reports` (new)

«گزارش اطلاعات نادرست». `listing_id` (cascade), `user_id` NULL (set null), `reason` code (`wrong_info`, `closed`,
`wrong_location`, `other` — validated in Go), `note` ≤ 500, `status` `open` | `resolved` | `dismissed`, `resolved_at`.

### `city_service_applications` (new)

The 4-step join draft of one account (CB-DIR-04). `status` `draft` → `submitted` → `in_review` → `approved` |
`rejected` (`needs_info` back to the applicant). `step` 1–4 = the furthest completed step.

| Step | Columns |
|---|---|
| 1 Join | `business_name`, `category_slugs` (JSON list of `city_service_categories.slug`, several allowed), `city_id`, `district`, `contact_name`, `contact_phone`, `owner_confirmed` |
| 2 JoinForm | `lat`, `lng`, `address`, `about`, `age_ranges`, `amenities`, `weekly_hours`, `photo_file_ids` (same shapes as the listing; ≥ 4 photos to submit) |
| 3 services | `services` JSON list of `{kind, name, description?, format?, duration_minutes?, age_min_months?, age_max_months?, capacity?, price_toman?, package_sessions?, package_validity_days?}` — copied into `city_service_services` on approval |
| 4 JoinDocs | `booking_mode` (`online` \| `request` \| `phone`), `documents` JSON `{"licence": <file id>, "instructor_cert": <file id>, "health_cert": <file id>}` — **private** files of purpose `place_licence` owned by the applicant, readable only by the Ritme review team (signed links), `terms_accepted_at` |

On submit: `submitted_at` and a unique `tracking_code` (JoinDone «کد پیگیری»). On approval the admin creates (or
links) the listing → `listing_id`, `owner_user_id` on the listing, `status = published`. `reviewed_at`, `review_note`
record the decision. Deleting the account cascades its applications.

## 3. Seeds

| Where | Rows |
|---|---|
| `city_service_categories` | bloom's `class`, `pool`, `playhouse` + `music_art`, `postpartum_fitness`, `baby_massage`, `parenting_workshop`, `other` (the Dir_Join types). Categories stay in bloom's table — there is **no** `dir_categories` catalog group (one source of truth; listings FK to it). |
| `catalog_items` `dir_amenities` | `nursing_room`, `changing_table`, `female_instructor`, `stroller_parking`, `parking`, `elevator`, `wheelchair_access`; `meta {icon}` |
| `catalog_items` `dir_age_ranges` | `pregnancy` (`meta {pregnancy:true}`), `m0_6`, `m6_12`, `y1_2`, `y2_4`, `y4_6` (`meta {min_months, max_months}`) |

All `INSERT IGNORE` (an admin edit is never overwritten), `needs_review = 0` (labels, not clinical copy).
