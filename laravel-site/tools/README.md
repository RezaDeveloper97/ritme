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
