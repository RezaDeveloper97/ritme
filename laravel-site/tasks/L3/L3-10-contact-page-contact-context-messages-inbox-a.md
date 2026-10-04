---
id: L3-10
title: Contact page + Contact context (messages inbox, anti-spam)
milestone: L3
type: fullstack
status: done
depends_on: [L3-01,L1-08]
parallel_group: L3-E
touches: [app/Domain/Contact,database/migrations,resources/views/pages/contact.blade.php,app/Http/Controllers/ContactController.php,app/Http/Requests/ContactRequest.php,app/Filament/Resources/ContactMessages,app/Notifications,tests/Feature/Contact]
skills: []
verify: composer verify && node tools/shot.mjs --design contact.html --route /contact
---

# L3-10 — Contact page + Contact context (messages inbox, anti-spam)

## Scope
- Form (name, email or phone, topic, message), FormRequest with Persian messages, honeypot + minimum fill time +
  rate limit (no external captcha), PRG redirect with flash; page cache bypassed for flash state; «اطلاعات سلامت
  ننویس» warning kept.
- `contact_messages` table + `SubmitContactMessage` action; queued admin notification (mail, log driver default).
- Filament inbox: unread/read/archived, reply-by-mail link, export CSV; `support` role.
- SEO: `ContactPage` + Organization `contactPoint`.

## Acceptance
- Valid submit stored + notification queued; spam signals rejected silently; diff < 3%; tests green.
