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

## 4. Business application API (CB-DIR-04)

Package `backend-go/internal/cityservices/join` (queries `db/queries/cityservices_join`, routes
`internal/http/routes_city_services_join.go`, OpenAPI tag `CityServiceApplications`, deviations **D-DIR-04**).
All under `/api/v1/city-services/applications`, auth:api, owner-only (a foreign id is the same 404 as a missing one).

| Route | What |
|---|---|
| `GET /` · `POST /` | my applications · open a draft (201) or get the open draft (200); ≤ 10 rows per account |
| `GET /{id}` · `DELETE /{id}` | full application (photos with public URLs, documents with 5-min signed links) · delete a draft + its files |
| `GET /{id}/status` | status + JoinDone timeline `submitted → documents_review → expert_contact → published` |
| `PUT /{id}/steps/{1..4}` | auto-save one step (whole step form; lenient — completeness only on submit) |
| `POST /{id}/photos` · `DELETE /{id}/photos/{fileId}` | multipart `file` → `files` purpose `place_photo_pending` (private, re-encoded WebP, 5-minute signed links for the owner and the review team); ≤ 20; adding also while submitted / in_review (still pending). Approval promotes each to a public `place_photo` (new id, used by the listing and kept on the application); rejection deletes them (decision 2026-10-12) |
| `PUT /{id}/documents/{slot}` · `DELETE …` | slots `licence`, `instructor_cert`, `health_cert`; multipart `file` → purpose `place_licence` (**private**, encrypted, signed links only) |
| `POST /{id}/submit` | 422 `application_incomplete` with per-field errors, else `submitted` + tracking code + SMS (fake provider) |

- **Submit needs**: step-1 fields + `owner_confirmed`, map pin + address, ≥ 1 age range, ≥ 1 open day, **≥ 4 live
  photos**, ≥ 1 service, booking mode, accepted terms, `licence` (+ `health_cert` when the categories include `pool`).
- **Edits** only in `draft` / `needs_info` (else 409 `application_locked`); a resubmission keeps the tracking code and
  clears `review_note`.
- **Tracking code**: 12 symbols of `23456789ABCDEFGHJKLMNPQRSTUVWXYZ` (60 random bits), `XXXX-XXXX-XXXX`. It identifies;
  it never authorizes (the status route is owner-scoped).
- **Files**: never accepted as ids from the body — only from the application's own upload routes, so an applicant can
  only reference files she uploaded there. Replacing / removing a document or photo deletes the file.
- **Limits**: uploads 60 / hour, submits 5 / hour per account (localized 429), plus the shared write throttle.
- **Logs** carry ids only (no name, phone, address or tracking code).
- **Open**: a real SMS template for the submit confirmation (the notifier is fake); photos are public from upload
  (before review) — the board allows adding them after submit.

## 5. Public API (CB-DIR-02)

bloom's `/api/v1/city-services/{options,listings,listings/{id}}` are extended in place (code:
`internal/cityservices/directory*.go`) and `GET /listings/{id}/slots` is added. These reads are **public** (no login,
for web SEO), throttled to 120 requests per minute per IP (`city-services-public`) and sent with
`Cache-Control: public|private, max-age=60` + `Vary: Accept-Language, Authorization`. The bookings routes still need a session.

- **Visibility.** A place is only ever read when `is_active = 1 AND status = 'published'` and its city and category are
  active. Draft, pending and suspended places return 404 everywhere, including when booking. `owner_user_id`,
  applications, licence files and booking rows are never selected. Only public `place_photo` files resolve to URLs.
- **Options** add `amenities` / `place_age_ranges` (catalog, picked labels + meta) and `sorts`. **Categories with no
  visible listing are hidden** (bloom listed every active category), the same rule bloom already used for cities.
- **List params**: `q`, `age_months` (0–216), `open_now`, `amenities` (comma list, all required), `lat`+`lng` (+
  `radius_km` ≤ 100), `bbox=min_lng,min_lat,max_lng,max_lat`, `sort=rank|distance|rating`. SQL narrows the results by
  city, category, q, amenities, the bounding box and the radius box (up to 1000 rows). Go then checks the exact radius,
  age fit (catalog ranges or an active service with age bounds; places with no age data stay), and open-now
  (`weekly_hours`, Tehran time, day 0 = Saturday).
- **Ranking** (`rank`) = 0.5 distance (`1/(1+km/2)`, when a position is sent) + 0.25 age fit (when an age is sent)
  + 0.25 Bayesian rating (prior 4.0 × 5 votes), with the weights renormalised to the signals in play. No paid
  signal exists. bloom's `sort_order`, then `id`, only break exact ties.
- **Card fields** (added after bloom's): `lat`, `lng`, `distance_km`, `verified`, `rating_avg`, `rating_count`, `age_ranges`,
  `amenities`, `price_from_toman`, `open_now` (null = unknown), `next_slot_at` (next open slot with a free seat),
  `age_fit`. The **place page** adds `about`, `website_url` (http/https only), `weekly_hours`, `rules`,
  `cancel_free_hours`, `cancel_policy`, `services`, `next_slots` (31 days, up to 8, full slots flagged), and
  `reviews {rating_avg, rating_count, scores, latest[3]}`. Only published reviews are used, and each author is shown
  by first name only.
- **Slots**: `?service_id=&from=Y-m-d&days=1..31` (14 by default). Results are future, `open` slots of active services
  (or of the whole place), and include `seats_left` / `is_full`.
