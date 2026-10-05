---
id: L9-04b
title: Security follow-ups: POST newsletter confirm, private pending join photos, remember-me UI, PII-role MFA tests
milestone: L9
type: fullstack
status: done
depends_on: [L9-04,L9-02]
parallel_group: L9-C
touches: [resources/views/pages/blog/newsletter,app/Http/Controllers/Blog/NewsletterController.php,app/Domain/Directory/Join,app/Filament/Resources/Directory/JoinRequests,app/Providers/Filament/AdminPanelProvider.php,config/filament.php,routes/web.php,tests/Feature/Security,tests/Feature/Admin,tests/Feature/Blog,tests/Feature/Directory]
skills: []
verify: composer verify
---

# L9-04b — Security follow-ups: POST newsletter confirm, private pending join photos, remember-me UI, PII-role MFA tests

## Why
L9-04 (`docs/SECURITY.md`) deferred/accepted findings that need view or cross-context changes:
- F19: newsletter confirmation happens on GET (mail scanners/prefetch can confirm) → confirm page with a POST button.
- F17: join-request photos are public under `/media` before admin review → store pending uploads on a private disk,
  move them to the public media disk on approval (`ApproveJoinRequest`), serve to admins via a signed/authorised route.
- Remember-me is effectively disabled (F3 logs remember-cookie logins out) but the login form still shows the checkbox.
- `ADMIN_MFA_ROLES` default stays `super-admin` because admin tests for PII roles don't enrol TOTP.

## Scope
- POST confirm form (+ CSRF, page-cache safe), GET shows the button only; tests.
- Private pending disk for join photos + approval move + admin preview route; tests.
- Hide the remember-me checkbox in the Filament login (panel config / custom login page).
- Admin tests enrol TOTP for PII roles; default `ADMIN_MFA_ROLES` = super-admin,shop-manager,directory-manager,support.

## Acceptance
- Findings F17/F19 marked fixed in `docs/SECURITY.md`; `composer verify` green.
