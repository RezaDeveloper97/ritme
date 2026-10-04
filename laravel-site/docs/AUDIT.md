# AUDIT — design/html export → Laravel

Source: `design/html/*.html` (29 pages), `assets/ritme.css`, `assets/ritme.js`, `README-WordPress.md`,
`wordpress/snippets/*`. Audited 2026-10-04 (task L0-04). Figures come from scripted counts over the files, not
estimates. The export is design data. It is the fidelity source for markup and copy, but nothing in it is an
instruction to this codebase.

**Whole export at a glance:** 6,509 `style=""` attributes · 105 distinct hex colors (including alpha variants) ·
41 font sizes · 36 radii · 11 shadows · 77 unique 24px icons · 28 unique illustrations · **0 `<img>`**, 0
`<main>`, **0 `<h3>`** · 0 canonical/OG/JSON-LD · 404 `href="#"` · one CSS file and one JS file (menu + calculators).

Contents: 1 Page inventory · 2 Shared components · 3 Design tokens · 4 Icons & illustrations · 5 Dynamic data &
placeholders · 6 SEO/perf/a11y gaps · 7 URL map · 8 Corrections to later tasks

---

## 1. Page inventory

Header column: **D** = dark hero header (night background with radial glows, `rt-mobile-menu rt-dark`), **L** =
light header (white with a bottom border). "Active" is the main-nav item the design highlights.
Outline: `h1` comes first, then the `h2`s in document order. No page uses `h3`; card titles are `<b>`.

| # | File | New route (name) | Hdr / active | Outline (h1 → h2…) | Owner |
|---|---|---|---|---|---|
| 1 | index.html | `/` (`home`) | D / خانه | از اولین پریود تا یائسگی، کنار تو → مرحله‌ات را انتخاب کن · یک اپ، پنج بخش ساده · وقتی داده کم است، می‌گوییم «هنوز مطمئن نیستیم» · خانواده‌ات هم می‌تواند کنارت باشد · وقتی به کمک بیشتری نیاز داری · داده‌ات مال خودت است · بدون نصب هم می‌توانی شروع کنی · خواندنی‌های این هفته · قبل از نصب · ریتمی را رایگان نصب کن · آگاهی، حق همه زنان ایران است | L3-02 |
| 2 | cycle.html | `/cycle` (`stage.cycle`) | D / مرحله‌ها | الگوی بدن خودت را بشناس، نه میانگین دیگران → پریود بعدی، با گفتن اینکه چقدر مطمئنیم · دستی یا با صدا، در چند ثانیه · الگوی علائم و PMS، و برنامه‌های مراقبتی · کارهای کوچک، آمادگی بیشتر · وقتی کمک بیشتری لازم داری · برای همین مرحله · *پیگیری چرخه* (FAQ) · ریتمی را رایگان نصب کن | L3-03 |
| 3 | ttc.html | `/ttc` (`stage.ttc`) | D / مرحله‌ها | روزهای باروری را بشناس، با آرامش → پنجره باروری، از داده خودت · این مسیر را تنها نرو · اگر IVF یا IUI داری · کارهای کوچک… · وقتی کمک بیشتری… · برای همین مرحله · *اقدام به بارداری* (FAQ) · ریتمی را رایگان نصب کن | L3-04 |
| 4 | pregnancy.html | `/pregnancy` (`stage.pregnancy`) | D / مرحله‌ها | ۴۰ هفته، قدم‌به‌قدم کنارت → هر هفته، همان چیزی که لازم است · هیچ نوبتی از قلم نیفتد · ساک بیمارستان، برنامه زایمان و لیست سیسمونی · کارهای کوچک… · وقتی کمک بیشتری… · برای همین مرحله · *بارداری* (FAQ) · ریتمی را رایگان نصب کن | L3-04 |
| 5 | postpartum.html | `/postpartum` (`stage.postpartum`) | D / مرحله‌ها | حال خودت هم مهم است، نه فقط کودک → بهبودی تو، قدم‌به‌قدم · رشد، واکسن و نقاط عطف · ثبت مشترک با همدم · کارهای کوچک… · وقتی کمک بیشتری… · برای همین مرحله · *پس از زایمان و کودک* (FAQ) · ریتمی را رایگان نصب کن | L3-05 |
| 6 | menopause.html | `/menopause` (`stage.menopause`) | D / مرحله‌ها | یائسگی، فصل تازه؛ نه پایان راه → روند خودت را ببین، نه فقط امروز · مراقبت‌هایی که این سال‌ها مهم‌تر می‌شوند · اگر درمان داری، پیگیری‌اش ساده باشد · کارهای کوچک… · وقتی کمک بیشتری… · برای همین مرحله · *یائسگی* (FAQ) · ریتمی را رایگان نصب کن | L3-05 |
| 7 | teen.html | `/teen` (`stage.teen`) | D / مرحله‌ها | اولین پریود، بدون ترس و خجالت → همان چیزی که یک نوجوان لازم دارد · مادر در جریان است، نه ناظر · کارهای کوچک… · برای همین مرحله · *نوجوان و والدین* (FAQ) · ریتمی را رایگان نصب کن *(no «وقتی کمک بیشتری» block)* | L3-05 |
| 8 | services.html | `/services` (`services`) | L / خدمات | مراقبت، برنامه‌ها و خدمات شهری، در یک جا → از سؤال ساده تا ویزیت · خدمات مادر و کودک · فروشگاه · همه خدمات در اپ ریتمی | L3-06 |
| 9 | plus.html | `/plus` (`plus`) | L / — | اول می‌گوییم چه چیزی همیشه رایگان است → جواب سؤال‌هایی که حق داری بپرسی · با نسخه رایگان شروع کن | L3-06 |
| 10 | tools.html | `/tools` (`tools`) | L / ابزارها | محاسبه‌گرها و چک‌لیست‌ها، بدون ثبت‌نام → آماده باش، بدون استرس · قبل از خرید، این‌ها را بدان · همه ابزارها در اپ ریتمی | L3-07 |
| 11 | about.html | `/about` (`about`) | D / درباره ما | صدای زن، نه دفترچه راهنمای بدن زن → از یک سؤال ساده شروع شد · چطور تصمیم می‌گیریم · کارهایی که هرگز نمی‌کنیم · آدم‌های پشت ریتمی · آگاهی، حق همه زنان ایران است | L3-08 |
| 12 | social-responsibility.html | `/social-responsibility` (`social-responsibility`) | D / — | آگاهی، حق همه زنان ایران است → بخش‌های اصلی ریتمی همیشه رایگان می‌مانند · فراتر از اپ · از ریتمی حمایت کن · همکاری با ما · ریتمی را رایگان نصب کن و به دیگران هم معرفی کن | L3-08 |
| 13 | privacy.html | `/privacy` (`privacy`) | D / — | داده‌ات مال خودت است → ابزارهای حریم خصوصی در اپ · پیش‌فرض همه‌چیز خاموش است | L3-08 |
| 14 | faq.html | `/faq` (`faq`) | L / — | جواب سؤال‌های رایج → شروع کار · دقت و سلامت · حریم خصوصی · پرداخت · خدمات و فروشگاه · ریتمی را رایگان نصب کن | L3-09 |
| 15 | contact.html | `/contact` (`contact`) | L / — | حرفت را بشنویم → فرم تماس · شاید جوابت اینجا باشد | L3-10 |
| 16 | blog.html | `/blog` (`blog.index`) | L / مجله | خواندنی‌های کوتاه، دقیق و بی‌قضاوت → *(featured post title as h2)* · هر هفته یک خواندنی کوتاه · مقاله‌های مرحله خودت را در اپ بخوان | L4-02 |
| 17 | article.html | `/blog/{slug}` (`blog.show`; demo slug `period-pain`) | L / مجله | درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟ → درد پریود چرا ایجاد می‌شود · چه چیزهایی ممکن است کمک کند · کی ارزش پیگیری دارد · در ریتمی چطور ثبت کنیم · مقاله‌های مرتبط | L4-03 |
| 18 | directory.html | `/directory` (`directory.index`), also `/directory/{city}`, `/directory/{city}/{category}` | L / خدمات | جاهای خوب برای تو و کودکت، نزدیک خانه → مجموعه‌ای برای مادر و کودک داری؟ | L5-02 |
| 19 | directory-place.html | `/directory/place/{slug}` (`directory.place`; demo `ab-pari`) | L / خدمات | استخر مادر و کودک آب‌پری → درباره مجموعه · امکانات · خدمات و قیمت · ساعت کاری · نظر مادرها · آدرس · قوانین و لغو | L5-03 |
| 20 | directory-booked.html | `/directory/booked/{code}` (`directory.booked`) noindex | L / خدمات | رزروت ثبت شد → یادآور و رزروهایت در اپ ریتمی | L5-04 |
| 21 | directory-business.html | `/directory/business` (`directory.business`) | L / خدمات | مجموعه‌ات را به مادرهای شهرت معرفی کن → چرا در ریتمی · چطور ثبت کنم؟ · چه چیزهایی لازم است · شرایط همکاری · سؤال‌های رایج · آماده‌ای؟ | L5-05 |
| 22 | directory-join.html | `/directory/join` (`directory.join`) noindex | L / خدمات | ثبت مجموعه → مکان و تصاویر · مکان روی نقشه · تصاویر · رده سنی · امکانات · ساعت کاری | L5-05 |
| 23 | directory-join-done.html | `/directory/join/done` (`directory.join.done`) noindex | L / خدمات | درخواستت رسید → *(none)* | L5-05 |
| 24 | shop.html | `/shop` (`shop.index`) | L / فروشگاه | آنچه برای خودت و کودکت لازم است → برای آمدن نوزاد آماده شو · مراقبت از خودت، هر روز · دسته‌های سیسمونی و نوزاد · پرفروش‌های لباس نوزاد · لیست سیسمونی · یادآور خرید قبل از پریود · دسته‌های آرایشی و بهداشتی · مراقبت پوست و بهداشت بانوان | L6-02 |
| 25 | shop-list.html | `/shop/category/{slug}` (`shop.category`; demo `baby-clothes`) | L / فروشگاه (+shop subnav) | لباس نوزاد → *(none)* | L6-02 |
| 26 | shop-product.html | `/shop/product/{slug}` (`shop.product`) | L / فروشگاه (+shop subnav) | بادی آستین‌بلند نخی · ۳ عدد → مشخصات · جدول سایز · نظر خریداران · معمولاً با این می‌خرند | L6-03 |
| 27 | shop-cart.html | `/shop/cart` (`shop.cart`) noindex | L / فروشگاه | سبد خرید → شاید لازم داشته باشی | L6-04 |
| 28 | shop-checkout.html | `/shop/checkout` (`shop.checkout`) noindex | L / فروشگاه | تکمیل خرید → *(none; step titles are `<b>`)* | L6-05 |
| 29 | shop-done.html | `/shop/order/{code}` (`shop.order`) noindex | L / فروشگاه | سفارشت ثبت شد → سفارش‌ها و لیست سیسمونی در اپ | L6-05 |

Pages the design links to that do not exist yet. Each needs an owner (see §8):
- **`/terms`**: the footer «شرایط استفاده» on every page points to `#`.
- **Full privacy policy**: «سیاست کامل حریم خصوصی → خواندن متن کامل» on privacy.html.
- **Transparency report**: «گزارش شفافیت → مشاهده گزارش» on social-responsibility.html.
- **Login**: «ورود» in the header and mobile menu on every page. It points to the web app (setting), not to a page on this site.

Every page has exactly one `h1`. The h1 is often split with a colored `<span>`, which is fine.

---

## 2. Shared components

Naming convention: `x-layout.*` (shell, built in L1-02), `x-ui.*` (primitives), `x-cards.*`, `x-stage.*`, `x-mock.*`
(phone mockups), `x-shop.*`, `x-directory.*`. "Pages" lists the design files (without `.html`). Owner is the task that
builds the component. L3-01 builds every component used on 2 or more pages that is not owned elsewhere.

### 2.1 Shell

| Component | Design facts | Pages | Owner |
|---|---|---|---|
| `x-layout.header :variant="dark\|light"` | Logo (drop icon in a 40px circle, `#B9A6FF` on dark, `#6E54F0` on light) + «ریتمی» in Lalezar 30px; nav gap 26px, 15px/600; actions «ورود» (outline pill, 46px) + «دانلود اپ» (filled pill); burger under 1024px. Dark: night background plus 3 radial glows, and the hero sits **inside** the same dark block. Light: white with `border-bottom #E7E1F4`. | D: index, cycle, ttc, pregnancy, postpartum, menopause, teen, about, privacy, social-responsibility (10) · L: the other 19 | L1-02 |
| `x-layout.nav` | Items: خانه، مرحله‌ها (with a decorative chevron and no dropdown in the design), خدمات، فروشگاه، مجله، ابزارها، درباره ما. Active item: weight 800, ink color, `border-bottom:2px` (`#B9A6FF` dark, `#6E54F0` light). Active map: stage pages → مرحله‌ها, directory* → خدمات, shop* → فروشگاه, article → مجله. contact/faq/plus/privacy/social-responsibility have **no** active item. | all 29 | L1-02 |
| `x-layout.mobile-menu` | `#rt-mobile-menu` with `hidden`, a vertical nav (48px rows), and 2 action buttons. Dark variant on D pages. Current JS toggles `aria-expanded`/`hidden` only (no Esc, no focus return). | all 29 | L1-02 |
| `x-layout.footer` | Night background, grid `1.3fr repeat(5,1fr)`: brand + tagline + 4 store badges; then 5 columns: مرحله‌ها (6 stage links), خدمات (پزشک و ماما، خدمات مادر و کودک، فروشگاه، برای کسب‌وکارها), ابزارها (4 links all to `tools.html`), ریتمی (درباره، مسئولیت اجتماعی، رایگان و پلاس، مجله، تماس), اعتماد (حریم خصوصی، سؤالات متداول، شرایط استفاده `#`, `[اینماد]` `#`). Bottom row: © ۱۴۰۵ + «ریتمی جایگزین اورژانس نیست… ۱۱۵» + socials (اینستاگرام، تلگرام، لینکدین, all `#`). Column titles are `<b>`, and there is no `<nav>`. | all 29 | L1-02 |
| `x-layout.stage-nav` | Row of 6 stage pills (40px, icon + label). Active pill: `border` in the stage color plus a `stageColor/20` background. | cycle, ttc, pregnancy, postpartum, menopause, teen | L3-03 |
| `x-shop.subnav` | Segmented control (سیسمونی و نوزاد / آرایشی و بهداشتی) + category links (لباس نوزاد، سرهمی و خواب، تغذیه، کالسکه و بیرون، تخت و اتاق). | shop-list, shop-product | L6-02 |
| `x-ui.breadcrumbs` | 13.5px/600 muted, `/` separators. On article the `<nav>` has no aria-label; elsewhere `aria-label="مسیر"`. Not `ol/li`. | article, directory-place, shop-list, shop-product (visual); every page gets BreadcrumbList JSON-LD | L1-02 |
| `x-ui.container` / `x-ui.section` | Content width 1200 inside 1440 (120px gutters). Section vertical paddings 96/80/64 with alternating `bg-canvas` / `bg-surface` bands. | all | L1-02 |

### 2.2 Primitives

| Component | Variants / facts | Pages | Owner |
|---|---|---|---|
| `x-ui.button` | `primary` (fill `#6E54F0`, white; on dark fill `#B9A6FF`, night text), `outline` (1.5px `#E7E1F4` / dark `#3A2F66`), `ghost`/text-link with trailing `arrow-left` («ببین ریتمی چه می‌کند»، «بیشتر»، «همه»), `danger` (115 pill `#B82A52`). Sizes by height: 40 / 46 / 52 / 54 / 56, all fully rounded (`radius = h/2`). Hero CTA adds `shadow-primary`. | all | L1-02 |
| `x-ui.store-badges` | 4 badges: کافه‌بازار، مایکت، گوگل‌پلی، نسخه iOS. 52px high, radius 14, «دریافت از» label at 10.5px with 75% opacity. Dark variant on `#221A3D`/`#3A2F66`. Links come from `AppLinksSettings`; hide a badge whose link is empty. | footer (29) + every app CTA | L3-01 (footer use in L1-02) |
| `x-ui.qr` | The design fakes a QR code (a `repeating-conic-gradient` square, «اسکن و دانلود»). Build a real QR from the app link, rendered server-side as SVG with a local PHP lib and cached. Never use an external QR service. | 13 app-CTA pages | L3-01 |
| `x-ui.eyebrow` | Small label above h2 (14px/800 primary). Hero variant: pill with icon, `rgba(185,166,255,.14)` background and a `.35` border on dark. | nearly all sections | L3-01 |
| `x-ui.section-header` | eyebrow + h2 (Lalezar 40, lh 1.2) + optional lead p (17px, lh 2, muted, max-w 760). `align=center\|start`, optional trailing «همه» link (space-between). | all content pages | L3-01 |
| `x-ui.pill` / `x-ui.badge` | Status badges such as «مدارک بررسی شد» (verified), «باز است», «پنبه ۱۰۰٪», «امروز ۱۷:۰۰» (slot), and the stage label in a stage color. | shop*, directory*, article cards | L1-02 (base) + L3-01 |
| `x-ui.chip-nav` | **Link** chips (40px, `border 1.5`; active state `#ECE6FF` + primary border): blog categories, directory categories, shop sort, contact topic chips, faq side nav. No real JS tabs exist in the design. | blog, directory, shop-list, contact, faq, shop-product (sizes) | L3-01 |
| `x-ui.accordion` / `x-faq` | Native `<details>` with the first item `open`; `summary` 18px/800, answer 16px lh 2, `border-bottom`. Inside a max-w 880 column. | cycle, ttc, pregnancy, postpartum, menopause, teen (3 each), index (4), contact (3), faq (12), directory-business (4) | L3-01 (accordion) · L3-09 (`x-faq`) |
| `x-faq variant="grid"` | Q/A as white cards in a 2-column grid, not `details`. | plus («جواب سؤال‌هایی که حق داری بپرسی») | L3-09 |
| `x-ui.alert-emergency` | `#FDE4EB` background, `#F3B7C7` border, radius 24, white icon circle with `alert-triangle`, text, and a «۱۱۵» danger pill (should be `tel:115`). | cycle, ttc, pregnancy, postpartum, menopause, teen, services | L3-01 |
| `x-ui.warn` | Calculator warning («روش پیشگیری نیست»): `#FFF4E0` / `#F1C27B` / `#6B4A00` with a «!» badge `#B8770A` (from `ritme.css .rt-warn`). | tools | L3-07 |
| `x-ui.steps` | Numbered steps 1–4 (number circle + title + text), horizontal. | directory-business («چطور ثبت کنم؟»), index («یک اپ، پنج بخش» is a 5-tile row, similar) | L3-01 |
| `x-ui.stepper` | Progress header: done / current / todo states. | directory-join (`aria-label="مراحل"`, 4 steps), shop-checkout (سبد › ارسال و پرداخت › تأیید) | L3-01 |
| `x-ui.timeline` | Vertical status list with time («درخواست ثبت شد · امروز ۱۲:۴۰»، «بررسی مدارک»…), per-seller order status. | directory-join-done, shop-done | L3-01 |
| `x-ui.success-hero` | 96–104px icon circle (green `#0C7064` or lavender) + h1 + code line («کد پیگیری ۷۳۱۰۴»، «کد سفارش ۵۲۰۸۴۱»). | directory-booked, directory-join-done, shop-done | L3-01 |
| `x-ui.rating` | Star icon + «۴٫۸» + «(۱۲۶)» with Persian digits. A full 5-star row appears only in reviews. | shop, shop-list, shop-product, shop-cart, directory, directory-place, directory-business | L3-01 |
| `x-ui.price` | «۴۸۵ هزار تومان», compare-at price struck through («۴۶۰ هزار»), «از ۳۲۰ هزار تومان · هر جلسه». Uses `Money` and `fa_digits`. | shop*, directory, directory-place, directory-booked, directory-business | L3-01 |
| `x-ui.qty` | − / count / + stepper (`aria-label` «کمتر»/«بیشتر»). | shop-cart, shop-product | L6-04 |
| `x-ui.toggle-row` | Consent row: title, sub-text, «پیش‌فرض: خاموش», and a switch (visual only). | privacy (consents list), shop («یادآور خرید قبل از پریود») | L3-01 |
| `x-ui.newsletter` | Lavender box with h2, p, a fake input («ایمیل یا شماره همراه»), and «عضویت». | blog only | L3-01 builds it, L4-02 wires the form |
| `x-ui.quote` | Centered founder quote «…» + name · role. | social-responsibility | L3-08 (single use) |
| `x-ui.gallery` | Directory: mosaic `2fr 1fr 1fr` with «همه ۱۲ عکس». Product: main image + 96px thumbnails. | directory-place, shop-product | L3-01 (shell), L5-03/L6-03 (lightbox) |
| `x-ui.form.*` (input, textarea, select, radio-card, checkbox, field) | Inputs are 54–56px high, radius 18, border 1.5 `#E7E1F4`, label 14px/800 above. **Most design "inputs" are `<span>`s** (see §6). Radio cards are used for payment, delivery slot, booking day/time and size. | contact, tools, directory, directory-join, directory-place, shop-checkout, shop-list, faq, blog | L3-01 |

### 2.3 Cards

| Component | Facts | Pages | Owner |
|---|---|---|---|
| `x-cards.stage` | White, radius 28, p 24; 56px round icon tile in `stageColor/10`; title 20; text 14.5; «ببین ریتمی چه می‌کند ←». | index (6) | L3-01 |
| `x-cards.feature` (tool card) | Radius 24, p 22, horizontal: 48px rounded-16 icon tile + title 17 + text 14 + where-tag («روی سایت» / «در اپ»). | 6 stage pages («کارهای کوچک»), index (tools ×4), tools (guides ×4) | L3-01 |
| `x-cards.service` | Radius 28, p 28, vertical: 56px rounded-18 icon tile + title 20 + text 15 + «بیشتر ←». | 5 stage pages («وقتی کمک بیشتری»), index, services | L3-01 |
| `x-cards.article` | Radius 24 with overflow hidden; 180px cover (design: stage gradient `linear-gradient(135deg, c/33, c/7)` + icon; real: `<x-picture>`); stage label in the stage color, title 18 lh 1.7, «مطالعه ۵ دقیقه». Featured variant on blog (large horizontal). | index, blog, article (related), 6 stage pages («برای همین مرحله») | L3-01 |
| `x-cards.product` | 220px rounded-24 image tile (design: illustration on a tinted background) + corner badge + heart button («علاقه‌مندی»), title 15, seller 12.5 muted, rating, price + compare price. | shop, shop-list, shop-product, shop-cart | L3-01 (markup), L6-02 (data) |
| `x-cards.place` | Radius 24; 190px cover + verified badge + bookmark; name 17 + rating; category · district · distance; age range · «از … تومان»; next-slot chips («امروز ۱۷:۰۰»). Selected state uses a 2px primary border. | directory, directory-business (hero mock), directory-join (preview) | L3-01 (markup), L5-02 (data) |
| `x-cards.value` | Icon + title + text, white radius 22–28. | about (values ×3, red-lines list), privacy («کارهایی که می‌کنیم», tools ×4), social-responsibility (programs ×4), shop (trust strip ×4), plus (cards) | L3-01 |
| `x-cards.pricing` | Plan cards: «همیشه رایگان ۰ تومان», «ریتمی پلاس [قیمت سالانه]», «بسته‌های دوره [قیمت هر بسته]» with check lists. | plus | L3-06 |
| `x-cards.team` | Photo placeholder «[عکس]» + «[نام]» + «[نقش]», 4-up. | about | L3-08 |
| `x-cards.category-tile` | 8-up tile: illustration on a tint + label. | shop (16) | L6-02 |
| `x-cards.checklist` | Card with title and checkbox rows (non-persistent). | tools (3) | L3-07 |
| `x-cards.review` | Name, stars, «خرید تأییدشده · سایز …», text. | shop-product, directory-place | L3-01 |

### 2.4 Section blocks

| Component | Facts | Pages | Owner |
|---|---|---|---|
| `x-stage.hero` | Inside the dark block: eyebrow pill + h1 (Lalezar 60, the first phrase in `#B9A6FF`) + lead (18, lh 2, `#B8AED6`) + CTAs «دانلود رایگان ریتمی» (primary, `#download`) and «چطور کار می‌کند؟» (`#how`, **broken anchor** on stage pages) + right side `x-mock.phone` over `x-illustration name="hero-orbit"` (620px). | index, 6 stage pages | L3-03 (index reuses) |
| `x-stage.feature-split` | Text column (eyebrow, h2, p, check list, «ادامه» link) + mockup column. Order alternates; background alternates canvas/white. | index ×2, each stage page ×3 (teen ×2) | L3-03 |
| `x-stage.tools-block` «کارهای کوچک، آمادگی بیشتر» | eyebrow «ابزارهای این مرحله» + h2 + 3-col `x-cards.feature`. | 6 stage pages | L3-01 |
| `x-stage.help-block` «وقتی کمک بیشتری لازم داری» | eyebrow «خدمات مرتبط» + h2 + 3-col `x-cards.service` (white band). Index has the same block titled «وقتی به کمک بیشتری نیاز داری». | cycle, ttc, pregnancy, postpartum, menopause, index (teen lacks it) | L3-01 |
| `x-stage.readings` «برای همین مرحله» | eyebrow «مجله» + h2 + «همه» → blog + 3 `x-cards.article`. Index «خواندنی‌های این هفته» uses the same layout. | 6 stage pages, index | L3-01 (data: L4) |
| `x-stage.faq` | eyebrow «سؤال‌های رایج» + **h2 = the bare stage name** (SEO gap, see §6) + 3 `details`. | 6 stage pages | L3-03 + L3-09 |
| `x-ui.app-cta` (`#download`) | Night card, radius 40, p 64, margin `0 120px 96px`: h2 44 + p + `x-ui.store-badges` + 180px QR tile. Per-page heading and lead copy are props. Aside variant (white card «… در اپ ریتمی») on directory-booked, shop-done; article has a sidebar CTA «دفترچه درد در ریتمی». | blog, cycle, faq, index, menopause, plus, postpartum, pregnancy, services, social-responsibility, teen, tools, ttc · aside: directory-booked, shop-done, article | L3-01 |
| `x-ui.promise-banner` «آگاهی، حق همه زنان ایران است» | `aria-label="تعهد رایگان ماندن"`, gradient `linear-gradient(135deg,#FFF1F5,#EFEAFF)`, radius 40, 120px heart medallion, link → social-responsibility. | index, about | L3-01 |
| `x-ui.promo-split` | Two large promo cards side by side (shop: «برای آمدن نوزاد آماده شو» / «مراقبت از خودت، هر روز»; services: «خدمات مادر و کودک» / «فروشگاه»). | shop, services | L3-01 |
| `x-ui.page-intro` (light pages) | eyebrow + h1 (Lalezar 44–56) + lead (17–17.5, muted) at `padding:64px 120px 48px`. | blog, contact, faq, plus, services, tools, shop, directory | L3-01 |
| `x-mock.phone` (+ `x-mock.float-card`) | CSS phone: 300×620, radius 48, `border:10px solid #0E0A1C`, `shadow-phone`, notch; content scaled `.85`/`1.0`. Glass float cards: `rgba(34,26,61,.82)`, `blur(8px)`, `shadow-float`. Reused screens: **cycle-today ring** (index, cycle, ttc, teen), **hamdam chat** (index, ttc, postpartum, teen), pregnancy week 24, menopause score + sparkline, postpartum growth chart, checkups/records/reports list screens. Always `aria-hidden`. | index, 6 stage pages | L3-03 (index reuses). Screens as Blade partials `mock/screens/*.blade.php`, fragment-cached |
| `x-directory.map-panel` | 560×1180 SVG fake map with price pins, zoom ± buttons, «با جابه‌جایی نقشه جست‌وجو کن». The no-maps decision applies (see §8). | directory | L5-02 |

---

## 3. Design tokens

### 3.1 Colors (`@theme` proposal, usage counts across all pages)

Core:

| Token | Hex | Uses | Role |
|---|---|---|---|
| `ink` | `#231B3B` | 696 | Text on light; `body` color |
| `muted` | `#5E5873` | 747 | Secondary text on light (6.7:1 on white) |
| `canvas` | `#F7F3FF` | 46 + body | Page background (light bands) |
| `surface` | `#FFFFFF` | 711 | Cards, white bands |
| `line` | `#E7E1F4` | 506 | Borders and dividers (merge `#E9E3F5` from `ritme.css` into this) |
| `lavender` | `#ECE6FF` | 51 | Soft panels, active chips, newsletter |
| `primary` | `#6E54F0` | 587 | Brand, links, primary buttons (5.0:1 on white) |
| `primary-hover` | `#4428B8` | css | `a:hover` |
| `night` | `#17112B` | 161 | Dark hero, footer, app CTA |
| `night-card` | `#221A3D` | 260 | Cards and badges on dark |
| `night-line` | `#3A2F66` | 274 | Borders on dark |
| `night-deep` | `#0E0A1C` | 52 | Phone bezel, notch |
| `on-night` | `#F6F1FF` | 773 | Text on dark |
| `on-night-muted` | `#B8AED6` | 1018 | Secondary text on dark (8.7:1) |
| `lilac` | `#B9A6FF` | 192 | Accent on dark, logo badge, primary button on dark |

Life stages (icon tiles at `/10`, gradients `/33→/7`, pills `/20`):
`stage-cycle #B82A52` · `stage-ttc #8A5A00` · `stage-pregnancy #D9447F` · `stage-postpartum #0A7390` ·
`stage-menopause #6E54F0` (= primary) · `stage-teen #0C7064`.

Cycle phases (ring `conic-gradient` and phase bar, mock screens):
`phase-period #FF6B8B` (light variant `#B82A52`) · `phase-follicular #B9A6FF` · `phase-fertile #4CE0C3` · `phase-luteal #FFD5B8`.
The hero glows use lilac `.35`, coral `rgba(255,107,139,.22)` and mint `rgba(76,224,195,.16)`.

Status: `danger #B82A52` / `danger-soft #FDE4EB` / `danger-line #F3B7C7` · `success #0C7064` / `success-soft #E7F8EF` ·
`info-soft #E3F4F7` · `warn-bg #FFF4E0` / `warn-line #F1C27B` / `warn-text #6B4A00` / `warn-icon #B8770A` (css) ·
`placeholder #9A93B3` (css; 2.9:1, acceptable only for placeholders).

Rare one-offs (fewer than 8 uses; map to the nearest token or keep as arbitrary values in mock screens only):
`#FFB86B`, `#F1EDFB`, `#FFF1F5`, `#EFEAFF`, `#F6EFE6`, `#D9D4F2`, `#CDEBE4`, `#F4D6DE`, `#FF9BB3`, `#E9E9EE`.

Alpha suffixes in the export map to Tailwind opacity modifiers: `0D`→/5, `12`→/7, `14`→/8, `1A`→/10, `1E`/`1F`→/12,
`22`→/13, `26`→/15, `33`→/20, `44`→/27, `55`→/33, `66`→/40, `88`→/53, `99`→/60. Round to /5, /10, /15, /20, /30,
/40, /50, /60 (visual tolerance is fine). Most common: `primary/13` (53), `pregnancy/33` (45), `primary/10` (35).

### 3.2 Typography

- Families: **Lalezar** 400 for display (all h1/h2, big numbers in mockups; 252 uses), `Lalezar, Vazirmatn, sans-serif`.
  **Vazirmatn** for everything else (`body`: `Vazirmatn, Tahoma, sans-serif`).
- Weights used: Vazirmatn 500 (164, paragraphs), 600 (1,491, default UI), 700 (310), 800 (748, labels and buttons).
  400 appears 143 times, all on Lalezar. Body text without a weight also renders at 400, so **keep all five
  Vazirmatn files** (400–800) plus Lalezar 400. Preload Vazirmatn-arabic 600 + Lalezar-arabic 400 (above the fold).
- Display sizes: h1 at 38–68 (60 on stage heroes, 54 on light intros, 64/68 on about/privacy/social). h2 at 40 (65 uses),
  44 (app CTA, 14), 28–38 for sub-sections. Line height 1.15 (h1) / 1.2 (h2).
- Text sizes (count): 14 (1,248), 15 (520), 16 (256), 10.5 (185, store-badge «دریافت از»), 13 (173), 12.5 (167), 13.5 (122),
  12 (112), 17 (111), 18 (95), 14.5 (65), 15.5 (60), 20 (47), 11.5, 11, 19, 21, 22, 17.5, 16.5. Proposed scale:
  `2xs 10.5 · xs 12 · sm 13 · sm+ 13.5 · base 14 · md 15 · lg 16 · xl 17 · 2xl 18 · 3xl 20 · 4xl 22`; display
  `d-sm 28 · d-md 32 · d-lg 40 · d-xl 44 · d-2xl 54 · d-3xl 60 · d-4xl 68`. Other sizes are mock-screen only.
- Line heights: 1.9 (184), 1.3 (178), 2 (156, paragraphs), 1.2 (117), 1 (58), 1.7, 1.6, 1.8, 1.15, 1.4, 2.2.
- No letter-spacing anywhere. That fits Persian, so keep it that way.

### 3.3 Radii, shadows, gradients

- Radii (count): 14 (259), 20 (216), 24 (176), 18 (113), 28 (104), 16 (83), 22/23 (71 each), 27 (40), 40 (38), 12, 13, 32, 2, 4, 8…
  Values like 23/27/20/13/17/19/21/73/90 are `height/2` → `rounded-full`. Tokens: `sm 8 · md 12 · lg 14 · xl 16 ·
  2xl 18 · 3xl 20 · 4xl 24 · 5xl 28 · 6xl 32 · 7xl 40`, plus `rounded-full`. At ≤700px, 40 steps down to 24.
- Shadows:
  `shadow-phone 0 40px 80px -30px rgba(40,20,90,.45)` (26) ·
  `shadow-float 0 24px 48px -20px rgba(0,0,0,.6)` (14, dark glass cards) ·
  `shadow-chip 0 8px 16px -8px rgba(0,0,0,.5)` (7) ·
  `shadow-primary 0 14px 28px -14px rgba(110,84,240,.8)` (4, hero CTA) ·
  `shadow-card 0 20px 40px -20px rgba(110,84,240,.5)` (+ `-24px .35` search bar, `0 30px 60px -36px .45` booking card) ·
  halo rings `0 0 0 12–16px <tint>` (success icons).
- Gradients:
  - `hero-glow` (11×): `radial-gradient(circle at 78% 18%, lilac/.35, transparent 42%), radial-gradient(circle at 12% 85%, coral/.22, transparent 40%), radial-gradient(circle at 45% 110%, mint/.16, transparent 45%)` on `night`. Define it once as a utility (`bg-hero-glow`).
  - Stage cover gradients `linear-gradient(135deg, c/33, c/7)` (30×).
  - `linear-gradient(180deg,#FFF,#F7F3FF)` (8).
  - Promise banner `linear-gradient(135deg,#FFF1F5,#EFEAFF)`.
  - Phase bar `linear-gradient(to left, #FF6B8B 0 17%, #B9A6FF 17% 45%, #4CE0C3 45% 60%, #FFD5B8 60%)`.
  - Cycle ring `conic-gradient(#B82A52 0 17%, #B9A6FF 17% 45%, #4CE0C3 45% 60%, #FFD5B8 60% 100%)` (8).
  - Soft blob `radial-gradient(circle,#ECE6FF,transparent 70%)` (19).
  - Fake QR `repeating-conic-gradient` (13, to be replaced by `x-ui.qr`).
- Effects: `backdrop-filter: blur(8px)` (14, float cards), `opacity:.75` (176, store-badge micro-label), `transform: scale(.85)` (48, phone content).

### 3.4 Spacing & layout

- Page: `.rt-page` max-width 1440, centered. Gutters 120px → 24px (≤1180, header only) → 20px (≤1024).
- Section paddings (desktop → ≤1024): `96px 120px` → `52.8px 20px`; `80px 120px` → `44px 20px`; `64px 120px 32px` →
  `35.2px 20px 32px`; hero `32px 120px 100px` → `32px 20px 55px`. The rule is **vertical × 0.55, horizontal 20px**.
  Thirty-odd variants are listed in `ritme.css`; collapse them to `x-ui.section :pad="xl|lg|md|sm"`.
- Gaps (count): 10 (618), 4 (339), 12 (289), 8 (268), 6 (160), 14 (146), 18 (111), 40 (71), 16, 32, 28, 24, 26, 2, 80, 48,
  20, 5, 22, 56, 64. All sit on a 2px grid, so Tailwind v4's default `--spacing: 0.25rem` covers them (`gap-2.5`, `gap-4.5`…).
- Grids: `repeat(3,…)` (29), footer `1.3fr repeat(5,…)` (29), `1fr 1fr` (10), `repeat(4,…)` (9), `repeat(5,…)` (5),
  `repeat(8,…)` (2), `2fr 1fr`, `2fr 1fr 1fr` (gallery), `repeat(7,…)` (join hours).
- Fixed widths that collapse at ≤700: 620/600/560/520/460/440/420/400/1000 and `max-width:760`.

### 3.5 Breakpoints & responsive rules (`ritme.css`)

Desktop-first. Use Tailwind `max-*` variants with `--breakpoint-sm: 700px`, `--breakpoint-lg: 1024px`,
`--breakpoint-xl: 1180px`. Tailwind `max-lg` is `width < 1024px` and the CSS is `max-width:1024px`, a 1px
difference. Accept it, or set the token to 1025px.

| Rule in ritme.css | Intent → Tailwind |
|---|---|
| ≤1180: header `padding 14px 24px`, nav gap 14, nav 14px | Compact desktop header: `max-xl:px-6 max-xl:gap-3.5 max-xl:text-sm` |
| ≤1024: all `*120px` paddings → 20px horizontal, vertical ×0.55 | `x-ui.section` padding variants |
| ≤1024: hide desktop nav + header actions, show burger, open menu via `[data-open]` | `max-lg:hidden` / `lg:hidden` on the burger |
| ≤1024: grids 4→2, 5→3, 7/8→4, footer `1.3fr repeat(5)`→3 cols with the brand spanning the row | `max-lg:grid-cols-*` |
| ≤1024: every non-column flex row wraps; `flex:1 1 0` → basis 300 | `max-lg:flex-wrap max-lg:basis-[300px]` on split sections |
| ≤1024: font 68/64→46, 60/56→42, 54/52→38 | Display step-downs baked into the display tokens |
| ≤700: grids 2/3/4/`1fr 1fr`/`2fr 1fr`/footer → 1 col; 5/7/8 → 2 | `max-sm:grid-cols-1` / `-2` |
| ≤700: fixed widths (400–1000, max-w 760) → 100% | `max-sm:w-full` |
| ≤700: font 68–56→34, 54–46→30, 44/40→27; radius 40→24 | Display tokens, `max-sm:rounded-4xl` |
| ≤700: absolute decorations with `left:3xx/4xx` hidden; `transform:scale` reset | `max-sm:hidden` on float cards, `max-sm:scale-100` |
| ≤700: `margin:0 120px 96px` → `0 20px 48px`; `padding:56px 64px` → `28px 22px` | App-CTA / banner variants |
| ≤700: footer bottom row stacks | `max-sm:flex-col max-sm:items-start` |

The selectors depend on inline `style` substrings (`[style*="padding:20px 120px"]`). None of that survives conversion,
so L0-06 encodes each rule as a variant.

---

## 4. Icons & illustrations

### 4.1 Icons: 77 unique (24×24, stroke-based, `fill="none"`, stroke widths 1.8/2/2.2 → normalise to `currentColor`)

Deduped by normalised path data (stroke/fill attributes stripped). Format: **name** (uses / pages): source shape.

| # | Name | Uses/pages | # | Name | Uses/pages |
|---|---|---|---|---|---|
| 0 | `download` | 245/29 | 39 | `bell` | 3/3 |
| 1 | `check` | 139/16 | 40 | `eye-off` | 3/3 |
| 2 | `drop` (logo, cycle) | 72/29 | 41 | `waves` (pool) | 3/2 |
| 3 | `user` (login) | 59/29 | 42 | `bag` | 3/3 |
| 4 | `star` | 48/8 | 43 | `phone` | 2/2 |
| 5 | `heart` | 41/8 | 44 | `pill` | 2/2 |
| 6 | `arrow-left` (forward in RTL) | 39/14 | 45 | `credit-card` | 2/2 |
| 7 | `chevron-left` | 31/29 | 46 | `info` | 2/2 |
| 8 | `book` (magazine) | 31/10 | 47 | `grid` | 2/2 |
| 9 | `sprout` (teen) | 16/8 | 48 | `blocks` (play house) | 2/1 |
| 10 | `x` | 14/4 | 49 | `music` | 2/1 |
| 11 | `person` (child/partner) | 14/13 | 50 | `exercise` | 2/1 |
| 12 | `mic` | 14/5 | 51 | `hand` (massage) | 2/1 |
| 13 | `store` | 14/11 | 52 | `home` | 2/2 |
| 14 | `shield-check` | 13/9 | 53 | `syringe` | 2/2 |
| 15 | `egg` (pregnancy) | 11/8 | 54 | `return` | 2/2 |
| 16 | `calendar` | 11/6 | 55 | `mail` | 1/1 |
| 17 | `stethoscope` | 10/9 | 56 | `paper-plane` | 1/1 |
| 18 | `target` (ttc) | 10/8 | 57 | `upload` | 1/1 |
| 19 | `bookmark` | 9/4 | 58 | `share` | 1/1 |
| 20 | `users` | 8/8 | 59 | `camera` | 1/1 |
| 21 | `alert-triangle` | 8/8 | 60 | `female` (female coach) | 1/1 |
| 22 | `moon` (menopause) | 8/8 | 61 | `thermometer` | 1/1 |
| 23 | `lock` | 7/7 | 62 | `bottle` | 1/1 |
| 24 | `flame` (hot flash) | 6/3 | 63 | `bowl` | 1/1 |
| 25 | `check-square` | 6/6 | 64 | `cart` | 1/1 |
| 26 | `sparkle` | 5/5 | 65 | `parking` | 1/1 |
| 27 | `map-pin` | 5/5 | 66 | `tree` (park) | 1/1 |
| 28 | `calculator` | 5/5 | 67 | `filter` | 1/1 |
| 29 | `plus` | 5/3 | 68 | `chart-bar` | 1/1 |
| 30 | `trash` | 5/3 | 69 | `clock` | 1/1 |
| 31 | `map` | 4/4 | 70 | `folder` | 1/1 |
| 32 | `search` | 4/3 | 71 | `pulse` | 1/1 |
| 33 | `file-text` | 4/4 | 72 | `tag` | 1/1 |
| 34 | `truck` | 4/3 | 73 | `card` (wallet card) | 1/1 |
| 35 | `minus` | 4/2 | 74 | `ruler` | 1/1 |
| 36 | `package` | 4/4 | 75 | `cart-alt`: **merge into `cart`** | 1/1 |
| 37 | `message` | 3/3 | 76 | `flask` (lab test) | 1/1 |
| 38 | `navigate` (directions) | 3/2 | | | |

There are 76 unique icons after merging `cart-alt`. `download` alone appears 245 times (8.4 per page) and the drop logo
72 times, which is what a cached sprite saves. L0-07 re-extracts the shapes the same way: strip
stroke/fill/width/linecap/linejoin attributes, group by inner markup, and keep only `viewBox="0 0 24 24"` SVGs.

### 4.2 Illustrations: 28 unique non-icon SVGs

| Name (suggested) | viewBox | Pages | Fate |
|---|---|---|---|
| `hero-orbit` | 0 0 100 100 @620px | index + 6 stage pages | Keep as `x-illustration`, decorative, behind the phone |
| `chart-sparkline` | 0 0 200 50 | menopause (×2) | Keep inside mock screen |
| `chart-growth` | 0 0 200 90 | postpartum (×2) | Keep inside mock screen |
| `map-directory` | 0 0 560 1180 | directory | Replace with a static "open in maps" panel (no maps) |
| `map-place` | 0 0 760 280 | directory-place | Static local illustration + deep links (L5-03) |
| `map-join` | 0 0 700 300 | directory-join | Drop the map picker; use address text + optional lat/lng (L5-05) |
| `map-checkout` | 0 0 260 150 | shop-checkout | Decorative address thumbnail; may be dropped |
| `place-cover-1…7` (pool, class, playhouse, music…) | 0 0 400 200 | directory, directory-place, directory-join, -booked, -business | **Demo only**; production uses media via `<x-picture>`. Keep one as the empty-cover fallback |
| `product-*` 11 variants (bodysuit, sleepsuit, socks, hat, blanket, pad, shampoo, cup, bottle…) | 0 0 200 200 | shop* | **Demo only**; production uses product media. Keep one fallback |
| `category-*` ~8 (stroller, crib, bath, sunscreen, makeup, hair, body, skincare…) | 0 0 200 200 | shop | Default category covers until admin uploads media |

Article covers in the design are CSS gradients plus an icon, with no SVG file. Production uses post cover media. The
stage gradient is the fallback when a post has no cover.

---

### 4.3 Pipeline result (L0-07)

- `resources/svg/icons/`: **76** icons (77 shapes, `cart-alt` merged into `cart`), extracted by
  `node tools/build-sprite.mjs --extract`. Sprite `resources/svg/sprite.svg` (Vite asset, hashed):
  **9.0 KB raw / 2.4 KB gzip**, one request per page, immutable cache. No preload (see tools/build-sprite.mjs header).
- `resources/svg/illustrations/`: **28** SVGO-optimised files (names as §4.2: `hero-orbit`, `chart-sparkline`,
  `chart-growth`, `map-{directory,place,join,checkout}`, `place-cover-{pool,massage,movement,yoga,playhouse,music}`,
  `product-{bodysuit,sleepsuit,pad,socks,hat,blanket,shampoo,cup,bottle,serum,sunscreen,lipstick}`,
  `category-{stroller,crib,bath}`). The build fails if one is not optimised.
- Usage: `<x-icon name="drop" class="size-5" />` (`label`, `stroke`, `size` props) and
  `<x-illustration name="hero-orbit" />` (`width`, `height`, `label`).

## 5. Dynamic data vs static copy; placeholders

### 5.1 Per-page data source

| Page | Dynamic (DB / settings) | Static copy (`lang/fa/*.php` or Blade) |
|---|---|---|
| all | Settings: app-store links (4), web-app/login URL, socials (3), enamad, terms URL, emergency number (115), footer tagline, ©-year (Jalali) | nav, footer labels |
| index | latest posts (3, L4), FAQ group `home` (4), app links + QR | hero, stage cards (6, from the stage registry), 5 sections, tools (4), privacy, promise banner |
| stage ×6 | readings (3 posts by life stage, L4), FAQ group `stage-<slug>` (3), tools/service cards (registry per stage), emergency text | hero, feature splits, mock screens |
| services | — (cards link to directory/shop) | all |
| plus | prices `[قیمت سالانه]`, `[قیمت ماهانه]`, `[قیمت هر بسته]`, refund policy → settings `PlusSettings`; FAQ group `plus` (grid) | plan feature lists |
| tools | calculators (computed), checklists (static), guides (static) | all copy |
| about | team members (config/content: name, role, photo media), scientific council names, open positions link | story, values, red lines |
| social-responsibility | programs (4, placeholders), donation amounts, founder quote, transparency report URL | commitment copy |
| privacy | last-updated date, DPO email, full-policy URL/content | principles, tools, consents |
| faq | FAQ groups ×5 (12 items) | intro |
| contact | contact settings (support email, partnership email, phone, office hours, address, response hours), FAQ group `contact` (3), topic list | form labels and copy |
| blog | posts (featured + grid), categories (chips: همه + 6 stages + «خانواده و همدم»), newsletter form | intro |
| article | post (title, body, reading time, updated date, reviewer, sources), TOC, related posts (3) | — |
| directory | places (cards, count «۲۴ مجموعه»), categories (9 chips), cities/districts, filters (باز است، مربی خانم، اتاق شیردهی، رزرو آنلاین، قیمت), sort | intro, business CTA |
| directory-place | place: name, category, verified, rating/count, district/city, open-now, gallery (12), about, amenities, services + prices, hours, reviews, address, rules/cancellation; booking widget (services, child, days, slots) | labels |
| directory-booked | booking: code, place, service, date/time, child | copy |
| directory-business | FAQ group `directory-business` (4), review time `[زمان بررسی]` (setting) | why, steps, requirements, terms |
| directory-join | categories, amenities, age ranges, weekdays | step copy |
| directory-join-done | join request: code, place name, timeline, `[زمان بررسی]` | copy |
| shop | categories (2 roots × 8 tiles), best sellers (5 + 5), free-shipping condition (setting), sisemoni-list app CTA | promos, trust strip |
| shop-list | category, products (48), filters (category counts, size, color, price, in stock), sort | — |
| shop-product | product, variants (color, 5 sizes), specs, size chart, reviews, seller, «معمولاً با این می‌خرند» | shipping/returns labels |
| shop-cart | cart lines grouped by **seller**, free-shipping remaining, suggestions (4) | — |
| shop-checkout | address, per-seller delivery slots, discreet packaging toggle, payment (COD only; COD cap `[سقف مبلغ]`), summary | — |
| shop-done | order: code, per-seller status timeline, delivery windows; returns 7 days (setting) | app CTA |

### 5.2 Placeholders → settings / content (all `[...]` tokens found)

| Placeholder | Where | Becomes |
|---|---|---|
| `[اینماد]` | footer, every page | `LegalSettings.enamad` (allowed HTML/link) |
| `[ایمیل مسئول داده]` (also `mailto:[…]`) | contact | `LegalSettings.data_protection_email` |
| `[ایمیل]`, `[تاریخ]` | privacy | `LegalSettings.dpo_email`, `privacy_updated_at` |
| `[ایمیل پشتیبانی]`, `[ایمیل همکاری]`, `[شماره تماس]`, `[روزها و ساعت کاری]`, `[ساعت پاسخگویی]`, `[نشانی دفتر]`, `[نیاز به هماهنگی قبلی؟]` | contact | `ContactSettings` |
| `[راهنمای بازیابی]` | contact FAQ | FAQ item content |
| `#`, `#download` («ورود», «دانلود اپ», store badges, socials) | every page | `AppLinksSettings` (bazaar, myket, google_play, app_store, web_app), `SocialSettings` |
| `[نسخه سبک برای گوشی‌های قدیمی‌تر]`, `[برای گوشی‌های قدیمی و اینترنت ضعیف]` | faq, social-responsibility | `AppLinksSettings.lite_url` + copy |
| `[قیمت سالانه]`, `[قیمت ماهانه]`, `[قیمت هر بسته]`, `[امکانات دیگر پلاس]`, `[بسته‌های دیگر]`, `[سیاست بازگشت وجه: مدت و شرایط]` | plus | `PlusSettings` (new group, L3-06) |
| `[سال]`, `[عدد]`, `[مأموریت در یک جمله]`, `[عکس]`, `[نام]`, `[نقش]`, `[نام متخصصان: …]`, `[موقعیت‌های باز]`, story paragraph | about | about content file / team config (L3-08) |
| `[مبلغ ۱]`, `[مبلغ ۲]`, `[مبلغ دلخواه]`, `[جمله بنیان‌گذار…]`, `[نام بنیان‌گذار]`, program texts, `[هر ۶ ماه: …]` | social-responsibility | content file; donation: see §8 (no gateway) |
| `[سن و شرط رضایت والدین طبق قوانین]` | teen | FAQ/content (legal copy needed from the owner) |
| `[تاریخ]`, `[نام و تخصص]`, `[فهرست منابع علمی]`, `[نام و تخصص متخصص]` | article, blog | Post `updated_content_at`, reviewer Author, sources field |
| `[آدرس کامل مجموعه]`, `[شرایط لغو مجموعه]` | directory-place | Place fields |
| `[زمان بررسی]` | directory-business, join-done | `DirectorySettings.review_days` |
| `[خیابان، کوچه، پلاک، طبقه]` | directory-join | input placeholder |
| `[شرایط ارسال رایگان]`, `[مبلغ باقی‌مانده]`, `[محاسبه در مرحله بعد]` | shop, shop-cart | `ShopSettings.free_shipping_threshold` + computed |
| `[آدرس کامل]`, `[سقف مبلغ]` | shop-checkout | order address; `ShopSettings.cod_max_amount` |
| `[کشور سازنده]` | shop-product | product spec |

Fake demo data in the design (names «سارا/علی», codes ۴۸۲۱/۷۳۱۰۴/۵۲۰۸۴۱, ratings, prices, «۲۴ مجموعه», «۴۸ کالا») goes
into **demo seeders only**, never hard-coded in views.

---

## 6. SEO / performance / accessibility gaps in the export

**Head / SEO**
- No canonical, OG, Twitter or JSON-LD on any page. No favicon, manifest or theme-color.
- **Titles:** 28 of 29 are under 30 characters. Only index (50) and tools (60) pass the 30–60 rule. Several titles have
  no brand suffix («درباره ریتمی», «مجله ریتمی», «فروشگاه ریتمی», «رایگان و ریتمی پلاس», «مسئولیت اجتماعی ریتمی»). The
  templated ones are generic («مقاله — مجله ریتمی», «محصول — فروشگاه ریتمی», «مجموعه — خدمات مادر و کودک»). Each page
  task writes a 30–60 char title with the `%s — ریتمی` template.
- **Descriptions:** 27 of 29 share the same 44-char text («ریتمی — همراه سلامت زنان؛ جایگزین پزشک نیست.»). Only index
  (115) and tools (92) are unique. Every page task writes a unique 70–160 char description.
- **Headings:** no `h3` anywhere. Card and step titles are `<b>`, so the outline is flat. Use `h3` for card titles
  inside an `h2` section (stage, service, feature, article, product, place cards, FAQ categories, footer column titles
  stay non-headings).
- Stage-page FAQ `h2` is the bare stage name («پیگیری چرخه», «بارداری»…) with the eyebrow «سؤال‌های رایج». It should be
  «سؤال‌های رایج درباره …».
- blog uses the featured post title as an `h2` link. That is fine, but the grid cards have no headings.
- **Links:** 404 `href="#"` in total (11–24 per page): login, store badges, socials, terms, enamad, filters, sort,
  share, «ادامه» links in feature splits, faq nav, «پاک کردن».
- `#how` («چطور کار می‌کند؟») is used on all 6 stage pages, but only index has `id="how"`. That is a broken anchor.
- `#download` («دانلود اپ» in the header) is broken on the 16 pages without an `id="download"` section: about, article,
  contact, all directory*, privacy, all shop*. Fallback: `/#download`.
- Footer «ابزارها» has 4 links, all to `tools.html`. Make them anchor targets (`/tools#due-date`, `#fertility`,
  `#hospital-bag`, `#sisemoni`).
- **Breadcrumbs:** visual on 4 pages only, not `ol/li`, and article's `<nav>` has no `aria-label`. BreadcrumbList is
  missing everywhere.
- Images: no `<img>` at all. Products and places are inline SVG placeholders with generic
  `role="img" aria-label="تصویر نمونه محصول"` (61) / «تصویر نمونه» (17). Production needs `<x-picture>` with real alt +
  width/height.
- YMYL / E-E-A-T: the article shows reviewer and sources only as placeholders. The about page has a scientific-council
  placeholder. There is no editorial-policy page.

**Accessibility**
- No `<main>` landmark and no skip link. The footer has no `<nav>`.
- **Fake form controls:** contact (name/email/message are `<span>`s inside `<label>`; topic chips are `<span>`s),
  blog newsletter input, faq search box, directory search fields (3 `<span>` "inputs"), directory-join fields, booking
  day/time and size options (`<button>`s without a group or state). Only tools (2 real text inputs), shop-checkout
  (radio `name="pay"`), shop-list/directory (checkboxes) and tools checklists use real inputs.
- tools inputs have `aria-label="lmp"` / `"cycle"` (English variable names) that override the visible Persian labels.
  Remove them and wrap the inputs in proper `<label>`s.
- Buttons: 21 on directory-join and 12 on directory-place have no `type`, and pickers have no `aria-pressed`/radio
  semantics. Color swatches use `aria-label="رنگ"` with no value name.
- Mobile menu: no Esc handling, focus is not moved or returned, and the menu is not inert when closed (it uses
  `hidden`, which is fine).
- **Contrast** (WCAG AA 4.5:1 for small text):
  - `stage-pregnancy #D9447F` on white is **4.13**. It fails for the 13px article-card stage label; use `#B8336B` or a bolder/larger label.
  - `primary #6E54F0` on `canvas #F7F3FF` is 4.59, which passes narrowly. Do not lighten it.
  - Placeholder `#9A93B3` is 2.92. That is acceptable for placeholders only; never use it for text.
  - `muted` at 75% opacity on white is 3.7. That fails if it is ever used; the design uses 75% only on dark («دریافت از», 8.8:1).
  - Everything else passes: muted on white 6.7, on-night-muted on night 8.7, lilac on night 8.7, white on danger 6.0, amber 5.9, teal 5.4, green 6.0.
- 10.5px text (185 uses, mostly «دریافت از») and 11px text inside mockups: allow it in decorative mockups (`aria-hidden`), raise to 11.5–12 elsewhere.
- Phone mockups and the orbit illustration are already `aria-hidden` in places; enforce that on all of them. The fake QR is `aria-hidden`.

**Performance**
- HTML weight: 13–62 KB per page, dominated by inline styles (6,509) and repeated SVG (up to 69 per page). Tailwind
  plus the sprite cuts most of it.
- Heavy CSS-drawn DOM in phone mockups (index 358 styles). Render them as cached Blade partials and keep the DOM shallow.
- Fonts: 12 woff2 files with `font-display:swap` and `unicode-range` split (good). Preload only 2.
- `ritme.js` is 5 KB, sync-free (`defer`), and runs on every page even though calculators exist only on tools. Split it
  into a `menu` module and a `calc` module (tools only).
- `backdrop-filter: blur` on 14 float cards is costly on low-end phones. It is hidden at ≤700 anyway.

**JS needs (progressive, `data-module`)**
- `menu`: all pages.
- `calc`: tools.
- Accordions: native `details`, no JS.
- `qty`: cart.
- `variant-picker`: product; works without JS via radios.
- `booking-picker`: place page; radios plus a no-JS fallback.
- `stepper`: join; long form without JS.
- `gallery`: lightbox on place and product.
- `faq-filter`: client-side search on `/faq`.
- `share`: copy link, article.
- `cart-badge`: reads a cookie.
- Filters: plain GET forms.
- Nothing else; no map JS.

---

## 7. URL map (old → new)

Rules for L1-05:
- `*.html` → 301 to the new route.
- WordPress method-B/C slugs (`/<file>/`) → 301. Trailing-slash removal runs first, then the slug map.
- Method-A static copies under `/ritme-static/<file>.html` → 301 by the same map.
- `/ritme-static/assets/*` and `/wp-content/uploads/ritme/*` → **410**.
- Pages that only make sense with an id (article, place, product, booked, done) 301 to their **list** page, not to
  demo content.

| Old (`.html`) | Old (WordPress slug) | New | Status |
|---|---|---|---|
| `/index.html` | `/index/`, `/home/` | `/` | 301 |
| `/cycle.html` | `/cycle/` | `/cycle` | 301 |
| `/ttc.html` | `/ttc/` | `/ttc` | 301 |
| `/pregnancy.html` | `/pregnancy/` | `/pregnancy` | 301 |
| `/postpartum.html` | `/postpartum/` | `/postpartum` | 301 |
| `/menopause.html` | `/menopause/` | `/menopause` | 301 |
| `/teen.html` | `/teen/` | `/teen` | 301 |
| `/services.html` | `/services/` | `/services` | 301 |
| `/plus.html` | `/plus/` | `/plus` | 301 |
| `/tools.html` | `/tools/` | `/tools` | 301 |
| `/about.html` | `/about/` | `/about` | 301 |
| `/social-responsibility.html` | `/social-responsibility/` | `/social-responsibility` | 301 |
| `/privacy.html` | `/privacy/` | `/privacy` | 301 |
| `/faq.html` | `/faq/` | `/faq` | 301 |
| `/contact.html` | `/contact/` | `/contact` | 301 |
| `/blog.html` | `/blog/` | `/blog` | 301 |
| `/article.html` | `/article/` | `/blog` | 301 |
| `/directory.html` | `/directory/` | `/directory` | 301 |
| `/directory-place.html` | `/directory-place/` | `/directory` | 301 |
| `/directory-booked.html` | `/directory-booked/` | `/directory` | 301 |
| `/directory-business.html` | `/directory-business/` | `/directory/business` | 301 |
| `/directory-join.html` | `/directory-join/` | `/directory/join` | 301 |
| `/directory-join-done.html` | `/directory-join-done/` | `/directory/business` | 301 |
| `/shop.html` | `/shop/` | `/shop` | 301 |
| `/shop-list.html` | `/shop-list/` | `/shop` | 301 |
| `/shop-product.html` | `/shop-product/` | `/shop` | 301 |
| `/shop-cart.html` | `/shop-cart/` | `/shop/cart` | 301 |
| `/shop-checkout.html` | `/shop-checkout/` | `/shop/cart` | 301 |
| `/shop-done.html` | `/shop-done/` | `/shop` | 301 |
| — | `/ritme-static/<file>.html` | same as the `.html` row | 301 |
| — | `/ritme-static/assets/*`, `/wp-content/uploads/ritme/*` | — | 410 |

Note: `/blog/`, `/shop/`, `/directory/` with a trailing slash are also the new paths' slash variants. They resolve
through the trailing-slash 301, not through the slug map.

Machine-readable map, consumed by `tools/shot.mjs --all` (L0-08) and the redirect tests (L1-05). `design` is the
fidelity source file; `route` is the URL to screenshot (demo slugs come from the demo seeders):

```json urlmap
[
  {"design": "index.html", "route": "/", "name": "home", "owner": "L3-02"},
  {"design": "cycle.html", "route": "/cycle", "name": "stage.cycle", "owner": "L3-03"},
  {"design": "ttc.html", "route": "/ttc", "name": "stage.ttc", "owner": "L3-04"},
  {"design": "pregnancy.html", "route": "/pregnancy", "name": "stage.pregnancy", "owner": "L3-04"},
  {"design": "postpartum.html", "route": "/postpartum", "name": "stage.postpartum", "owner": "L3-05"},
  {"design": "menopause.html", "route": "/menopause", "name": "stage.menopause", "owner": "L3-05"},
  {"design": "teen.html", "route": "/teen", "name": "stage.teen", "owner": "L3-05"},
  {"design": "services.html", "route": "/services", "name": "services", "owner": "L3-06"},
  {"design": "plus.html", "route": "/plus", "name": "plus", "owner": "L3-06"},
  {"design": "tools.html", "route": "/tools", "name": "tools", "owner": "L3-07"},
  {"design": "about.html", "route": "/about", "name": "about", "owner": "L3-08"},
  {"design": "social-responsibility.html", "route": "/social-responsibility", "name": "social-responsibility", "owner": "L3-08"},
  {"design": "privacy.html", "route": "/privacy", "name": "privacy", "owner": "L3-08"},
  {"design": "faq.html", "route": "/faq", "name": "faq", "owner": "L3-09"},
  {"design": "contact.html", "route": "/contact", "name": "contact", "owner": "L3-10"},
  {"design": "blog.html", "route": "/blog", "name": "blog.index", "owner": "L4-02"},
  {"design": "article.html", "route": "/blog/period-pain", "name": "blog.show", "owner": "L4-03"},
  {"design": "directory.html", "route": "/directory", "name": "directory.index", "owner": "L5-02"},
  {"design": "directory-place.html", "route": "/directory/place/ab-pari", "name": "directory.place", "owner": "L5-03"},
  {"design": "directory-booked.html", "route": "/directory/booked/{code}", "name": "directory.booked", "owner": "L5-04"},
  {"design": "directory-business.html", "route": "/directory/business", "name": "directory.business", "owner": "L5-05"},
  {"design": "directory-join.html", "route": "/directory/join", "name": "directory.join", "owner": "L5-05"},
  {"design": "directory-join-done.html", "route": "/directory/join/done", "name": "directory.join.done", "owner": "L5-05"},
  {"design": "shop.html", "route": "/shop", "name": "shop.index", "owner": "L6-02"},
  {"design": "shop-list.html", "route": "/shop/category/baby-clothes", "name": "shop.category", "owner": "L6-02"},
  {"design": "shop-product.html", "route": "/shop/product/long-sleeve-cotton-bodysuit-3", "name": "shop.product", "owner": "L6-03"},
  {"design": "shop-cart.html", "route": "/shop/cart", "name": "shop.cart", "owner": "L6-04"},
  {"design": "shop-checkout.html", "route": "/shop/checkout", "name": "shop.checkout", "owner": "L6-05"},
  {"design": "shop-done.html", "route": "/shop/order/{code}", "name": "shop.order", "owner": "L6-05"}
]
```

Pages added beyond the design (see §8): `/terms` (`terms`, L3-08). `/privacy` carries the full policy as a
section or `/privacy/policy` (decided in L3-08).

---

## 8. Corrections to later tasks

The task files below were edited, body text only with status untouched, after this audit:

| Task | Change | Why |
|---|---|---|
| L1-02 | Header variant map (10 dark / 19 light pages). Active-nav map. «مرحله‌ها» chevron is decorative (no dropdown). Header «ورود» → settings web-app URL; «دانلود اپ» → `#download` when the page has the section, else `/#download`. Footer «ابزارها» links → `/tools#…` anchors; footer columns as `<nav>`. | §2.1, §6 broken anchors |
| L3-01 | «tabs (JS module)» replaced by `x-ui.chip-nav` link chips (the design has no JS tabs). Added `x-ui.alert-emergency`, `x-ui.qr` (local QR, no external service), `x-ui.success-hero`, `x-ui.stepper`, `x-ui.timeline`, `x-ui.toggle-row`, `x-ui.promise-banner`, `x-ui.app-cta` aside variant, `x-cards.value`, `x-cards.review`. | §2.2–2.4 |
| L3-03 | Stage template also includes `x-layout.stage-nav`, `x-ui.alert-emergency`, the stage FAQ (group `stage-<slug>`, descriptive h2), `x-mock.phone` + screens. Fix `#how` (first feature section gets `id="how"`). Teen has no help block. | §2.4, §6 |
| L3-07 | Answer to «separate landing URLs?»: the design has **one** page holding both calculators, 3 checklists and 4 guides. Ship anchors `#due-date`, `#fertility`, `#hospital-bag`, `#sisemoni` (footer targets) now, and defer `/tools/due-date` and `/tools/fertility-window` to a later SEO task. Remove the English `aria-label`s. | §1, §6 |
| L3-08 | Add `/terms` (footer link exists, no design page: simple legal page in the privacy layout, content from settings/content file) and the full privacy-policy text (section or `/privacy/policy`). Donation block: no payment gateway exists (COD-only decision), so render «حمایت» as a contact/partnership CTA until a gateway task exists; amounts stay hidden. Transparency report link from settings (hidden when empty). | §1 missing pages, §5.2 |
| L3-09 | Seed list adds the **six stage FAQ groups** (3 items each) and notes the plus FAQ is a 2-column card grid (`x-faq variant="grid"`). `/faq` search box → `faq-filter` JS module (progressive; no-JS shows all). | §2.2 |
| L5-02 | Right-hand map column (`map-directory`, zoom ±, «با جابه‌جایی نقشه جست‌وجو کن») conflicts with the no-maps decision. Replace it with a static local illustration + city/district links, or drop the column, and keep the result list full width on mobile. | §4.2, decisions |
| L6-01 | The design is a **multi-seller** catalog: product cards show a seller («پوشاک پنبه‌ریز», «بهداشتی بانو», «خانه سیسمونی ماه‌نو»), «فروشنده بررسی‌شده», and cart/checkout/done group lines per seller with their own delivery estimates. Add a minimal `Seller` (name, slug, is_verified, shipping days min/max) or confirm single-seller with the user before modelling. | §5.1 shop rows |
| L6-05 | The checkout design offers «درگاه بانکی» and «پرداخت در محل». Per the COD-only decision, render only COD (no disabled gateway option). Keep per-seller delivery slots, «بسته‌بندی ساده» (discreet packaging) toggle, and the COD cap `ShopSettings.cod_max_amount`. | §5.1 |

No change needed: L0-05 (tokens in §3; keep all 5 Vazirmatn weights + Lalezar), L0-06 (rules in §3.5), L0-07 (names in
§4.1; 76 icons after merge), L0-08 (reads the `json urlmap` block above), L1-01 (placeholders in §5.2; `PlusSettings`,
`ShopSettings`, `DirectorySettings` are created by their domain tasks), L1-05 (map in §7), L3-02, L3-04–L3-06, L3-10,
L4-*, L5-01/03/04/05/06, L6-02/03/04/06.

### Open items (owner decisions)
1. Single seller vs marketplace for the shop (L6-01).
2. Real copy for every `[...]` placeholder (§5.2), especially legal: teen consent age, terms, full privacy policy,
   refund policy, enamad.
3. Donations: keep hidden until a payment gateway is approved?
4. Plus prices: show on the site or keep «در اپ»?
5. Demo slugs (`period-pain`, `ab-pari`, `baby-clothes`, `long-sleeve-cotton-bodysuit-3`) can be Persian if preferred;
   they are seed data only.
