---
id: L3-08
title: About, social responsibility and privacy pages
milestone: L3
type: frontend
status: done
depends_on: [L3-01,L1-04]
parallel_group: L3-D
touches: [resources/views/pages/about.blade.php,resources/views/pages/social-responsibility.blade.php,resources/views/pages/privacy.blade.php,app/Http/Controllers/AboutController.php,app/Http/Controllers/SocialResponsibilityController.php,app/Http/Controllers/PrivacyController.php,lang/fa/about.php,lang/fa/social.php,lang/fa/privacy.php,tests/Feature/Pages/InfoPagesTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design about.html --route /about && node tools/shot.mjs --design social-responsibility.html --route /social-responsibility && node tools/shot.mjs --design privacy.html --route /privacy
---

# L3-08 — About, social responsibility and privacy pages

## Scope
- `/about` (`AboutPage` schema; team section — people data from a config/content file, photos via media later),
  `/social-responsibility`, `/privacy` (legal placeholders from `LegalSettings`, last-updated date shown and in
  schema `dateModified`).
- E-E-A-T: about page links to editorial/medical review policy section (copy from design; flag missing text).
- Audit corrections (`docs/AUDIT.md` §1, §8): add `/terms` (footer «شرایط استفاده» has no design page: same layout
  as privacy, content from a content file/settings) and the full privacy-policy text («خواندن متن کامل»), either as
  a section on `/privacy` or as `/privacy/policy` (decide and register in the static page registry + URL map).
  Donation block («از ریتمی حمایت کن»): no payment gateway exists (COD-only decision), so render it as a
  partnership/contact CTA and keep the amounts hidden until a gateway task exists. Transparency report link from
  settings (hidden when empty). Team/council data from a content file (placeholders in AUDIT §5.2).

## Acceptance
- Diff < 3% at 390/1440 for all three; `seo:audit` passes; tests green.
