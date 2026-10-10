# N10 extras — scoping, partners and data sources (B-N10-01)

Status: proposed, to be reviewed with the user before B-N10-02..07 start. Docs only.
Binding inputs: `bloom/README.md` decisions (web + backend only; external services behind adapters with a `fake`
default; every screen light + dark), `roadmap/DECISIONS.md` #12–#15 (insurance = self-tracking, marketplace MVP without
online payment, everything run by the Ritme team in admin-web, internal COD shop supersedes link-out purchase).

## 0. Cross-cutting rules (apply to every extra)

1. **Health data never leaves for a partner.** No cycle phase, log, symptom, mode-derived health state, lab value, child
   data or diagnosis is ever put in a partner URL, request body, adapter call, UTM parameter or ad/shop event.
   Partner payloads are built from an explicit allow-list of fields typed by the user into *that* form.
2. **Shop and ads are separate from health data.** Shop/ad tables and queries never join health tables; no shop/ad code
   imports a health package (enforced by review + a grep check in the N10 audit). Targeting is by **app mode only**
   (adult / ttc / pregnancy / postpartum / menopause) and placement — never by logs, predictions, phase, conditions,
   labs, children, or purchase history. The design itself states it: «فروشگاه از داده سلامت تو جداست و پیشنهادهایش بر
   اساس ثبت‌هایت نیست.» (v17_Main).
3. **Where ads/shop promos must not appear:** teen mode, loss/bereavement mode, emergency/safety screens (`/emergency`,
   «اورژانس است؟»), consent/privacy screens, health-record, labs, doctor chat, and any urgent path.
4. **Adapters.** Every partner touchpoint is a Go interface with `fake` (default) and a config-switched real provider
   added later; secrets only from env; no real partner is contacted by N10. Admin-run data (listings, products,
   ads, insurance text) needs no adapter.
5. **No payment** in N10 (DECISIONS #13). **No Android.** Outward actions through partners are always user-initiated
   with an explicit consent line naming what is sent and to whom.
6. **Analytics:** impression/click/interest events carry only {placement|item id, mode, day}; no user id on ad events
   (aggregate counters), no health payload (existing no-health-in-analytics rule).
7. **Roadmap alignment.** The canvas queue extends N10 and must not be re-implemented here: CB-SHOP-* extends
   `internal/shop`, CB-DIR-* extends `internal/cityservices`, CB-INS-* adds `internal/insurance` and keeps B-N10-06 as
   the «how it works» page. N10 therefore ships *thin, extensible* schemas (below) and avoids anything the canvas
   supersedes (partner orders row, my-bookings list).

## 1. Home ad slots (B-N10-02)

- **Design shows** (`a-start-plus/nbl_Prem_TrialHome`): on Today, cards labelled «تبلیغ»: slider 342×170 (title, sub,
  «بیشتر»), square 166×166 pair (title + «خرید»/«ثبت‌نام»), strip 342×76 (title, sub). Also two *non-ad* «پیشنهاد»
  cards (child profile, home lab / checkup) — those belong to §3 / existing product logic, not to ad slots.
- **MVP:** reuse the existing banner system (`GET /api/v1/banners`, calendar-scheduled, admin CRUD with image upload).
  Add `kind = ad`, new slot values `ad_slider | ad_square | ad_strip`, advertiser name (admin-only), CTA label, link
  (internal route or external, external opens with `rel=noopener sponsored`), mode targeting (set of modes; empty = all),
  schedule, priority; «تبلیغ» label is rendered by the client, not editable away. Per-day impression/click counters
  (`POST /api/v1/banners/{id}/events`, rate-limited, no auth-identity stored).
- **Partner/adapter:** none. Advertisers are manual deals entered in admin-web. No ad network/SDK (privacy).
- **Privacy:** targeting = mode only; server picks banners from the mode the client sends/knows, never from logs;
  suppressed per §0.3; frequency cap done client-side (localStorage), not server-profile.
- **Data owner:** admin content (Ritme team).
- **Open questions:** Q1, Q2.

## 2. Shop entry and catalog (B-N10-03)

- **Design shows** (`d-doctor-assistant/nbl_v17_Main`, Services): «فروشگاه — خرید · ورود» block with two categories
  «سیسمونی و نوزاد» and «آرایشی و بهداشتی» plus the separation disclosure line. Me hub has «سفارش‌ها — ۱ مرسوله در راه».
- **MVP:** `internal/shop`: categories (the two above, admin-editable), products (title, image, price text, optional
  badge, `external_url`, partner label, active/order), public `GET /api/v1/shop/categories` + `/shop/products`,
  admin CRUD, shop screen = categories → product list → link-out (external, `noopener sponsored`, interstitial line
  «به سایت فروشنده می‌روی»). Keep a nullable `external_url` and no cart: CB-SHOP-01 extends this schema (internal COD
  shop, DECISIONS #15) and says link-out products must keep rendering until migrated.
- **Partner/adapter:** `ShopOrders` adapter interface exists only as a stub with `fake` returning «unsupported».
  **Recommendation: drop the Me «سفارش‌ها» partner-orders row** (DECISIONS #15; CB-SHOP-08 owns `/profile/orders`
  with internal orders; roadmap/PROGRESS.md already flags this question). The row stays hidden in N10.
- **Privacy:** separate tables/package, no health joins, no per-user browsing history stored; link-out URL carries no
  user identifiers, only an optional static `utm_source=ritme`.
- **Data owner:** admin content now; internal shop (Ritme team) from CB-SHOP.
- **Open questions:** Q3, Q4.

## 3. Home lab test and screening booking requests (B-N10-04)

- **Design shows:** only home «پیشنهاد» cards on Today (`nbl_Prem_TrialHome`): «آزمایش در منزل — نمونه‌گیری در خانه،
  جواب مستقیم در ریتمی — درخواست» and «چکاپ سالانه‌ات نزدیک است — پاپ‌اسمیر و آزمایش‌های دوره‌ای — رزرو». No dedicated
  form/status artboard exists (gap — screens designed from existing components, light+dark).
- **MVP:** `internal/partners`: catalog of orderable services (admin: title, description, partner, active, e.g. home
  blood draw, Pap smear/checkup); request form (service, name, phone, city/address, preferred day+time window, note
  — free text warned «اطلاعات بیماری ننویس»); `POST /api/v1/partners/lab-requests`, `GET` my requests, cancel;
  status machine requested → accepted → sampled → result_ready / rejected / cancelled; admin queue. `LabPartner`
  adapter (`Submit`, `Status`, `FetchResult`) with `fake` (admin-advanced statuses + a canned PDF result). Cards
  open the form; shown only when at least one service is active.
- **Results → labs (B-N6-06):** only if the user ticks a consent on the request (re-asked at result time); then the
  result file is copied into her labs/record storage as a user-owned document. Add dependency on **B-N6-06**.
- **Privacy:** payload to partner = allow-listed form fields only; consent line names the partner; requests encrypted
  at rest like other health-adjacent rows; address purged after completion window (Q6); card visibility must not be
  computed from health logs (static/admin rule, optionally «due» from her own checkup reminder evaluated on device).
- **Data owner:** partner (real lab) for fulfilment and results; Ritme admin for catalog/queue.
- **Open questions:** Q5, Q6, Q7.

## 4. City services — classes, pools, playhouses + bookings (B-N10-05)

- **Design shows** (`nbl_v17_Main`): Services tile «برای مادر و کودک — کلاس، استخر و خانه بازی نزدیک تو»; Me hub «رزروها —
  کلاس آب‌پری · چهارشنبه». No dedicated listing/detail artboards in bloom (the canvas adds them: `nbl_Dir_*`).
- **MVP:** `internal/cityservices`: cities (admin), categories (class / pool / playhouse, admin-editable), listings
  (name, category, city, district, address, phone, description, price text, photos via public disk, hours text,
  `booking_mode = link | request | phone`, booking URL, active); list with city + category filter, detail, booking:
  link-out, or a request (name, phone, preferred date, optional child age range, note) stored and shown in the admin
  queue, user can see status. Bookings list: a simple «رزروها» page; superseded by CB-DIR-09 (`/profile/bookings`), so
  keep it a thin list over the same table and expose it at that route name if free.
- **Partner/adapter:** none required; `BookingNotifier` interface (`fake` = log) to ping the venue by SMS/email later.
  Admin team confirms by phone (DECISIONS #14). No map in N10 (CB-DIR, Neshan).
- **Privacy:** requests carry contact + date only; **no child health/medical info, no cycle/pregnancy state** sent to
  venues; child details only age range, typed by the user. City choice stored in profile as a plain preference.
- **Data owner:** admin content (listings, cities); venue owns the real booking (outside the app in N10).
- **Open questions:** Q8, Q9.

## 5. Insurance info page (B-N10-06)

- **Design shows:** Services tile «بیمه — پوشش و خسارت» (`nbl_v17_Main`); the canvas adds full self-tracking
  (`nbl_Ins_*`, CB-INS-01).
- **MVP:** read-only admin-edited content page `/insurance` (coverage overview, how to file a claim steps, FAQ,
  document checklist, optional partner/insurer link-out), fa/en/ar via the existing content/i18n mechanism,
  admin editor (rich-text-lite or blocks, published flag). The tile hides while the page is unpublished.
- **Partner/adapter:** none. Link-out only.
- **Privacy:** zero personal/health data collected or sent; link carries no identifiers; copy states «ریتمی اطلاعات
  بیمه‌ات را به هیچ شرکتی نمی‌فرستد».
- **Data owner:** admin content.
- **Open questions:** Q10.

## 6. Devices & backup (B-N10-07)

- **Design shows** (`g-me-settings/nbl_Me_Hub`): «داده و دستگاه»: «ساعت و گجت‌ها — ۱ دستگاه وصل» and «پشتیبان و خروجی
  داده — آخرین پشتیبان: دیروز». The «۱ دستگاه وصل» sample implies pairing, which the web cannot honour: show the
  honest state instead.
- **MVP (frontend-only, per `touches`):**
  - Backup: screen with «last server sync» time (taken from the existing sync/last-updated signal; if none exists,
    from the latest successful authenticated API write — Q11), explanation that data lives in her account on the server,
    and manual export reusing the JSON export (B-N1-12) and the record export (`/record/export`, B-N6-04). No new
    backend; no cloud-drive upload.
  - Devices: «به‌زودی» state, list of the kinds of devices, an «به من خبر بده» interest toggle/button. Interest is
    sent through the existing analytics/support-event path as {device kind, locale} with no health data; if that
    path cannot carry it, B-N10-07 adds one small endpoint and its `touches` (Q12).
- **Partner/adapter:** none now. Real-integration notes to put in the screen's doc comment / `docs/night-bloom`:
  web cannot pair watches (Web Bluetooth is Chromium-only, no background sync); real route = native HealthKit /
  Health Connect (out of scope: no Android/native) or vendor cloud OAuth (Fitbit, Garmin, Huawei Health, Withings)
  with a `DeviceProvider` adapter, token vault, per-metric consent, import into vitals (B-N6 vitals), webhook/poll
  worker, and revoke/delete. Iranian availability of those clouds is uncertain.
- **Privacy:** export file contains health data and is generated on-device/for the user only; interest capture holds
  no health data.
- **Data owner:** user (her own data); future: device vendor.
- **Open questions:** Q11, Q12.

## 7. Open questions for the user (defaults used if unanswered)

| # | Question | Default |
|---|---|---|
| Q1 | Ads in Pregnancy/Postpartum/Menopause modes, or Adult/TTC only? Never in teen/loss/urgent is fixed. | All non-teen modes except loss, mode-targeted |
| Q2 | Who sells ads and do you need advertiser reports/invoicing (beyond impression/click counts in admin)? | Manual deals; counts only |
| Q3 | N10 shop is link-out; confirm the partner «سفارش‌ها» row is dropped in favour of the internal COD shop (DECISIONS #15)? | Dropped, row hidden |
| Q4 | Which product partners/links are first (seed categories only, or real products)? | Seed two categories, demo products via fake data |
| Q5 | Which real lab partner(s) for home sampling/Pap, and in which cities? | `fake` adapter only |
| Q6 | Retention: delete the address from lab requests after N days? Allow result auto-import into labs only with explicit consent? | 30 days after completion; consent required |
| Q7 | Should the «چکاپ سالانه» card appear for everyone, or only based on her own reminders (computed on device)? | Everyone in adult/TTC modes, no health inference |
| Q8 | Initial cities and venue categories; who enters listings and phones venues? | Tehran + admin team |
| Q9 | Keep the N10 «رزروها» list or skip straight to CB-DIR-09 (`/profile/bookings`)? | Thin list now, same table later |
| Q10 | Insurance copy and any insurer partner link: who writes the content? | Admin writes; no partner link |
| Q11 | What defines «آخرین پشتیبان» (no explicit backup concept exists since data is server-side)? | Last successful server write/sync time |
| Q12 | Interest capture for wearables: OK to add a tiny endpoint if the existing analytics path can't carry it? | Reuse analytics; add endpoint only if needed |
