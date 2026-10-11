# N10 audit (B-N10-08, audit part)

Every N10 screen in light + dark at 390 px (admin-web 1440) with **real data**: own Go API (`ritme_n1008` DB, migrations at
start, port 8108), an ad banner set (slider, 2 squares, strip), two shop categories with 3 products, 2 home-lab services
and one request, a city with 3 listings (class / pool / playhouse, photos) and one booking request, three insurance info
sections, an adult account and a teen-mode account. Dev servers: frontend :3108, admin-web :3109 (own dist dirs, removed).
Seeding went through the admin APIs. Screens: `docs/qa/bloom/B-N10-08/screens/` (+ `admin/`); side-by-sides for fixed
items: `docs/qa/bloom/B-N10-08/side-by-side/`.

No N10 artboards exist for most of these (extras.md §3-§6: screens designed from existing components), so the judge is the
neighbouring screens, `frontend/CLAUDE.md` and the extras.md rules. Data-driven differences are not deviations.

Severity: **high** = broken / misleading, **med** = visible departure, digit rules, **low** = polish.
Totals: **high 0 · med 1 · low 6** — fixed: 1 med; 6 low open.

## Privacy rules (extras.md §0) checked on real API responses

| # | Rule | Result |
|---|---|---|
| P1 | Ads: no user id on counters | `banner_daily_stats` = {banner_id, day, impressions, clicks}; `POST /banners/{id}/events` stores nothing else. `GET /banners/ads` omits the advertiser name. **Pass.** |
| P2 | Shop counters carry no user id | `shop_click_counts` = {product_id, day, clicks}; link-out URL only gets `utm_source=ritme`. **Pass.** |
| P3 | Teen: no ads, no banners, no shop | teen account: `/banners/ads` all three slots empty, `/banners` empty, `/shop/categories` empty (screen: «فعلاً دسته‌ای نیست»), home does not mount the slots. **Pass.** |
| P4 | Ads never on health / emergency paths | `pathAllowsAds` regex blocks teen/loss/emergency/safety/consent/health-record/labs/doctor/assistant/telemed; slots are mounted only in the cycle home. **Pass.** |
| P5 | No health payload to a partner | `partners.Submission` = reference, service code, name, phone, city, address, date, window, note (form fields only); `payload_enc` is sealed in the DB; the admin queue shows a masked mobile and never name/address/note. `internal/partners` imports `internal/labs` only for the user-consented result import. **Pass.** |
| P6 | City-service request carries contact + date + age range only | request body/row has no cycle / pregnancy / health fields. **Pass.** |

## Findings

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| M1 | Latin digits in Persian copy built from numbers: home-lab form retention note («… 30 روز …»), shop / city-services pager («صفحه 1 از 2»), city-service photo alt («عکس 1 از …»). Interpolated numbers bypass the locale. | med | `screens/home-lab/ui/HomeLabFormPage.tsx`, `screens/shop/ui/ShopCategoryPage.tsx`, `screens/city-services/ui/CityServicesPage.tsx`, `CityServiceDetailPage.tsx` | **Fixed** — `useFormatter().number()`. `side-by-side/M1-*.{light,dark}.png`. |
| L1 | Local-dev only: `img-src` allows `https:` only, so admin-uploaded images served from an `http://localhost` API are blocked by the CSP; ads then drop themselves (onError) and listing / booking thumbnails show a broken-image icon. Production and stage are https. Worked around for the shots with a temporary `img-src http://localhost:8108` (reverted). | low | `frontend/next.config.ts:67` | Note — add `http://localhost:*` to dev CSP if it bites. |
| L2 | Listing / booking thumbnails and the detail hero have no fallback when an image fails (broken-image glyph). | low | `screens/city-services/ui/*`, `screens/profile` bookings | Open — add a neutral placeholder. |
| L3 | Phones («شمارهٔ تماس», venue phone) are shown with Latin digits (`dir="ltr"`); admin-web date cells (`2026-10-16`, `سن کودک: 3-1`) are ISO / Latin while neighbours are Jalali / Persian. | low | `HomeLabRequestPage.tsx:170`, admin-web lab-requests / city-service-bookings | Open — pick one convention for phones; Jalali the preferred-day cells. |
| L4 | Admin banners list shows raw `ad_slider` / `ad_square` codes, no kind / advertiser column, and no impressions / clicks (the stats endpoint exists). | low | `admin-web/src/app/(panel)/banners` | Open — labels + a stats column. |
| L5 | Link-out interstitial «ادامه →» arrow points right while the flow is RTL (next = left). | low | `screens/shop/ui/ShopCategoryPage.tsx` | Open. |
| L6 | `/profile/backup` says «هنوز چیزی از این دستگاه ذخیره نشده» for an account that has written data but a fresh browser profile, because the signal is the last successful write seen on this device (Q11 default). | low | `screens/backup` | Note — accepted default; copy could say «روی این دستگاه». |
| A0 | Ads (slider 342x170, squares, strip) with «تبلیغ» label + CTA on the cycle home, home-lab «پیشنهاد» cards, `/shop` categories + disclosure line, category list + link-out interstitial, `/services/home-lab` list / form / request timeline, `/city-services` list / detail / bookings, `/insurance` (accordion, last update, no-send line), `/profile/devices`, `/profile/backup`, admin-web banners / shop / lab services / lab queue / city services / bookings / insurance sections: light + dark consistent, tokens only, touch targets fine. | — | — | match |

## Verify

frontend: `tsc --noEmit` ok, `eslint` (touched screens) clean, `fsd:lint` no problems, `lint:styles` passed, `lint:dark` passed,
vitest for the touched screens 8/8. admin-web untouched.
