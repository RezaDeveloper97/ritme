---
id: L3-08
title: About, social responsibility and privacy pages
milestone: L3
type: frontend
status: todo
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

## Acceptance
- Diff < 3% at 390/1440 for all three; `seo:audit` passes; tests green.
