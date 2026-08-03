---
name: wow-design
description: >-
  Turn any UI, landing page, component, or artifact into a distinctive, modern, "wow"-level design by first researching current cutting-edge design trends and then implementing one bold, unique direction. ALWAYS use this skill when the user asks — in any language, however casually — to make something beautiful, pretty, cool, unique, modern, or impressive. Persian triggers included: «خوشگلش کن», «قشنگش کن», «خفنش کن», «یه دیزاین خفن بزن», «یونیک باشه», «مدرنش کن», «حرفه‌ای‌ش کن». Also trigger when the user says a design looks boring, generic, templated, or "AI-looking" and wants it upgraded, or asks for "creative ideas" for a UI. Do not use for pure backend/logic tasks with no visual output.
---

# Wow Design — از عامیانه تا خفن

When the user casually says "make it pretty / خوشگلش کن", they are actually commissioning a creative director. Treat it as a full design brief with three phases: **Research → Concept → Execute**. Never skip straight to code.

## Phase 1 — Research (خفن‌ترین ایده‌ها)

Before designing, spend 2–4 web searches discovering what is genuinely current and award-winning *for this type of product*:

- Search things like: "awwwards site of the day <product type> 2026", "<industry> landing page design trends 2026", "godly.website <category>", "modern <component> UI patterns".
- Extract 3–5 concrete, *specific* ideas (a scroll technique, a typographic treatment, a color logic, an interaction) — not vague adjectives.
- If the project is RTL/Persian, additionally consider Persian typography opportunities: variable Persian fonts (Vazirmatn, Estedad, Dana, Peyda), large expressive Farsi display type, RTL-native layout ideas. Persian display type used boldly is itself a signature few sites have.
- Skip research only if the user explicitly says "no need to search" or the environment has no web access — then rely on the trend-avoidance rules below.

## Phase 2 — Concept: one bold direction

Write (in thinking, briefly summarized to the user) a mini design plan:

1. **Concept name + one-line thesis** — e.g. "Blueprint Garage: the parts catalog as an engineer's technical drawing".
2. **Palette** — 4–6 named hex values derived from the subject's real world (materials, instruments, environment), not from defaults.
3. **Type** — a characterful display face + complementary body face (+ mono/utility if data-heavy). For Persian: pick real Persian webfonts, never fall back to Tahoma/Arial.
4. **Signature element** — the ONE thing the page will be remembered by: a hero moment, an interaction, a layout device. Spend all boldness here; keep everything else disciplined.
5. **Motion plan** — one orchestrated moment (load sequence or scroll reveal) beats scattered effects. Respect `prefers-reduced-motion`.

### Anti-generic checklist (must pass before coding)

Reject the plan and revise if it contains any of these AI-design tells:
- Cream background + serif display + terracotta/#D97757 accent
- Near-black + single acid-green/vermilion accent
- Broadsheet hairline-rules newspaper layout (unless the brief asks)
- Purple-to-blue gradient on white SaaS layout, glassmorphism cards everywhere
- Emoji as icons; numbered 01/02/03 markers when content isn't a sequence
- Identical card grids of three features with icon-title-paragraph

Ask yourself: "would I produce this exact design for a different product in the same category?" If yes, it's a template, not a choice — change it.

## Phase 3 — Execute

- Follow the plan exactly; derive every color/spacing/type decision from it.
- Build to a quality floor without announcing it: responsive to mobile, visible focus states, reduced-motion support, real content (write specific copy in the product's language — no lorem ipsum, no filler marketing-speak).
- For RTL: set `dir="rtl"`, use logical properties (`margin-inline-start`, `text-align: start`), mirror asymmetric layouts, and test numerals (prefer Persian digits for display text if the brand is Persian).
- Watch CSS specificity collisions between section-level and element-level selectors (paddings/margins cancelling out).
- After building, self-critique once: remove one decoration (Chanel rule), verify the signature element actually lands, check nothing on the anti-generic list crept back in.

## Communicating with the user

Present the concept in 3–5 lines before or alongside the build ("رفتم ترندهای ۲۰۲۶ رو دیدم، این ایده رو انتخاب کردم چون…"), naming the 1–2 research findings that shaped it. Don't dump the whole research log. If the user's request was one casual sentence, don't interrogate them with questions — make confident opinionated choices and state them; offer one alternative direction at the end in a single line.
