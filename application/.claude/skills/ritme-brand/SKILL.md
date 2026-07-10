---
name: ritme-brand
description: Apply the Ritme (ریتمی) brand & design system — pink-based palette, violet accent, RTL-first Persian layout, bundled Persian font (Vazirmatn), light+dark theme. Use whenever building UI, choosing colors, theming, or working with text/typography.
---

# Ritme Brand & Design System

This app is **Ritme (ریتمی)**, a Persian women's-health / cycle-tracking app
(`ir.ritmeapp.ritme`). Identity is **pink-based** (warm, feminine, caring) with a
violet accent — same tokens as the web frontend (`frontend/src/app/globals.css`).
CLAUDE.md §5c. Source of truth for colors:
`app/src/main/res/values/colors.xml`.

## Palette (named tokens — never inline hex in screens)

- Primary `ritme_pink #E91E63` · deep `#E60076` · light `#FB64B6` · container `#FFF1F7`
- Accent `ritme_accent #A91EE9` (violet)
- Neutrals: ink `#11202F`, muted `#707983`, surface `#FFFFFF`, background `#EFF2F4`,
  outline `#E6EAF0`
- Semantic: success `#22B07D`, warning `#F5A623`, error `#D64545`, info `#2E7CD6`
- Dark: background `#0D141B`, surface `#16202A`, ink `#ECF1F6`, container `#3A2230`

> These hexes mirror the web frontend design tokens. When a value changes, edit
> `colors.xml` only (and keep it in sync with `globals.css`).

## Rules

- [ ] No inline `Color(0xFF...)` in Composables — every color is a named token mapped
      through one central `RitmeTheme` (colors + typography + shapes).
- [ ] **RTL-first**: default layout direction RTL, text right-aligned, directional
      icons/chevrons mirrored. Swipe-back (§5b) still pops from the leading edge.
- [ ] **Persian typography**: bundle the Vazirmatn font file as an app asset (same
      family as the web frontend) and set it as the default `FontFamily`. Do NOT add
      a font library (§3).
- [ ] Persian digits (۰۱۲۳) in user-facing numbers, dates, cycle-day counters.
- [ ] Light + dark theme both derived from the same tokens; respect system setting.
- [ ] Screens read colors/typography from `RitmeTheme`, never define their own palette.

## How to check
Grep changed UI files for `Color(0x` or raw `#` hex in Kotlin → flag, replace with a
theme token. Confirm new screens are wrapped in `RitmeTheme` and render correctly in RTL.
