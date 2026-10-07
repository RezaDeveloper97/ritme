---
id: B-N8-01b
title: Courses security fixes — phone enumeration, SMS flood caps, re-add cooldown, rename re-review, paging clamp
milestone: N8
type: backend
status: todo
depends_on: [B-N8-01,B-N8-08]
parallel_group: N8-A2
touches: [backend-go/internal/learning,backend-go/internal/telemed,backend-go/internal/platform/httpx,backend-go/db,backend-go/api]
skills: [new-endpoint,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N8-01b — Courses security fixes — phone enumeration, SMS flood caps, re-add cooldown, rename re-review, paging clamp

## Why

## Scope
Security audit of 6467a90b (B-N8-01) / 392b0793 (B-N7-02):
- H1 phone enumeration: `POST /instructor/v1/grants` response uniform for every phone (no `registered`, no
  active/pending summary); `student_name`/`registered`/`activated_at`/`progress_percent` hidden until the student has
  opened a course of that instructor (`learning_grants.seen_at`, migration **00055** + Laravel twin, or progress row);
  refuse grants to groups/courses with no published course; daily cap on new phones per instructor (e.g. 500).
- M2 outbox flood: cap rows at enqueue (instructor's last-24h pending+sent ≥ SMSPerInstructorPerDay → skip); total /
  daily grant cap; fair per-instructor pick in `ListDueSMS`.
- M3 revoke/re-add spam: no notice/SMS for revoked→re-added within 7 days per phone+instructor; STOP suppression list
  for non-user numbers (document how a STOP is recorded).
- L4 approved instructor renaming display_name/title → back to pending (or admin-approved draft).
- L5 `GetSMSJob` joins instructor status; revoked instructor → skipped.
- L6 `ListPendingGrantsByPhone` LIMIT 100; lazy claim under `context.WithTimeout`.
- L7 `httpx.PageParam` clamps huge pages (no overflow → panic); telemed list/reviews guard negative offsets.
- L8 telemed doctor photo: reject (422) instead of storing the original when re-encode fails (strip EXIF).

## Out of scope
- 

## Acceptance
- 
