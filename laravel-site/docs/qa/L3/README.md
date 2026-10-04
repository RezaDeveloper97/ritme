# L3-11 — Marketing pages fidelity + SEO sweep

Run on 2026-10-04 against `php artisan serve --port=8131` (dev SQLite, demo seeders), production build.

```bash
node tools/shot.mjs --all --widths 390,768,1440 --base http://127.0.0.1:8131 --out docs/qa/L3/sweep-null    # app/social links NULL (dev default)
node tools/shot.mjs --all --widths 390,768,1440 --base http://127.0.0.1:8131 --out docs/qa/L3/sweep-filled  # links = https://example.test/… (temporary)
php artisan seo:audit
```

The `pages` cache namespace was bumped before each run. For the filled run the `app_links` (bazaar, myket, google_play,
app_store, web_app) and `social` (instagram, telegram, linkedin) settings were set through `UpdateSettings` to fake
`https://example.test/…` URLs and then **set back to NULL** (checked in the DB afterwards: 0 non-NULL rows). PNGs are
git-ignored. Neither run had a hard failure: no external request, no console error, no failed request or HTTP ≥ 400 on
the Laravel side.

## Why the NULL column is always higher

With NULL app links the layout hides the header «ورود»/«دانلود اپ», every store badge (in the app CTA and the footer),
the QR code and the social links (L1-02 decision: nothing is linked until it is configured). That makes every page
170–600 px shorter than the design, and the diff counts the height mismatch fully. The **filled** column is the one
that measures fidelity. Production must fill these settings before go-live.

## Fixes made in this sweep

| Fix | Where | Effect (filled run) |
|---|---|---|
| **Flex-basis is now padding-aware.** The design CSS uses `content-box`, so its ≤1024 rule `flex:1 1 0 → flex-basis:300px` leaves out padding and border. The Tailwind port (border-box) used `max-lg:basis-75` on padded cards, so at 768 those cards stayed side by side where the design stacks them. Basis is now 300 + 2×padding (+ border): `basis-93` (p-9), `basis-93.5` (p-9 + border), `basis-91.5` (p-8 + border), `basis-89.5` (p-7 + border), `basis-86.5` (p-5.5 + border), and in promo-split `basis-95`/`[375px]`/`[383px]` (wide / dashed). | `components/ui/promo-split`, `pages/{plus,tools,about,privacy,social-responsibility}`, `pages/home/static` (#how tiles), `pages/directory/{business,join}` | services 768 24.89→11.93 %, plus 46.31→13.59 %, tools 35.83→7.94 %, about 9.49→2.10 %, social 15.11→1.57 %, privacy 14.30→7.47 % |
| **Feature card wraps at ≤1024 like the design.** The design's `max-lg` rule wraps every flex row, so in the 3-column tools grid at 768 the icon sits above the text. `x-cards.feature` now has `max-lg:flex-wrap` (on wide phone cards the text still fits next to the icon). | `components/cards/feature` | 768: cycle 3.99→1.24 %, ttc 3.93→1.31 %, pregnancy 9.87→1.79 %, postpartum 6.78→2.45 %, menopause 6.44→2.03 %, teen 3.91→1.40 % |
| `/search` title was 22 characters (`seo:audit` error) and is now «جستجو در مقاله‌ها و پرسش‌های ریتمی». | `lang/fa/search.php` | audit green |

## Results

Diff % per width: **NULL links / filled links**. Where this sweep changed a number it reads `before→after`. Audit =
`php artisan seo:audit` (every page also has a warning that og:image is missing until a default OG image is uploaded).

| Page | 390 | 768 | 1440 | Audit | Cause / decision for widths over 3 % (filled) |
|---|---|---|---|---|---|
| `/` | 42.50 / 42.04 | 7.08 / 1.57 | 4.62 / 1.03 | ✔ | 390: **deliberate design fix** (L3-02). The design keeps the `#how` card's 120 px side margins on phones, which leaves a text column about 22 px wide and makes the page 1.8k px taller. Sections above and below match (L3-02: 3.45 / 2.51 %). |
| `/cycle` | 7.47 / 1.32 | 9.05→7.18 / 3.99→1.24 | 4.85 / 0.73 | ✔ | green |
| `/ttc` | 7.87 / 1.52 | 9.03→7.31 / 3.93→1.31 | 4.73 / 0.77 | ✔ | green |
| `/pregnancy` | 8.43 / 2.38 | 13.13→7.54 / 9.87→1.79 | 5.16 / 1.24 | ✔ | green |
| `/postpartum` | 10.33 / 6.25 | 11.26→8.37 / 6.78→2.45 | 5.77 / 1.87 | ✔ | 390: the readings block shows real seeded posts (two-line titles, different covers), and the FAQ has a descriptive h2 (L3-03 decision). Everything above the readings is 1.18 % (L3-05). |
| `/menopause` | 9.54 / 6.47 | 10.97→8.00 / 6.44→2.03 | 6.28 / 3.89 | ✔ | Same readings cause (+31/+61 px). Above the readings: 1.62 %. |
| `/teen` | 10.28 / 5.18 | 10.08→8.62 / 3.91→1.40 | 5.48 / 0.86 | ✔ | 390: same readings cause (+30 px). |
| `/services` | 12.11 / 1.31 | 30.60→18.77 / 24.89→11.93 | 8.23 / 0.77 | ✔ | 768: **design render bug not copied**. The design's `max-lg` flex-wrap pushes each check icon in the card lists onto its own line above the text (cards grow by about 80 px). We keep the icon inline. |
| `/plus` | 26.33 / 16.20 | 51.67→20.47 / 46.31→13.59 | 9.89 / 1.16 | ✔ | Placeholder price rows («[قیمت سالانه]», «[امکانات دیگر پلاس]», «[بسته‌های دیگر]») are dropped and «قیمت در اپ» is shown instead (L3-06, owner decision pending). The design's check-icon wrap bug is not copied. |
| `/tools` | 16.29 / 9.77 | 38.09→15.39 / 35.83→7.94 | 8.62 / 1.90 | ✔ | Calculators add a two-line estimate note plus a next-period detail (L3-07). The design's checklist check-icon wrap bug at 768 is not copied. |
| `/about` | 5.45 / 3.39 | 10.75→3.43 / 9.49→2.10 | 6.62 / 1.29 | ✔ | 390: 3.39 % from small text-wrap differences (−8 px). Accepted. |
| `/social-responsibility` | 15.82 / 8.66 | 19.85→9.27 / 15.11→1.57 | 8.17 / 4.64 | ✔ | Donation amounts are replaced by a contact CTA (AUDIT §8 and the COD-only decision). The transparency link stays hidden until its URL is set (L3-08). |
| `/privacy` | 6.81 / 4.48 | 16.42→5.38 / 14.30→7.47 | 8.25 / 1.01 | ✔ | Policy card shows the real date «۱۲ مهر ۱۴۰۵» and «[ایمیل مسئول داده]» instead of the short `[تاریخ]`/`[ایمیل]`. The longer line wraps and moves the button below it at 768/390. Real content. |
| `/terms` | — | — | — | ✔ | No design page (AUDIT §8) |
| `/faq` | 16.19 / 1.54 | 14.37 / 2.99 | 8.51 / 2.01 | ✔ | green |
| `/contact` | 8.26 / 5.47 | 4.97 / 3.87 | 9.77 / 5.43 | ✔ | In the design, the first FAQ answer's second paragraph (the privacy-channel note) is a smaller 14 px/600 line with a linked email. Our FAQ items render seeded admin HTML in one answer style, and the placeholder email is not a link. Accepted until real contact copy exists (L3-10: 4.46 % above the footer). |
| `/blog` | 28.25 / 22.25 | 17.45 / 7.62 | 14.62 / 8.58 | ✔ | Seeded content. The design repeats the featured post as the 6th card. Reading times and covers are real, and the featured cover uses the category illustration (L4-02: 2.35 / 8.66 % with equal content). |
| `/blog/period-pain` | 27.41 / 29.43 | 22.75 / 21.11 | 18.23 / 12.05 | ✔ | The seeded body lacks the design's red callout, and the sidebar stacks on mobile. Article region 1.73 / 3.46 % (L4-03). |
| `/directory` | 49.16 / 46.91 | 21.88 / 20.47 | 27.75 / 21.69 | ✔ (noindex: demo-only) | 6 demo places instead of the mock's 4-page pagination. No fake ratings or distances. A local areas panel replaces the map (no-maps decision). The design's broken 390 search form is fixed (L5-02). |
| `/directory/place/ab-pari` | 46.73 / 46.71 | 24.51 / 23.26 | 14.04 / 13.77 | ✔ (noindex: demo) | Mobile kit gallery has 1 tile vs the design's 5 stacked. Booking widget is L5-04. No fake rating bars. 9 vs 8 amenities (L5-03). |
| `/directory/business` | 25.81 / 23.25 | 22.34 / 20.81 | 14.36 / 9.07 | ✔ | No fake rating on the sample card. The design's 390 CTA margin bug is fixed. FAQ uses seeded group copy (L5-05). |
| `/directory/join` | 67.32 / 67.29 | 56.56 / 56.55 | 67.99 / 67.77 | ✔ (noindex) | The design shows step 2 of the multi-step form, and the route shows step 1 (`stepper.js`). Step-2 comparison: 33.9 / 21.6 % (L5-05). The map picker is replaced by address + lat/lng fields. |
| `/directory/join/done` | 14.89 / 9.01 | 16.53 / 12.71 | 21.92 / 10.80 | ✔ (noindex) | Shorter page: no sample photos, and the review time is still the placeholder (L5-05). |
| `/directory/booked/{code}` | — | — | — | — | Skipped by the tool (needs a parameter). Booking is L5-04, not built yet. |
| `/shop*` (5 routes) | 46–87 | 69–84 | 46–75 | ✘ title length | **Not converted yet**: L6 placeholders. A Shop agent is working on L6 now. Out of scope for L3-11. |

Utility pages: `/search` ✔ (fixed); `/offline` ✘ (title 25 / description 65 chars; noindex fallback page, text set in
`App\Http\Controllers\Pwa\OfflineController`, outside this task's touches); `/_components` (non-production preview) ✔.
Also audited: `/blog/category/cycle`, `/directory/tehran` ✔.

## Links, `#download`, external requests

- **Internal links**: a crawl of all 23 marketing routes (65 unique internal URLs, fragments checked against `id`s on
  the target page) found no broken link, no redirect and no missing anchor. `seo:audit` found no `href="#"`, one h1 per
  page, and no duplicate titles or descriptions.
- **`#download`**: pages with an app CTA section (`id="download"`) link `#download`. Pages without one (about,
  privacy, terms, contact, directory pages, article header) link `/#download`. All store links (Bazaar, Myket, Google
  Play, App Store, web app) and social links come from the `app_links`/`social` settings: with fake values every badge
  pointed to the configured URL, and with NULL they disappear. `resources/views` and `lang/fa` contain no hard-coded
  store or external URL (only doc comments).
- **External requests**: none in either run (`shot.mjs` blocks any non-local request and fails on it). The only
  absolute external URLs in the HTML are the directory place page's Neshan/Balad/Google Maps deep links. They are
  outbound `<a href>` navigation, not requests, as the maps decision allows.

## Open items

- Fill `app_links` + `social` settings before go-live (otherwise the CTAs, footer badges and login stay hidden).
- Upload a default OG image (`seo.default_og_media_id`): it clears the og:image warning on every page.
- `/offline` title/description are too short for the audit (controller outside touches; noindex page).
- Contact FAQ note styling and `/plus` prices depend on owner copy and decisions (L3-06, L3-10 open items).
- Design-bug decisions kept: check icons are not wrapped at ≤1024 (services, plus, tools), and the `#how` margin is fixed on phones.
