# Components (L3-01)

Shared Blade components that page tasks assemble pages from. Every component is an **anonymous** Blade component;
its full prop list and a usage example sit in the `{{-- … --}}` header of its file. Live review page:
**`/_components`** (route `preview.components`, non-production only; `resources/views/previews/components.blade.php`).
Screenshots: `docs/qa/L3-01/` (`node tools/shot.mjs --design index.html --route /_components --out docs/qa/L3-01`;
the diff number is meaningless for the kit page).

Rules that hold for all of them:

- **No queries.** Data arrives as props: scalars, arrays of props, or DTOs (`PostCardData`, `AppLinksSettings`,
  `MediaData`/media ids for `<x-picture>`). Controllers prepare them; settings are never read inside a component.
- No inline `style`, no raw hex, RTL logical utilities only, design tokens from `@theme` (tested in
  `tests/Feature/Components/ComponentKitTest.php`).
- **No JS.** The design has no real tabs (AUDIT §2 correction) — tab-like UI is `x-ui.chip-nav` (links). Accordion is
  native `<details>`, radio cards use CSS `has-checked:`. `resources/js/modules/` gained nothing.
- Persian digits: numbers passed as integers/floats are rendered with Persian digits (`۴٫۸`, `۱٬۲۵۰`). Dates/times are
  passed as ready display strings (format them with the Jalali helper before handing them over).
- Headings: card titles are `h3` (`as` prop), block titles `h2`; only `x-ui.page-intro` and `x-ui.success-hero` emit an
  `h1` (both take `as` to demote). Give section headers an `id` and point `aria-labelledby` at it.
- Stage colours (`color` / `stage` / `tint` props): `cycle`, `ttc`, `pregnancy`, `postpartum`, `menopause`, `teen`,
  plus `primary`, `muted` (and `lilac`, `primary-soft`, `surface` on `x-ui.icon-tile`). `LifeStage` enums are accepted
  where a stage is expected.

## Existing (L1-02 / L0-07 / L2-02) — unchanged

| Component | Notes |
|---|---|
| `x-ui.button` | `href`, `variant` primary\|outline\|ghost\|danger, `tone` light\|dark, `size` sm\|md\|lg\|xl\|2xl, `icon`, `icon-end`, `icon-class` |
| `x-ui.badge` | `tone` lavender\|success\|danger\|warn\|info\|night\|surface\|none, `icon` |
| `x-ui.section` / `x-ui.container` | bands with gutters: `pad` xl\|lg\|md\|sm\|none, `bg` canvas\|surface\|night\|none, `as` |
| `x-ui.breadcrumbs` | class component; registers BreadcrumbList JSON-LD |
| `x-icon`, `x-illustration`, `x-picture` | sprite icon, inlined illustration, responsive image |

## Primitives (`x-ui.*`)

| Component | Key props | Example | Design pages |
|---|---|---|---|
| `icon-tile` | `icon`, `color`, `size` sm(48)\|md(56)\|lg(120), `shape` rounded\|circle | `<x-ui.icon-tile icon="drop" color="cycle" shape="circle"/>` | inside every card, promise banner |
| `eyebrow` | `tone`, `color` primary\|cycle, `pill`, `icon` | `<x-ui.eyebrow tone="dark" pill icon="sprout">…</x-ui.eyebrow>` | nearly all sections; hero pill on stage heroes |
| `section-header` | `eyebrow`, `title`, `lead`, `align` start\|center, `tone`, `size` xl\|lg\|md\|sm, `as`, `id`, `more-href`, `more-label`, `more-icon` | `<x-ui.section-header eyebrow="مجله" title="برای همین مرحله" id="r-title" more-href="{{ route('blog.index') }}"/>` | all content pages |
| `pill` | `tone` teen\|primary\|surface\|danger\|muted, `dot`, `icon`, `icon-class`, `size` xs\|sm\|md | `<x-ui.pill dot>امروز ۱۷:۰۰</x-ui.pill>` | shop*, directory*, cards |
| `chip-nav` | `items` [label, href, active?, icon?], `label` (nav aria-label), `size` sm\|md\|lg, `current` page\|true, `vertical` | `<x-ui.chip-nav label="دسته‌های مجله" :items="$chips"/>` | blog, directory, shop-list (sort), shop-product (sizes), contact (topics), faq (side nav) |
| `accordion` | `items` [question, answer], `open` first\|all\|none | `<x-ui.accordion :items="$faq"/>` | 6 stage pages, index, contact, faq, directory-business |
| `alert-emergency` | `number` (default 115) + slot text | `<x-ui.alert-emergency :number="$emergency">…</x-ui.alert-emergency>` | 6 stage pages, services |
| `steps` | `items` [title, text], `tone` | `<x-ui.steps :items="$steps"/>` | directory-business, index («یک اپ، پنج بخش» uses `tone="dark"` or cards) |
| `stepper` | `steps` [label, hint?], `current` (1-based), `variant` list\|bar, `label`; slot = note (list) | `<x-ui.stepper variant="bar" :current="2" :steps="$s"/>` | directory-join (list), shop-checkout (bar) |
| `timeline` | `items` [title, time?, state done\|current\|todo], `label` | `<x-ui.timeline :items="$status"/>` | directory-join-done, shop-done |
| `success-hero` | `icon`, `tone` success\|lavender, `title`, `align`, `as`; slot = lead; `actions` slot | `<x-ui.success-hero title="رزروت ثبت شد">کد پیگیری …</x-ui.success-hero>` | directory-booked, directory-join-done, shop-done |
| `rating` | `value`, `count`, `size` xs\|sm\|md, `stars` | `<x-ui.rating :value="4.8" :count="126"/>` | shop*, directory* |
| `price` | `amount` (int toman), `compare`, `from`, `unit`, `size` md\|lg\|inline, `currency` | `<x-ui.price :amount="320000" from unit="هر جلسه" size="lg"/>` | shop*, directory, directory-place, directory-booked, directory-business |
| `toggle-row` | `title`, `text`, `variant` status\|switch, `checked`, `default-label`, `divided` | `<x-ui.toggle-row title="…" text="…" divided/>` | privacy (consents), shop (reminder) |
| `newsletter` | `title`, `text`, `action` (null = disabled), `placeholder`, `button`, `name`, `id`; slot | `<x-ui.newsletter :action="route('newsletter.store')"/>` | blog (L4-02 wires it) |
| `gallery` | `images` (MediaData\|ids), `variant` mosaic\|product, `alt`, `total`, `more-href` | `<x-ui.gallery :images="$photos" alt="…" :total="12" more-href="#photos"/>` | directory-place, shop-product (lightbox: L5-03/L6-03) |
| `store-badges` | `links` (AppLinksSettings or key⇒url array), `tone` dark\|light | `<x-ui.store-badges :links="$appLinks"/>` | footer (own markup, L1-02), every app CTA |
| `qr` | `url`, `label`, `size` | `<x-ui.qr :url="$appLinks->webApp"/>` | 13 app-CTA pages |
| `app-cta` | `title`, `lead`, `links`, `qr-url`, `variant` full\|aside\|strip, `icon`, `id` (default `download`), `heading-id` | `<x-ui.app-cta title="ریتمی را رایگان نصب کن" lead="…" :links="$appLinks" :qr-url="$qr"/>` | full: blog, cycle, faq, index, menopause, plus, postpartum, pregnancy, services, social-responsibility, teen, tools, ttc · aside: directory-booked, article · strip: shop-done |
| `promise-banner` | `href`, `eyebrow`, `title`, `lead` (HtmlString ok), `cta` | `<x-ui.promise-banner href="{{ route('social-responsibility') }}"/>` | index, about |
| `promo-split` | `items` [href, tone lavender\|blush\|dashed, eyebrow?, title, text?, cta?, illustration? \| icon?], `level` | see file header | shop (illustration layout), services (icon layout) |
| `page-intro` | `eyebrow`, `title`, `lead`, `size` lg\|md\|sm, `plain`, `as`; slot (search, chips) | `<x-ui.page-intro eyebrow="مجله ریتمی" title="…" lead="…"><x-ui.chip-nav …/></x-ui.page-intro>` | blog, contact, faq, plus, services, tools, shop (`plain`), directory |

### Forms (`x-ui.form.*`)

| Component | Key props | Pages |
|---|---|---|
| `field` | `for`, `label`, `hint`, `error`, `required`, `as` div\|fieldset (legend) | contact, tools, directory-join, directory-place, shop-checkout, faq, blog |
| `input` | `type`, `invalid`, `described` (+ any attribute: id, name, value, autocomplete) | same |
| `textarea` | `invalid`, `described`, slot = value | contact, directory-join |
| `select` | `options` value⇒label, `selected`, `placeholder`, `invalid` | directory-join, shop-checkout |
| `radio-card` | `name`, `value`, `title`, `text`, `icon`, `checked`, `disabled`, `type` radio\|checkbox | shop-checkout (payment, delivery), directory-place (day/time), shop-product (size) |
| `checkbox` | `checked` + attributes; slot = label | directory-join, shop-checkout |

Design "inputs" that are `<span>`s become real controls with these (AUDIT §6).

## Cards (`x-cards.*`)

| Component | Key props | Pages |
|---|---|---|
| `stage` | `href`, `icon`, `color`, `title`, `text`, `cta`, `as` | index (6) |
| `feature` | `href?`, `icon`, `color`, `title`, `text`, `where`, `as` | 6 stage pages, index tools, tools guides |
| `service` | `href`, `icon`, `color`, `title`, `text`, `cta`, `as` | 5 stage pages, index, services |
| `value` | `icon`, `color`, `title`, `text`, `variant` tile\|inline\|compact, `as` | about, privacy, social-responsibility, shop trust strip, plus, index privacy grid |
| `article` | `post` (PostCardData) or `href`, `title`, `stage`, `label`, `minutes`, `media`, `mobile-media`, `excerpt`, `reviewer`, `featured`, `as` | index, blog (+ featured), article related, 6 stage pages |
| `product` | `href`, `title`, `seller`, `rating`, `reviews`, `price`, `compare`, `badge`, `media` \| `illustration`, `tint`, `wishlist`, `as` | shop, shop-list, shop-product, shop-cart (data: L6-02) |
| `place` | `href`, `name`, `rating`, `reviews`, `category`, `district`, `distance`, `ages`, `price-from`, `slots`, `verified`, `media` \| `illustration`, `selected`, `bookmark`, `as` | directory, directory-business, directory-join (data: L5-02) |
| `review` | `name`, `rating`, `meta`, `text`, `avatar` | shop-product, directory-place |

Product and place cards are `<article>`s with a stretched title link; the wishlist/bookmark buttons sit outside the
link (no nested interactive content) and are inert until L6/L5 wire them.

## Stage blocks (`x-stage.*`)

| Component | Key props | Pages |
|---|---|---|
| `tools-block` «کارهای کوچک، آمادگی بیشتر» | `items` (feature-card props), `eyebrow`, `title`, `id`, `bg` | 6 stage pages |
| `help-block` «وقتی کمک بیشتری لازم داری» | `items` (service-card props), `eyebrow`, `title`, `id`, `bg` (default surface) | cycle, ttc, pregnancy, postpartum, menopause, index («وقتی به کمک بیشتری نیاز داری», `:eyebrow="null"`) |
| `readings` «برای همین مرحله» | `posts` (list<PostCardData> or article props), `more-href`, `more-label`, `eyebrow`, `title`, `id`, `bg`; renders nothing when empty | 6 stage pages, index («خواندنی‌های این هفته», `more-label="همه مقاله‌ها"`) |

## Not here (owned by page tasks)

`x-layout.stage-nav`, `x-stage.hero`, `x-stage.feature-split`, `x-stage.faq`, `x-mock.*` (L3-03), `x-ui.warn`,
`x-cards.checklist` (L3-07), `x-ui.quote`, `x-cards.team` (L3-08), `x-faq` (L3-09), `x-cards.pricing` (L3-06),
`x-shop.subnav`, `x-cards.category-tile`, `x-ui.qty` (L6), `x-directory.map-panel` (L5-02).

## Known gaps / follow-ups

- `fa_digits()` and `App\Support\Jalali` / `Money` (task scope) live outside this task's `touches`; components use
  `App\View\Components\Layout\Footer::persianDigits()` and an in-component toman formatter for now. Swap both when the
  helpers land; dates are display strings until then.
- `x-ui.qr` and `x-ui.store-badges` keep their (small) logic in Blade because `app/View/Components` is outside
  `touches`; moving them to class components is a mechanical follow-up.
