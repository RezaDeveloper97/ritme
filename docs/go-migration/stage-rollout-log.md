# Staging rollout log (T-M2-25)

Environment: `stage.ritmeapp.ir` (stage branch @ f35a2c9), both backends on the staging MariaDB/Redis,
Laravel still running (instant rollback via `deploy/switch-go-route.sh stage <group> off`).

## Method
- Smoke account: user `09900000901` ("Go Smoke", staging only), Laravel-issued Passport token (tinker `createToken`),
  profile + 3 health logs created through Laravel before any flip.
- Smoke = 21 authenticated GETs × fa/en = 42 requests (content, reminders, health-logs, profile, pregnancy, cycle
  status/today/month full+calendar/period status+history, messages/daily, home, plus 404 paths).
  Baseline recorded with every group on Laravel; after each flip the same 42 requests were compared with the
  baseline (status + semantic JSON equality). Which stack answered = `X-Backend: go` header.
- Gate: staging Basic auth → `ritme_stage` cookie, then Bearer.

## Flips (2026-09-23T11:39Z)
| # | Group | Go-served requests (cumulative) | Diffs vs Laravel baseline | Result |
|---|---|---|---|---|
| 1 | content | 8 / 42 | 0 | ✔ on Go |
| 2 | reminders | 12 / 42 | 0 | ✔ on Go |
| 3 | healthlog | 16 / 42 | 0 | ✔ on Go |
| 4 | profile | 18 / 42 | 0 | ✔ on Go |
| 5 | pregnancy | 20 / 42 | 0 | ✔ on Go |
| 6 | cycle | 32 / 42 | 0 | ✔ on Go |
| 7 | messages | 34 / 42 | 0 | ✔ on Go |
| 8 | home | 36 / 42 | 0 | ✔ on Go |
| 9 | auth | 36 / 42 | 0 | ✔ on Go |

(auth routes themselves are POST-only and not part of the GET smoke; see below.)

## Auth checks (after `auth` → Go)
- Laravel queue drained before the flip: `jobs` = 0, `failed_jobs` = 0.
- Laravel-issued token keeps working on every group (smoke above ran with it after the auth flip: 0 diffs).
- `POST /api/v1/auth/refresh-session` → 200 from Go, `{"refreshed": false, "expires_at": "2027-09-23T15:04:10+03:30"}`
  (token too fresh to rotate — same rule as Laravel).
- Garbage token → 401 `{"message":"Unauthenticated.","error_code":"unauthenticated"}` from Go.
- Write through Go: `POST /api/v1/health-logs` → 201, `X-Backend: go`.
- Go-issued token accepted by Laravel: covered by `internal/auth/crossstack_int_test.go` (not re-run on stage — would
  need a real OTP/SMS login).

## Rollback test
`profile` group, same token: on Go → 200 `X-Backend: go`; `switch-go-route.sh stage profile off` → 200 from Laravel;
`on` again → 200 `X-Backend: go`. Each flip = `nginx -t` + graceful reload of the shared proxy.

## Not yet done
- Human smoke via the web app and the Android shell against staging (all groups on Go).
- 30-minute log watch per group and the 48 h soak: staging has almost no real traffic (1 real user), so the soak
  only means something if someone uses staging; started $NOW with all groups on Go.
- Admin host on staging (admin-web + `/api/admin/`): in preparation.
