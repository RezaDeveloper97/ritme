# Accessibility pass — WCAG 2.1 AA (L9-03)

Date: 2026-10-05 · Standard: WCAG 2.1 AA (+ 2.2 AA 2.5.8 target size) · Target (tasks/README.md): Lighthouse a11y ≥ 95
on every route.

**Result: Lighthouse accessibility 100 on all 34 sampled templates (mobile + desktop); 84 page × width runs (42
pages/states × 390 + 1440) clean under axe-core + the custom checks; 10 of 10 runnable keyboard walkthroughs pass. One
walkthrough (the place gallery lightbox) is skipped because the dev database has no photos.**

## How to re-run

```bash
php artisan serve                                                    # or any base URL; PAGE_CACHE off is best
node docs/qa/a11y/a11y-check.mjs --base http://127.0.0.1:8000                       # every page, 390 + 1440
node docs/qa/a11y/a11y-check.mjs --base … --only shop,cart --widths 390             # a subset
node docs/qa/a11y/a11y-check.mjs --base … --json docs/qa/a11y/results.json          # the committed summary
```

`a11y-check.mjs` uses headless system Chrome over CDP, through the helpers in `tools/critical.mjs`. It needs no
npm dependency of its own:

- **axe-core** comes from a local file: `AXE_PATH`, `./node_modules/axe-core`, or the monorepo's
  `../frontend/node_modules/axe-core` (4.11.4 here). It is never loaded from a CDN, and it is not added to this
  package. Without it the custom checks still run. Tags: `wcag2a wcag2aa wcag21a wcag21aa best-practice`, with
  `target-size` enabled (24 px).
- **Text over gradients and images.** axe cannot judge text on gradients, so the script measures it in pixels.
  The glyphs are hidden through the CSSOM. The script then takes a screenshot of each line box, and the 10th
  percentile of background pixels must reach 4.5:1 (3:1 for large text). Text inside `aria-hidden` mockups is
  incidental and exempt under 1.4.3.
- **Custom checks:**
  - `lang="fa"` and `dir="rtl"`.
  - Exactly one `<h1>` and no skipped heading levels.
  - One `<main>`, and a skip link with an existing target that is the first tab stop.
  - Every `aria-invalid` control has a non-empty `aria-describedby` error.
  - With `prefers-reduced-motion: reduce`, no running motion animation or transition.
  - No request leaves 127.0.0.1.
- **Keyboard walk.** The script presses Tab up to 250 times through each page. Every stop must be visible: a
  non-zero box, not inside `hidden`, `inert` or `aria-hidden`. Every stop must also show a focus indicator, meaning
  outline, box-shadow, border or background differs between the focused and blurred state on the element, its
  label, its parent or its peer. A colour-only border change is flagged as `focus-weak`. Stops under 44 px are
  listed as info (`small` in the JSON).
- **Flows:** `cart` and `checkout` add a buyable product first. `*-errors` submits the empty form after the
  3-second time trap (`FormTimer::MIN_SECONDS`), so the server-side error state is what gets audited.
- **Keyboard walkthroughs.** These are scripted key presses with expectations, in `SCENARIOS`; see the table below.

Lighthouse (a11y category only) ran per sample URL, mobile and `--preset=desktop`, with the Lighthouse 13.4 already
in the npm cache (`npx lighthouse … --only-categories=accessibility`). Lighthouse cannot drive the session flows. The
cart, checkout and error states are covered by axe above.

## Fixes made

| # | Issue (WCAG) | Where | Fix |
|---|---|---|---|
| 1 | Pregnancy pink `#D9447F` as 13 px label text: 4.13:1 on white (1.4.3, AUDIT §6) | article card stage label (`/blog/*`, stage «برای همین مرحله», home) | New token `--color-stage-pregnancy-ink: #b8336b` (5.64:1) in `@theme`, used only for the text label. Tiles, gradients and icons keep `#D9447F`; icons on the /10 tint are 3.6:1, which passes 1.4.11. |
| 2 | Primary `#6E54F0` CTA text on `bg-lavender`: 4.13:1 (1.4.3) | `x-ui.promo-split` lavender card («ورود به …», `/services`, `/shop`) | The lavender tone uses `text-primary-hover` (#4428B8, 7.7:1). The other tones keep primary. |
| 3 | Hero eyebrow pill: lilac text on `lilac/14` over the hero glow measured 3.4–4.2:1 (1.4.3) | `x-ui.eyebrow pill tone=dark`: home + 6 stage heroes | Pill fill `bg-night-card/80` (border unchanged), now ≥ 6:1. Same shape and outline. |
| 4 | Closed opening-hour tiles faded with `opacity-50`: 3.3:1 (1.4.3) | `x-directory.hours` (place page) | Closed tiles are dashed, `bg-canvas` and muted text (6.2:1), with no opacity. |
| 5 | Focus on the calculator inputs, FAQ search and price-filter inputs shown only by a 1.5 px border colour change; the inputs had `outline-none` (2.4.7) | `/tools`, `/faq`, `x-shop.filters` | Wrappers get `focus-within:ring-2 ring-primary`, the same pattern as the directory search. The filter inputs drop `focus:outline-none`, so they get the global `:focus-visible` outline. |
| 6 | `<aside>` (complementary) nested inside `<main>` (landmark best practice; Lighthouse/axe `landmark-complementary-is-top-level`) | shop filters, cart summary, checkout summary, directory areas map, join preview, booked page | Now `<section aria-labelledby/label>` (a named region). On the booked page it is a `<div>`, because the column has no name. `blog/show` and the place booking card stay `<aside>`: inside `<article>` they are scoped and correct. |
| 7 | Radio-group errors not tied to the group (3.3.1 / 1.3.1) | contact «موضوع», join «نحوه رزرو» | `x-ui.form.field as="fieldset"` with `for` now puts `aria-describedby="{for}-error"` on the `<fieldset>`. |
| 8 | `aria-label` on a `<b>` (generic role: the label is prohibited and ignored) (4.1.2) | cart line quantity | The visible number is `aria-hidden`, and the full «تعداد: n» is `sr-only` text. |
| 9 | Calculator result announced from a region that was itself un-hidden, which is often not announced (4.1.3) | `/tools` | A persistent `aria-live="polite" aria-atomic` wrapper around the result. While the result is hidden, a negative margin cancels the form gap, so the layout is unchanged. |
| 10 | Reduced motion depended on each component (2.3.3) | global | A `prefers-reduced-motion: reduce` safety net in `@layer base` sets animation and transition to 0.01 ms and scroll-behavior to auto. The join stepper's `scrollIntoView` honours it. |

After the CSS and class changes, `npm run build` regenerated the critical CSS. The budget check is still green on
every template.

## Per-page results

Columns: axe = axe-core violations after the fixes; px = gradient/image text nodes measured in pixels (all pass);
Tab = tab stops at 390 / 1440; LH = Lighthouse a11y, mobile / desktop.

| Page (sample) | Found → fixed | axe | px | Tab | LH |
|---|---|---|---|---|---|
| Home `/` | eyebrow pill contrast (3) | 0 | 12/16 | 52 / 59 | 100 / 100 |
| Stage ×6 `/cycle` `/ttc` `/pregnancy` `/postpartum` `/menopause` `/teen` | eyebrow pill contrast (3); stage-nav chips, nav and hero lead measured ≥ 4.5 | 0 | 9–14 | 46–52 / 53–59 | 100 / 100 |
| Services `/services` | promo CTA contrast (2) | 0 | 2 | 30 / 37 | 100 / 100 |
| Plus `/plus` | none | 0 | ✓ | ✓ | 100 / 100 |
| Tools `/tools` | input focus (5), live region (9) | 0 | 2 | 51 / 58 | 100 / 100 |
| About, Social responsibility, Privacy, Terms | none (lilac h1 highlight on night measured ≥ 3:1 large) | 0 | 3–6 | 28–30 | 100 / 100 |
| FAQ `/faq` | search focus (5) | 0 | 1 | 45 / 52 | 100 / 100 |
| Contact `/contact` (+ error state) | topic group error link (7) | 0 | 1 | 37 / 44 | 100 / 100 |
| Blog `/blog`, category, tag, author | pregnancy label (1) | 0 | ✓ | ✓ | 100 / 100 |
| Post `/blog/period-pain` | pregnancy label in related cards (1) | 0 | 0 | 44 | 100 / 100 |
| Search `/search?q=…` and empty | none | 0 | ✓ | ✓ | 100 / 100 |
| Shop `/shop` | promo CTA contrast (2) | 0 | 0 | ✓ | 100 / 100 |
| Shop category `/shop/category/baby` | filters aside → section (6), price input focus (5) | 0 | 0 | 76 / 83 | 100 / 100 |
| Product `/shop/product/…` | none | 0 | 0 | 60 | 100 / 100 |
| Cart (empty, with a line) | aside → section (6), quantity label (8) | 0 | 0 | 38–41 / 45–48 | 100 / 100 (empty) |
| Checkout (+ error state, 5 invalid fields all described) | aside → section (6) | 0 | 0 | 39 / 46 | axe only |
| Directory `/directory`, city, city/category | areas aside → section (6) | 0 | 4–6 | 64–73 / 71–80 | 100 / 100 |
| Place `/directory/place/aramesh` | closed-day contrast (4) | 0 | 0 | 50 / 57 | 100 / 100 |
| Business `/directory/business` | none | 0 | 4 | 34 / 41 | 100 / 100 |
| Join `/directory/join` (+ error state, 5 invalid fields) | preview aside → section (6), booking-mode group error (7), reduced-motion scroll (10) | 0 | 0 | 37 / 44 | 100 / 100 |
| Join done, Offline, 404 | none | 0 | ✓ | ✓ | 100 / 100 (404: axe only) |
| Booked `/directory/booked/{code}`, Order `/shop/order/{code}` | booked aside → div (6) | not run (no booking or order rows in the dev DB); reviewed in code | | | |

Checked everywhere: `lang="fa"` + `dir="rtl"`; one `<h1>`; heading order; one `<main>`; skip link first and working;
footer `<nav>`; every image has alt (or is decorative) plus width and height (the `x-picture` contract); no external
request.

## Keyboard walkthroughs (390 px, scripted)

| Widget | Tab | Enter / Space | Esc | Arrows | Result |
|---|---|---|---|---|---|
| Skip link | 1st stop | Enter → focus on `<main id=main>` | – | – | ✔ |
| Mobile menu (`menu`) | burger | Enter opens (`aria-expanded=true`); next Tab enters the menu | closes, focus back on burger | – | ✔ |
| FAQ accordion (native `details`) | each summary | Enter opens, Space closes | – | – | ✔ |
| FAQ search (`faq-filter`) | input | typing filters, empty note shown | – | – | ✔ |
| Calculators (`calculators`) | inputs → submit | Enter submits; invalid → error text + `aria-invalid` + focus stays; valid → result in live region | – | – | ✔ |
| Shop filters (GET form) | checkboxes, inputs, submit | Space toggles | – | – | ✔ |
| Product variants (radios) | one stop per group | – | – | ← → move the selection | ✔ |
| Cart quantity stepper (`cart`) | + / − / remove | Enter changes quantity (fetch); focus returns to the same button | – | – | ✔ |
| Join stepper (`stepper`) | fields → «ذخیره و ادامه» | Enter with empty fields → focus on the first invalid field + native message | – | – | ✔ |
| Place / product gallery lightbox (`gallery`, native `<dialog>`) | opener | Enter opens (`showModal`, so the page is inert and focus stays in the dialog) | closes, focus to opener | ← next / → previous (RTL) | skipped live (no media in dev DB); code review ✔ |

## Contrast tokens (on the backgrounds they are used on)

| Pair | Ratio | Use |
|---|---|---|
| ink #231B3B / white | 15.9 | body |
| muted #5E5873 / white · canvas · lavender | 6.7 · 6.2 · 5.6 | secondary text |
| primary #6E54F0 / white · canvas | 5.0 · 4.6 | links, labels |
| primary / lavender #ECE6FF | **4.1 ✘** → primary-hover #4428B8 used there | promo CTA (fix 2) |
| stage-pregnancy #D9447F / white | **4.1 ✘** → stage-pregnancy-ink #B8336B 5.6 | small labels (fix 1) |
| stage cycle · ttc · postpartum · teen / white | 6.0 · 5.9 · 5.4 · 6.0 | labels |
| on-night-muted #B8AED6 / night | 8.7 | dark bands |
| lilac #B9A6FF / night-card | 7.8 | hero eyebrow (fix 3) |
| placeholder #9A93B3 / white | 2.9 | placeholders only (labels are always visible; never used for text) |
| focus outline primary / night | 3.6 | ≥ 3:1 non-text (1.4.11) |

## Documented, not changed

- **Touch targets.** The WCAG 2.2 AA 2.5.8 minimum (24 px, or spacing) passes everywhere (axe `target-size`). The
  task's 44 px comfort size is not met by some targets, which are kept to preserve the design:
  - footer and desktop-nav text links (22 px tall, in spaced lists)
  - wishlist hearts on product cards (40 × 40)
  - cart ± (34 × 34)
  - shop category chips (22–24 px)
  - burger is 46 × 46, and mobile-menu links are 48 px tall.

  `results.json` → `small` lists each one.
- **10.5 px «دریافت از»** on the store badges is kept as designed: 75 % ink/on-night, ≥ 8:1. There is no size
  requirement in AA. Mockup text (11 px) sits inside `aria-hidden` phone mockups and counts as incidental.
- **Mixed Latin text.** Emails, phones, URLs, codes and Latin inputs already carry `dir="ltr"` or `<bdi>`
  (contact, privacy DPO, join, checkout, booked, share). «نسخه iOS» is short and needs no isolation.
- **Placeholder colour** (#9A93B3, 2.9:1) is used only for placeholders, and every field has a visible label.

## Open items

- Run the gallery lightbox walkthrough once photos exist: upload a place or product photo, then run
  `node docs/qa/a11y/a11y-check.mjs --only place`. The scenario runs on its own when a `<dialog>` is present.
- Audit booked and order confirmation pages live once a booking or an order exists. They share
  `x-ui.success-hero` and the layout, which are already covered.
- Manual screen-reader pass (VoiceOver / TalkBack, Persian voice). The automated checks cannot judge announcement
  quality.
