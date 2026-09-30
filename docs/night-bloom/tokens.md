# Night & Bloom — design tokens (B-N1-01)

Extracted by script from all 363 artboards (`docs/design/night-bloom/*/`). Light = `nbl_` + `v19_` (181 files),
dark = `nbd_` + `nb2_` (182). Light↔dark pairs were derived by zipping the colour sequence of every `nbl_X`/`nbd_X`
pair (identical markup, only values differ), so every dark value below is the one the designer actually paired.

B-N1-02 implements this table in `frontend/src/app/globals.css` (`:root` = light, `[data-theme="dark"]` = dark).

## 1. Two light dialects — pick A

The light artboards come in two generations:

| Dialect | Files | Primary | Text 1 / 2 / 3 | Notes |
|---|---|---|---|---|
| **A (canonical)** | 16 hub/home screens: `Cycle_Home`, `Cycle_Calendar`, `An_Hub`, `PregFull_Main/Week`, `v15_Main`, `v16_ChildHome`, `v14_Main`, `Vitals_Hub`, `v17_Main`, `v18_AssistantHub`, `Learn_Hub`, `Todo_Home`, `Me_Hub`, `Me_Mode`, `Prem_TrialHome` | `#6E54F0` | `#231B3B` / `#5E5873` / `#6A6480` | purple-tinted inks; the newest pass; matches the task spec |
| B (legacy) | the other 165 light files | `#7B61FF` | `#2F2F35` / `#5A5A64` / `#6F6F78` | = today's app palette re-used |

Dark has **one** dialect (both A and B map onto it). Decision (default, see `bloom/QUESTIONS.md`): **A is canonical**;
B values are treated as A when auditing fidelity. Reason: white on `#6E54F0` = 5.0:1 (AA) vs `#F7F3FF` on `#7B61FF`
= 3.85:1 (fails AA for 15px button text).

## 2. Colour tokens

Names keep today's tokens where one exists (so existing screens restyle for free); new semantic names are added and
the legacy names become aliases (§3).

| Token | Light | Dark | Today (light → dark) | Use |
|---|---|---|---|---|
| `--page` | `#F7F3FF` | `#17112B` | `#F2ECFF` → `#131022` | canvas / body background |
| `--surface` | `#FFFFFF` | `#221A3D` | `#fff` → `#1B172B` | cards, sheets |
| `--surface-2` | `#F4F0FF` | `rgba(255,255,255,.07)` | `#F4F0FF` → `#251E3E` | round header buttons, tab-list track, fields |
| `--surface-3` | `#F1EDFB` | `rgba(255,255,255,.05)` | `#FAF8FF` → `#211C36` | inset tiles inside a card |
| `--surface-4` *(new)* | `#F1EDFB` | `#2A2150` | — | opaque inset (chart tooltips, overlapping tiles) |
| `--surface-glass` *(new)* | `rgba(255,255,255,.94)` | `rgba(34,26,61,.92)` | — | bottom nav, sticky footers (+ `backdrop-filter: blur(10px)`) |
| `--line` | `#E7E1F4` | `#34295A` | `#E7E1F4` → `#2A2440` | 1px card/control border (cards have **no** shadow) |
| `--line-strong` *(new; = `--brand-line-soft`)* | `#DDD6FA` | `#3B2F63` | `#DDD6FA` → `#3A3060` | ghost-button / selected outline |
| `--track` | `#E7E1F4` | `rgba(255,255,255,.15)` | `#DBD4EC` → `#3D3560` | progress rail, empty bars, switch off |
| `--text-1` *(new)* | `#231B3B` | `#F6F1FF` | `--ink #2F2F35` | titles, body |
| `--text-2` *(new)* | `#5E5873` | `#B8AED6` | `--ink-3 #5A5A66` | secondary text, inactive nav |
| `--text-3` *(new)* | `#6A6480` | `#8C82AD` | `--muted-2 #6F6F78` | captions, axis labels |
| `--text-4` *(new)* | `#8C82AD` | `#6E6392` | `--muted-soft` | placeholder / disabled only (<AA) |
| `--brand` | `#6E54F0` | `#B9A6FF` | `#7B61FF` → `#8A72FF` | primary as text/icon, active-tab bar |
| `--brand-fill` | `#6E54F0` | `#B9A6FF` | `#7B61FF` (theme-stable) | CTA, FAB, switch on, selected tab/chip |
| `--on-brand` *(new)* | `#FFFFFF` | `#17112B` | `--on-accent #fff` (stable) | text/icon on `--brand-fill` — **flips** |
| `--brand-strong` | `#5B41E6` | `#D6CBFF` | `#5B41E6` → `#B3A5FF` | links, pressed/hover |
| `--brand-ink` *(new)* | `#4428B8` | `#D6CBFF` | — | text on `--brand-soft` |
| `--brand-soft` *(= `--pink-bg`, `--violet-soft`)* | `#ECE6FF` | `rgba(185,166,255,.16)` | `#ECE6FF` → `#2C2450` | tint chips, icon circles, avatars |
| `--brand-2` *(new)* | `#5B41E6` | `#7C5CFF` | — | second chart series / illustration |
| `--data` | `#0FA3C9` | `#4CE0C3` | `#3DD6F3` → — | ovulation, chart lines, measured/predicted markers, «طبیعی» verdicts |
| `--data-deep` | `#0A7390` | `#6FE7D0` | `#0FA3C9` → `#5FDCF5` | turquoise as text (`#0FA3C9` on white is 2.96:1) |
| `--data-soft` | `#E3F9FE` | `#1E3340` | `#E3F9FE` → `#10303B` | turquoise surface |
| `--warm` *(= `--amber`)* | `#F5A623` | `#FFB86B` | `#F5A623` | fertile window, streaks, Plus, warnings |
| `--warm-deep` *(= `--amber-deep`)* | `#B45309` | `#FFC98A` | `#B45309` → `#F0B45E` | amber as text |
| `--warm-soft` *(= `--amber-tile`)* | `#FFF3DF` | `#3A2F1E` | same | amber surface |
| `--period` | `#E8436F` | `#FF6B8B` | `#E8436F` → `#FF6D91` | period marker/fill (red = menstruation) |
| `--period-deep` / `--rose` | `#B82A52` | `#FF6B8B` | `#C92C57` → `#FF8FAB` | period / rose as text, hero numbers |
| `--period-soft` | `#FDE4EB` | `#3A2140` | `#FFE4EC` → `#3E1B29` | period day cell |
| `--period-line` *(= `--care-rose-line`)* | `#F9C3D0` | `#6B3A4D` | same | rose outlines (cancel) |
| `--bloom` *(= `--pink`)* | `#FF6FAE` | `#FFD5B8` | `#FF6FAE` | illustration / avatar / companion accent (peach at night) |
| `--bloom-deep` | `#D9447F` | `#FFD5B8` | `--rose-deep #D8407E` | bloom as text |
| `--success` | `#0F7B6C` | `#7FE0A8` | same | done / taken |
| `--success-soft` | `#E7F8EF` | `#1F3A2E` | same | |
| `--success-fill` | `#0F7B6C` | `#0F7B6C` | same (stable) | fill under white text |
| `--danger` | `#C42D57` | `#FF8FA3` | `#E5484D` → `#FF7B7E` | errors, destructive text (design uses the rose ramp) |
| `--danger-soft` | `#FDE4EB` | `#5A2A3D` | `#FCE7E7` → `#3A2224` | error surface / urgent modal |
| `--caution-soft` *(= `--amber-soft`)* | `#FEF3C6` | `#3A2F14` | same | «ارزش پیگیری» boxes |
| `--scrim` | `rgba(0,0,0,.35)` | `rgba(0,0,0,.6)` | `rgba(17,32,47,.38)` → `rgba(4,2,12,.62)` | behind sheets/dialogs |

The M5 `--fert-*` dark values already equal these (they were taken from `nb2_`), so they can be re-pointed to the
generic tokens.

## 3. Legacy aliases (B-N1-02)

Keep these names working, re-pointed so every screen picks up the new palette:

- `--ink`, `--ink-2` → `var(--text-1)`; `--ink-3`, `--muted`, `--slate`, `--steel`, `--checkup-body` → `var(--text-2)`;
  `--muted-2`, `--muted-3` → `var(--text-3)`; `--muted-soft` → `var(--text-4)`.
- `--violet`, `--brand-deep`, `--pink-vivid`, `--indigo*` → `var(--brand)` / `var(--brand-strong)`.
- `--teal*`, `--blue*` → `--data*`; `--amber*`, `--fert-amber*` → `--warm*`; `--care-rose*`, `--fert-rose` → `--period*`.
- `--on-accent` stays theme-stable **white** only for the remaining saturated fills (`--success-fill`, period fill);
  primary buttons switch to `--on-brand` (dark text on lavender at night). Update `THEME_STABLE` in
  `scripts/check-dark-mode.mjs` accordingly.

## 4. Gradients & background layer

The saturated purple→pink brand gradient (`--gradient-brand`, today mandated for CTAs/FAB by `frontend/CLAUDE.md`
§10.2) **does not appear** in Night & Bloom: CTAs and the FAB are solid `--brand-fill`. Gradients are soft tints only:

| Token | Light | Dark | Use |
|---|---|---|---|
| `--hero-tint` | `linear-gradient(135deg, rgba(110,84,240,.14), var(--surface) 72%)` | `linear-gradient(135deg, rgba(124,92,255,.38), var(--surface) 72%)` | hero cards (phase, week, plus) |
| tint pattern | `linear-gradient(135deg, <accent> 15–20%, var(--surface) 70%)` | same with dark surface | amber/rose/bloom highlight cards |
| `--avatar-grad` | `linear-gradient(135deg, #FF6FAEaa, #7B61FF55)` | `linear-gradient(135deg, #FFD5B8aa, #B9A6FF55)` | avatars, illustration discs |
| `--bg-glow` | `radial-gradient(circle, rgba(124,92,255,.32), rgba(124,92,255,0) 65%)` | same | 300×300 circle, `top:-120px; inset-inline-start:-80px` (RTL right), on every hub/form |
| starfield | 9 dots, opacity 0 (invisible) | 9 white dots r 0.8–1.4px, opacity .3–.55 | top 390×300 band of dark screens |

Starfield dot positions (x,y,r,opacity): (30,90,1.2,.5) (120,60,.9,.35) (300,110,1.4,.55) (360,70,.8,.3) (200,140,1,.4)
(80,200,.9,.3) (340,220,1.2,.35) (160,30,.8,.3) (260,40,1,.4). Implement as one CSS layer (radial-gradient dots or an
inline SVG data-URI in CSS), `pointer-events:none`, static (nothing animates in the artboards).

Browser chrome: `theme-color` / `manifest.ts` `background_color` → `#F7F3FF` light, `#17112B` dark.

## 5. Typography

- **Vazirmatn** (existing variable font): weights used 800 (1650×), 700 (1526×), 600 (1242×), 400 (203×), 900 (145×), 500 (6×).
- **Lalezar** 400 (`font-family: Lalezar, Vazirmatn`; `line-height:1`): numerals **and** display titles — 153 digit-only
  uses, 159 with letters (Plus headlines, onboarding titles, companion, all admin page titles). The shipped
  `Lalezar-Subset.woff2` only covers digits + «روز/امروز» → B-N1-02 must ship a full Arabic-script Lalezar subset.

| Role | Size / weight | Token proposal |
|---|---|---|
| Hero number (home ring) | Lalezar 72 (also 56–88 rare) | `--fs-num-xl: 72px` |
| Big stat / hero title | Lalezar 36–40 | `--fs-num-l: 40px` |
| Stat number | Lalezar 28–32 (30 most common) | `--fs-num-m: 30px` |
| Display title | Lalezar 20–26 | `--fs-display: 24px` |
| Greeting / page H1 | Vazirmatn 20/800 | `--fs-2xl: 20px` |
| Screen title (header) | 17/800 | `--fs-xl: 17px` |
| Button, section title | 15/800 | `--fs-lg: 15px` |
| Body strong | 13.5–14/700 | `--fs-base: 13.5px` |
| Body | 12.5–13/600 | `--fs-md: 12.5px` |
| Caption | 11.5/600–700 | `--fs-sm: 11.5px` |
| Nav label, micro caption | 11 / 10.5 | `--fs-xs: 10.5px` |
| Chart axis | 9.5 (8.5 rare) | `--fs-2xs: 9.5px` |

Paragraph line-height 1.7–1.9 (`--lh-body: 1.8`); numerals `line-height:1`; uppercase Latin micro labels use
`letter-spacing: .08em`. Sizes in between (11, 12, 13, 14) round to the nearest token unless a fidelity audit shows a
visible difference.

## 6. Radii

| Token | Value | Where |
|---|---|---|
| `--r-xs` | 4px (2–7) | bars, progress segments, chart bars |
| `--r-sm` | 12px | small tiles, day cells, tags |
| `--r-md` | 14px | inputs, segmented-tab buttons, list tiles, 58px secondary buttons (16) |
| `--r-lg` | 18px | tab-list track, list groups |
| `--r-xl` | 20px | secondary cards, chips 40px (20/26) |
| `--r-card` | 24px | cards (most common: `24px`, 638×) |
| `--r-pill` | 27px / 999px | 54px buttons, status pills |
| `--r-sheet` | 30–32px (top corners) | bottom sheets |
| `--r-nav` | 35px | floating bottom nav |
| circle | 50% (22px on 44px) | icon buttons, avatars, FAB (28 on 56) |

## 7. Shadows

| Token | Light | Dark | Where |
|---|---|---|---|
| `--shadow-cta` | `0 14px 28px -14px rgba(110,84,240,.8)` | `0 14px 28px -14px rgba(185,166,255,.9)` | primary buttons |
| `--shadow-fab` | `0 0 0 6px rgba(110,84,240,.18), 0 12px 24px -10px rgba(110,84,240,.8)` | `0 0 0 6px rgba(185,166,255,.18), 0 12px 24px -10px rgba(185,166,255,.9)` | FAB |
| `--shadow-float` | `0 20px 40px -18px rgba(0,0,0,.45)` | `0 20px 40px -18px rgba(0,0,0,.8)` | bottom nav, popovers |
| `--shadow-hero` | `0 18px 34px -20px rgba(93,71,214,.35)` | `0 20px 40px -22px rgba(0,0,0,.7)` | hero / care cards (= today `--care-card-shadow`) |
| `--shadow-modal` | `0 8px 30px rgba(17,32,47,.14)` | `0 8px 30px rgba(0,0,0,.8)` | dialogs |
| `--shadow-knob` | `0 1px 4px rgba(0,0,0,.2)` | same | switch thumb |
| `--glow-data` | `0 0 6px var(--data)` | same | «today» dot on charts |

Cards: flat — `background: var(--surface); border: 1px solid var(--line)`, no shadow.

## 8. Spacing & sizes

- 4px grid. Screen gutter **16px** (hubs), **20px** (forms/detail). Section gap 14px (18px on home top).
  Card padding 14–18px (`16px` most common). Scroll bottom padding 130–150px to clear the floating nav.
- Touch: round icon buttons **44×44**; stepper −/+ 44; chips/tabs 40 high; primary button **54** high full width;
  secondary tile button 58; FAB **56**; nav 70 high; date-strip cell 42×54 (radius 16); switch 48×30 (46×28 compact).
- The 54px status bar (`۹:۴۱`) in every artboard is a mock — never build it.

## 9. Contrast (WCAG, computed)

AA holds for: text-1/2/3 on surface in both themes (16.3 / 6.7 / 5.6 light; 14.8 / 7.8 / 4.6 dark), `--on-brand` on
`--brand-fill` (5.0 / 8.65), all dark accents on `#221A3D` (≥6.0). **Fails** if used as text: `#0FA3C9` (2.96),
`#F5A623` (2.03), `#E8436F` (3.84), `#8C82AD` light (3.55) → use the `-deep` token for text.
