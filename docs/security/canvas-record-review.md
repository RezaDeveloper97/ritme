# Security review — health record, sharing & emergency card (canvas REC epic, CB-REC-06)

Cross-task QA + security audit of the canvas «پرونده سلامت» epic (CB-REC-01 … CB-REC-05) on top of bloom's doctor
report (B-N6-04), on branch `stage` at `94577a4c`, with the fixes of CB-REC-06.

## Scope

- Backend (Go): `backend-go/internal/healthrecord/**` (record build, audiences, documents, report), 
  `internal/sharelinks/**` (7-day report links, 24h summary + doctor code, access log, summary documents),
  `internal/emergency/**` (owner card, lock screen, public link), `internal/files/**` (signed links, purposes),
  `internal/platform/httpx/logpath.go` (request-log collapsing), the related `db/queries/*`, routes and throttles in
  `internal/http/routes_*.go`, account deletion / orphan sweep.
- Frontend: `frontend/src/screens/record-*/**` (export, preview, shared report, `/d` doctor code, emergency),
  `entities/health-record/**`, `shared/lib/page-hide`, public route segments, `next.config` headers, the service
  worker.
- Edge: `deploy/vhost-web.inc`, `deploy/vhost-stage.inc`.

Method: code reading of every route, query and public view, plus unit / integration tests for each fix. Nothing was
run against stage or production.

## Findings

| ID | Severity | Where | Finding | Status |
|---|---|---|---|---|
| M1 | Medium | `frontend/src/app/[locale]/shared/report/[token]/page.tsx`, `screens/record-export/ui/SharedReportPage.tsx` | The server's QR path for a 24h summary is bloom's `/shared/report/<token>`. It kept the token in the address bar and kept the report after the tab was hidden. `/d/<token>` scrubs the token and drops the report on `pagehide`, but this path did neither. On a shared clinic PC, the back button or a bfcache restore showed the record again. | **Fixed**: when the opened report is `kind: summary`, the page scrubs to the token-less `/<locale>/d` and drops the report on `pagehide`, the same as `/d/[token]`. bloom's 7-day report links are unchanged. |
| M2 | Medium | `backend-go/internal/healthrecord/{service,pregnancies}.go` | The share audience (doctor report, 24h summary, emergency card) listed one neutral «ended» pregnancy row for each pregnancy loss. Without type or date, it still tells a third party that a loss happened. | **Fixed** (privacy default): `AudienceShare` gets `LossCount = 0`, and the loss count is not even queried. The owner's view is unchanged. **Decision 2026-10-12:** an explicit per-share opt-in `include_losses` (default off) on the 7-day report link and the 24h doctor code, stored on the link (goose 00090) and re-applied on every public read; the switch «شامل سابقه‌ی از دست دادن بارداری» is off by default. The emergency card and the companion scope never carry losses. The same question for the assistant's profile context belongs to bloom (in flight) and is open there. |
| L1 | Low | `deploy/vhost-web.inc`, `deploy/vhost-stage.inc` | Access-log exemption only covered `/<locale>/shared/report/` and `/<locale>/emergency/`. The `/d/<token>` page and locale-less paths were logged with the token. | **Fixed**: one regex location `^/(?:[A-Za-z-]+/)?(?:shared/report|d|emergency)/` with `access_log off`; proxy settings identical to the previous blocks. |
| L2 | Low | `deploy/vhost-stage.inc`, `internal/platform/httpx/logpath.go` | `/api/v1/files/{id}/download?exp=…&sig=…` (the signed links in a shared summary) was written to the nginx access log and the Go request log with the signature. | **Fixed**: `location ~ ^/api/v1/files/[0-9]+/download$ { access_log off; … }` with the same proxy settings as `/api/`. `/api/v1/files` added to `sensitivePrefixes`, so it is logged as `/api/v1/files/*` with no client IP. Covered by a test. |
| L3 | Low | `internal/sharelinks/summary.go` (`openPayload`) | Every open counted a view and wrote an access-log row. A reload loop could inflate the owner's count and bury real entries in «سابقه دسترسی». | **Fixed**: when the same (link, via, device, browser) was logged in the last 10 minutes, the open neither counts nor logs again (`RecentShareLinkView`, `ViewDedupWindow`). Covered by unit and integration tests. |
| L4 | Low | `internal/sharelinks/documents.go` (`publicDocuments`) | A summary's file ids were signed if the file belonged to the owner, whatever its purpose. A signed link could also outlive the share link by up to 5 minutes. | **Fixed**: only `files.PurposeRecordDocument` signs. The TTL is `min(DefaultLinkTTL, link expiry − now)`, and nothing is signed when less than 1 s is left. Covered by a unit test. |
| L5 | Low | sharelinks / emergency for teen accounts | There is no teen-specific sharing policy (age gate or guardian rules for doctor codes and the emergency card). | **Fixed** (decision 2026-10-12): a teen-mode account gets the emergency card always; doctor codes and 7-day links only with an active parent link (403 `teen_share_needs_parent`, before the Plus gate; `policy` in the sharing overview and the link list). |
| L6 | Low | `internal/emergency` | The public emergency card link has no expiry and is not cut by an admin block of the account. | **Open**: needs a product decision. |
| L7 | Low | `internal/emergency/service.go` (`Save`) | Partial PUTs ran read-modify-write without a lock, so two concurrent saves could drop each other's fields. | **Fixed**: `Save` runs in one transaction: ensure the row, then `SELECT … FOR UPDATE`, then update. Covered by a concurrency integration test. |

Other open items, which are not code findings:

- Stage `SHARE_CODE_PEPPER` must be a real secret (rotate it from the test value). In production a missing pepper
  turns the doctor code off (kill switch).
- At the production cutover, the nginx vhosts for `api.ritme.app` need the same access-log exemptions as stage:
  `/api/v1/shared-reports/`, `/api/v1/emergency-cards/` and `/api/v1/files/{id}/download`.

## Verified clean

- **IDOR**: every owner query filters by `user_id` in SQL. Documents, files, links, codes, access log and the
  emergency card were all checked.
- **Tokens**: 256-bit random tokens. Only their SHA-256 is stored.
- **Summary snapshot**: sealed under a key derived from the token. The database alone cannot read it.
- **Doctor codes**: stored as an HMAC under a server-side pepper. The code wraps the token.
- **File links**: HMAC-signed and short-lived (5–15 min).
- **Public rate limits**: reports 30/min. Codes 10 per 10 min, plus a global breaker. Emergency card 20/min. Public
  files 300/min. Public responses are `no-store`, `noindex` and `no-referrer`.
- **Caps**: the active-link and code caps are atomic, under the user-row lock.
- **Account deletion**: cascades to links, views, documents and the card, and removes the blobs. A 6-hour orphan
  sweep catches leftovers.
- **Loss data**: none in the timeline, the emergency card or the companion scope. With M2, also none in any share
  audience.
- **Request logs**: sensitive paths (`loss`, `shared-reports`, `emergency-cards`, `files`) are collapsed to
  `<prefix>/*`, with no client IP.
- **AI extraction**: the fake provider is refused in production. Consent is re-checked for each file. Names are
  redacted before the provider sees the text.
- **Frontend**:
  - The public segments are matched exactly.
  - The service worker never caches `/api`.
  - Shared-report queries use `gcTime: 0`.
  - `safeFileUrl` guards document links.
  - The QR code is generated on the same origin.
  - There is no `dangerouslySetInnerHTML`.
