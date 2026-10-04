# tools/

Node scripts used while building the site. They run from `laravel-site/` with Node ≥ 22 and use only Node
built-ins (plus packages already in `node_modules` where noted). Nothing here ships to production.

## style2tw.mjs — inline styles → Tailwind (L0-06)

First-pass converter from a design page in `design/html/` to Blade with Tailwind v4 utilities. Use it at the start
of a page task, then extract components by hand.

```bash
node tools/style2tw.mjs design/html/cycle.html > /tmp/cycle.blade.php            # whole <body>
node tools/style2tw.mjs design/html/cycle.html --section 3 > /tmp/s3.blade.php    # 3rd child of .rt-page
node tools/style2tw.mjs design/html/cycle.html --no-icons --quiet                  # keep inline SVGs, no report
node --test tests/tools                                                            # unit tests
```

What it does:

| Input | Output |
|---|---|
| `style="…"` | removed; utilities appended to any existing `class` |
| colours | theme tokens from `resources/css/app.css` `@theme` (`text-on-night-muted`, `bg-primary/13` for `#6E54F022`, `bg-surface` for white backgrounds); anything else → `bg-[#hex]` and listed in the report |
| spacing / sizes | 2px grid → spacing scale (`p-4.5`, `size-10`, `max-w-190`), other values → `[13px]`; `width == height` → `size-*` |
| typography | text/leading tokens (`text-base`, `text-d-3xl`, `leading-relaxed`), else arbitrary; Lalezar → `font-display` |
| radius | radius tokens; `radius == height/2` → `rounded-full` |
| gradients | plain 2–3 stop palette `linear-gradient` → `bg-linear-135/srgb from-… to-…`; the hero glow → `bg-hero-glow`; others → `bg-[…]` with palette colours rewritten to `var(--color-*)` |
| shadows | shadow tokens; `0 0 0 Npx c` → `ring-N ring-*`; others arbitrary |
| RTL | the design is `dir="rtl"`: `left`→`end-*`, `right`→`start-*`, `margin/padding-left/right`→`me/ms`, `pe/ps`, `border-left/right`→`border-e/s`, `text-align:right`→`text-start` |
| responsive | the plain `.rt-page [style*="…"]` rules of `design/html/assets/ritme.css` are parsed and applied with the same substring + source-order semantics as `max-xl:` (≤1180) / `max-lg:` (≤1024) / `max-sm:` (≤700) variants — section paddings (`px-30 max-lg:px-5`, vertical ×0.55), grids (`grid-cols-4 max-lg:grid-cols-2 max-sm:grid-cols-1`), `flex:1 1 0` bases, fixed widths, radius 40, absolute decorations, scale reset. The structural selectors (header nav/actions hidden, flex rows wrap, footer brand cell, footer bottom row) are encoded by hand. Display font sizes need no variants: the `text-d-*` tokens step down in `app.css`. |
| links | `x.html#y` → `{{ route('…') }}#y` from the URL map in `docs/AUDIT.md` §7; slug pages use the demo slug (`route('blog.show', 'period-pain')`), id-only pages (booked, order) fall back to their list route — both listed in the report for review |
| icons | when `resources/svg/icons/` exists (L0-07), an inline SVG whose geometry matches an icon file becomes `<x-icon name="…" class="size-* text-*"/>` (stroke widths are normalised by the sprite) |
| SVG colours | `fill`/`stroke="#hex"` attributes → `fill-*`/`stroke-*` classes |
| text | Blade-safe (`{{` → `@{{`, `@if` → `@@if`) |

Anything it cannot map becomes an arbitrary property (`[prop:value]`) so the output never needs `style=""`.

The report on stderr lists: declaration/utility counts, variants per breakpoint, arbitrary-value count, links that
need review, unresolved `.html` links, icons replaced, **unmapped declarations** and **colours outside the palette**,
and the legacy `rt-*` classes that were kept (burger, mobile menu, calculator helpers — styled by the layout task, not
by Tailwind).

Known limits (fix by hand in the page task):
- `.rt-page` and the `rt-*` helper classes come from `ritme.css`; the layout (L1-02) replaces them.
- Substring semantics are kept on purpose (e.g. `padding:32px 120px 100px` gets `max-lg:py-8`, as the design renders),
  even where AUDIT §3.4 describes the intent differently; the max-lg `flex-wrap` lands on every non-column flex
  row, including inline ones where it changes nothing.
- Repeated markup is not turned into components, and copy is not moved into lang files or settings.

## shot.mjs — fidelity screenshots + pixel diff (L0-08)

Proves a converted page still looks like its design page. Drives headless Chrome over the DevTools protocol with
Node's built-in `WebSocket` (no npm deps). Chrome comes from `/Applications/Google Chrome.app`; set `CHROME_PATH`
to use another binary. The Laravel app must be running (`php artisan serve` → `http://127.0.0.1:8000`).

```bash
node tools/shot.mjs --design cycle.html --route /cycle --out docs/qa/L3-03              # 390 + 1440 (default)
node tools/shot.mjs --design cycle.html --route /cycle --widths 390,768,1440 --out docs/qa/L3-03
node tools/shot.mjs --design shop-done.html --route /shop/order/R-1001 --out docs/qa/L6-05   # parameter routes
node tools/shot.mjs --all --out docs/qa/L3-11                                           # every converted page
node tools/shot.mjs --all --only L3-05 --strict --json /tmp/shot.json                   # owner task / name / file
```

For each page and width it writes `<slug>-<w>-design.png` (the `design/html/<file>` page over `file://`),
`<slug>-<w>-route.png` (`<base><route>`) and `<slug>-<w>-diff.png` to `--out` (default `docs/qa/adhoc`; PNGs under
`docs/qa/` are git-ignored, so keep `--json` reports outside the repo or delete them). Pages are captured full height
(the viewport is grown to the page, so lazy images load), with animations/transitions off,
`prefers-reduced-motion: reduce`, light colour scheme and device scale 1; widths under 768 use mobile emulation.

`--all` reads the ```` ```json urlmap ```` block of `docs/AUDIT.md` §7 and skips routes that do not answer 2xx yet
(not converted) or that contain a `{parameter}` — run those one by one with a real value.

**Diff number.** Share of pixels where any channel differs by more than 32/255, over the larger of the two canvases.
Height mismatches count fully (shown magenta in the diff PNG; changed pixels red over the faded design), so a page
that is a few hundred pixels taller or shorter scores high even when the sections match — fix the height first.

**Thresholds.** Target **< 3 % per width**. Above that the run still exits 0 (prints `!`), unless `--strict`
(exit 2). Every width above 3 % needs an explanation in the task's `tasks/PROGRESS.md` entry (e.g. real content in
place of design placeholders, `<x-picture>` crops, sprite icon stroke widths, intentional fixes listed in AUDIT §8).

**Hard failures (exit 1).** Every request to a non-local origin is blocked and fails the run — the "no external
requests" rule; local means `file:`, `data:`, `blob:`, `about:` and `http(s)://127.0.0.1 | localhost | [::1]`. The run
also fails on console errors, uncaught exceptions, failed requests or HTTP ≥ 400 responses **of the Laravel page**
(the same for the design page are printed but do not fail). `--base` must itself be a local origin.

Exit codes: `0` ok · `1` external request, page error or tool error · `2` above threshold with `--strict`.

## pwa-icons.mjs — default PWA icons, favicons, screenshots (L8-01)

```bash
node tools/pwa-icons.mjs                 # public/icons/*.png|svg + public/favicon.ico from the drop icon + primary token
node tools/pwa-icons.mjs --screenshots   # also public/icons/screenshot-{mobile,desktop}.webp from a running local site
```

Rasterised by headless Chrome (no npm deps, local files/origins only); WebP conversion via PHP GD.
