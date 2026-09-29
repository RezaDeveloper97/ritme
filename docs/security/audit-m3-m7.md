# Security audit — M3–M7 (care reminders, checkups, fertility, pregnancy v2) + recent fixes

Date: 2026-09-29. Auditor: security-auditor subagent. Scope: `git diff ddfbe5d~1..HEAD` on `stage` (71 commits), read
from HEAD — backend-go, frontend, admin-web, `deploy.sh`, `deploy-stage.sh`, `frontend/Dockerfile`. Nothing under
`deploy/` changed in this range. Fixes tracked in T-M7-19.

**Counts:** Critical 0 · High 1 · Medium 2 · Low 2

## Findings

| # | Severity | File:line | Issue | Exploit scenario | Fix |
|---|---|---|---|---|---|
| 1 | **High** (health-data privacy) | `frontend/src/entities/checkup/model/attachments.ts:18-23`; `frontend/src/shared/session/cleanup.ts:51-58` | The `checkup-reports` IndexedDB store (lab photos/PDFs) never registers `onSessionEnd`; logout, account deletion and a trusted 401 leave every report on the device. Deleting a custom checkup type cascades its records server-side but leaves their on-device files orphaned (only single-record delete removes a file, `features/record-checkup/api/mutations.ts:176`). | On a shared or handed-down phone, whoever uses the device next (another account, DevTools, a profile backup) can read the previous user's lab reports. Contradicts T-M7-17 / §11.1 and what users expect from account deletion. | Add `clear()` to `createLocalFileStore`; register `onSessionEnd(() => checkupAttachments.clear())` in the checkup entity (like `entities/user/model/store.ts:204`); delete attachments of a custom type's records when it is deleted; test that logout empties the store. |
| 2 | **Medium** | `frontend/src/entities/checkup/model/attachments.ts:10` (`accept: ['image/*', 'application/pdf']`); `frontend/src/screens/checkup-history/ui/CheckupHistoryPage.tsx:105-106` | `image/*` admits `image/svg+xml`; "open attachment" does `window.open(URL.createObjectURL(blob))`, and a blob-URL document inherits the app origin, so script in an SVG runs as the app. The frontend CSP is Report-Only (`frontend/next.config.ts:120`). | A crafted "lab result" SVG, attached and later opened from history, can read `localStorage['ritme_token']` (365-day JWT) and act as the user. Requires the user to attach and open it; no server bug needed. | Explicit allow-list (jpeg, png, webp, heic, heif, pdf) in `matchesAccept` and the `<input accept>` (`MarkDoneSheet.tsx:290`); only open allow-listed types via blob URL, otherwise force download (`application/octet-stream`) or render images with `<img>`; move the CSP from Report-Only to enforced. |
| 3 | **Medium** (§11: no health data in URLs) | `frontend/src/widgets/checkups-card/model/counts.ts:39-41`; `frontend/src/screens/pregnancy-calendar/model/view.ts:38-47` | "Book appointment" links put the checkup title (user-typed for custom checkups) and, for pregnancy, `topic`, `care_item_key` and `date` in the query string of `/reminders/appointment/new`. App Router fetches the RSC payload for the full URL, so it reaches the Next server and nginx access logs, and browser history. | Anyone with log or device-history access sees per-request health context next to the client IP — e.g. that the user is pregnant and which prenatal test is due. | Pass prefill via memory/`sessionStorage` keyed by a random one-time id, or pass only an opaque catalog id and resolve the title client-side from cached queries. Never put free text or pregnancy markers in URLs. |
| 4 | **Low** | `frontend/src/shared/lib/ics/ics.ts:69-71` | `escapeIcsText` uses `'\;'`, which in JS is just `';'`, so semicolons aren't escaped; a bare `\r` isn't normalised (`/\r?\n/`). | The user's own title/notes/place can produce a malformed `.ics`, or inject extra properties into her own calendar file on parsers that split on bare CR. Self-injection only. | Use `'\\;'`, normalise `/\r\n?|\n/g` → `\\n`, strip other control characters, unit-test `;` and bare CR. |
| 5 | **Low** | `backend-go/internal/http/routes_care.go:24,31`; `backend-go/internal/http/routes_checkups.go:27,31` | New write endpoints (medications, appointments, custom checkup types, checkup records) have no per-user row cap or throttle; only `/auth/*` is throttled (`routes_auth.go:22-27`). | One authenticated account can script unlimited rows, growing tables and slowing `/care/today` and the checkup plan, which compute over all of a user's rows. | Per-user caps in the insert path (e.g. ≤100 active medications/appointments, ≤50 custom checkups) and a per-user rate limit (`platform/ratelimit` keyed by `user_id`) on `/api/v1/*` writes. |

## Checked, no finding

- **AuthN / route guards:** every new user route (`routes_care.go`, `routes_checkups.go`, `routes_fertility.go`,
  `routes_pregnancy.go`, `routes_pregnancy_v2.go`) uses `auth.MustGuard(...).RequireUser`; no public endpoints added.
- **IDOR / SQL scoping:** all queries in `db/queries/{care,checkups,fertility,pregnancy/v2_*}` read and write user rows
  filtered by `user_id`; catalog visibility is `user_id IS NULL OR user_id = ?`; `InsertIntake` and
  `UpsertCheckupSetting` run after an ownership/visibility check in Go; queries without `user_id` touch only admin
  content tables; admin catalog queries are pinned to `user_id IS NULL`; stats queries return aggregates only.
- **Account deletion:** all new per-user tables have `FOREIGN KEY (user_id) … ON DELETE CASCADE` (00002–00005).
- **SQL injection:** sqlc only, no string-built SQL.
- **Mass assignment:** request keys allow-listed (`care/appointment_request.go:14-19`, `pregnancy/v2/handlers.go:112-121`);
  length and array caps; doctor-report range capped (`MaxReportDays`).
- **Admin API:** new checkup and pregnancy routes go through `httpadmin.Handle` with `kit.Admin` (CSRF + session);
  admin-web's new screens use the shared `api` client.
- **Admin-authored links:** `sources.*.url` and tip links validated by `registry.IsLink`; alert `tel:` stripped to
  digits and `+`; external links `rel="noopener noreferrer"`.
- **XSS:** no `dangerouslySetInnerHTML`/markdown/HTML rendering in new frontend or admin-web code; admin icons map to a
  fixed path table.
- **PDF export:** text drawn on canvas, embedded as JPEG pages — no user text in PDF syntax.
- **On-device attachments:** only `has_attachment` reaches the API.
- **Session:** T-M7-17 `clearAuthToken → runSessionCleanups` clears outbox and onboarding store, no
  `localStorage.clear()`; gap is finding #1.
- **Logging:** no new `slog` calls log health payloads, OTPs or tokens; admin audit logs ids/counts only; no
  analytics/Sentry/console logging added.
- **Out of range, not re-audited:** OTP, rate limiting, JWT, CORS, uploads, `/storage`, nginx.
- **Secrets:** no keys, `.env` or htpasswd committed (test fixtures only).
- **Deploy:** `BUILD_REV` = `git rev-parse --short` + timestamp, only echoed in the Dockerfile; `deploy-stage.sh` ships
  `git archive` and excludes `.env` and `stage-gate.conf`; its checks assert the gate (401 without credentials, the
  `/admin` redirect doesn't mint the `ritme_stage` cookie).
