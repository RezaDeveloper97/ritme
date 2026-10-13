# Canvas stage smoke (CB-REL-01)

| | |
|---|---|
| Date | 2026-10-13 |
| Environment | https://stage.ritmeapp.ir (compose project `ritme-stage`, Go backend) |
| Deployed commit | `f598a02d` (branch `stage`) |
| Goose version | 93 (`{"msg":"migrations","action":"goose_managed","applied":null,"version":93}`) |
| Containers | admin-web, backend-go, frontend, instructor-web, mysql, redis healthy |

## Method

- Gate: one Basic-auth page load minted the `ritme_stage` cookie; credentials stayed in a chmod-600 scratch file.
- Three fresh test users (`0990000200x`; OTP read from `otp_verifications`): U1 cycle user (non-TTC), U2 menopause, U3 teen.
  Admin API used only to seed a throwaway seller, product, city and listing (stage had no shop or directory data).
- API: every key endpoint per epic as the logged-in user (HTTP status plus a check of the response keys).
- Web: `bloom/bin/stage-smoke.mjs` (headless Chrome, 390 px, fa, light and dark; desktop 1440 px for the public web pages).
  Every captured page: no 4xx/5xx from the app and no console errors, except the expected teen 403 on `/shop`.
- Cleanup: the three users deleted with `DELETE /account`, seeded shop/directory rows deleted through the admin API,
  their OTP rows removed. Screenshots are kept out of the repo.

## Results

| Epic | Check | Result |
|---|---|---|
| CORE | `GET /catalog/teen_faq` and `meno_faq` without login | PASS 200 |
| CORE | `GET /catalog/meno_alerts` without login | PASS 401 (non-FAQ groups stay private) |
| CORE | `GET /catalog/pelvic_levels` logged in; `/languages`; `/home`; `/cycle/today` | PASS 200 |
| NAV | `GET /search?q=...` and `/fa/search`, `/fa/home`, `/fa/services`, `/fa/calendar` light + dark | PASS |
| MENO | profile, today, hot flash start + stop, scores, patterns, treatment, report, `/messages/menopause` | PASS 200/201 |
| MENO | `/fa/menopause/{stage,log,hot-flash,score,score/questionnaire,treatment,report,alert}`, `/fa/checkups` light + dark | PASS clean |
| IVF | start cycle, overview, meds, scans, tww | PASS 200/201 |
| IVF | `/fa/ivf`, `/ivf/meds`, `/ivf/scan`, `/ivf/tww` light + dark | PASS clean |
| LOSS | `GET /loss`, `/profile/life-stage/loss-copy`; `/fa/loss` light + dark | PASS (read only) |
| LOSS | creating a loss event | SKIP (changes life mode and pregnancy state; covered by CB-LOSS-03) |
| COND | hub, enrol endo and pmdd, save and read a pain day, pmdd chart and day, pbac day, `/messages/nudges` | PASS |
| COND | `/fa/programs`, `/programs/{pain,pmdd,bleeding}` light + dark | PASS clean |
| CONTRA | set pill method, log a pill, read back | PASS 200 |
| CONTRA | `/fa/contraception`, `/missed`, `/other` light + dark | PASS clean |
| PELV | get, start program, log session, diary day read and write | PASS 200 |
| PELV | `/fa/programs/pelvic`, `/pelvic/kegel` light + dark | PASS clean |
| PRIV | `/profile/notification-settings`, `/profile/consents`, `/fa/profile/privacy`, `/fa/record/emergency` | PASS |
| PRIV | public emergency card with an unknown token | PASS 404 |
| TEEN | profile, today, linked, `teen_kit` and `teen_faq` catalog | PASS 200 |
| TEEN | `/banners` empty for a teen; `/shop/catalog/home` 403 for a teen | PASS (commercial surfaces hidden) |
| TEEN | `/fa/teen/onboarding`, `/teen/parent`, `/fa/home` light + dark | PASS (only the expected shop 403) |
| REC | health record, timeline, categories, extras, emergency card | PASS 200 |
| REC | create a document with an uploaded file, read it, see it on the timeline, delete it | PASS 201/200 |
| REC | `/fa/record`, `/record/timeline`, `/record/share` light + dark | PASS clean |
| SHARE | create a 24 h doctor code (201); resolve it without login (`POST /shared-reports/code`, `GET /shared-reports/{token}`) | PASS 200 |
| SHARE | share-code page `/fa/shared/report/{token}` (redirects to the public doctor page `/fa/d`, 1440 px light + dark) | PASS 200 |
| SHARE | create a 7-day share link as a free user | PASS 402 (Plus gate, as designed) |
| FILES | `POST /files` PNG, `purpose=record_document` (encrypted storage) | PASS 201 |
| FILES | signed URL download | PASS 200 `image/webp` |
| INS | insurers, centres, policy save, create claim, claims list, claim detail, delete claim, questionnaires | PASS 200/201 |
| INS | `/fa/insurance`, `/claims`, `/coverage`, `/centers`, `/questionnaire` light + dark | PASS clean |
| VOICE | `POST /logs/voice` as a free user | PASS 402 `plus_required` (gate works) |
| VOICE | real recording parse | SKIP (needs a Plus user and the AI provider is `fake` on stage) |
| DIR | options, listings public and logged in, listing detail, booking request, cancel, my bookings, business application draft | PASS 200/201 |
| DIR | `/fa/services`, `/profile/bookings`, desktop `/fa/places`, `/places/1`, `/places/booked/1`, `/business` (to `/city-services/join`), `/business/join` light + dark | PASS clean |
| SHOP | catalog home, products, product detail, checklist, supply reminder, address, cart add, checkout preview | PASS 200/201 |
| SHOP | cash-on-delivery order end to end: place (201), list, detail (`placed`, 130000), cancel (`cancelled`) | PASS |
| SHOP | `/fa/shop`, `/shop/checklist`, `/shop/cart`, `/profile/orders`; desktop `/fa/store`, `/store/product/1`, `/store/cart`, `/store/checkout`, `/store/order/1` | PASS clean |
| EXPORT | `GET /profile/export` (keys: account, ai_usage, condition_programs, consents, contraception, cycle_histories, directory, exported_at, health_logs, insurance, ivf, life_profile, loss, menopause, notification_settings, pelvic_floor, pregnancy, profile, reminders, shop, support_reports) | PASS 200 |
| COMP | list, invite a spouse (fake SMS provider, nothing sent), audit, revoke; `/fa/companions` | PASS |
| PUBLIC | `/fa/places`, `/fa/business`, `/fa/store`, `/fa/business/join`, `/fa/emergency`, `/fa/signup` without login | PASS 200 |
| PUBLIC | `/robots.txt` (disallows `/*/d`, `/*/shared`, `/*/emergency`, booking, cart, checkout, order) | PASS 200 |
| PUBLIC | `/sitemap.xml` | SKIP 404 (no sitemap exists in the roadmap) |
| ADMIN | `/panel/login` 200, `/admin` 301 to `/panel/`, admin login and `auth/me`, catalog, users, shop sellers/shipments/returns/stock/slots/discount codes, city-service reviews and reports | PASS 200 |
| LOGS | `docker logs ritme-stage-backend-go-1 --since 2h`: 5xx count | PASS 0 (only 4xx from the validation and wrong-path probes of this run) |

Totals: 42 PASS, 0 FAIL, 3 SKIP (one result per table row).

## Known gaps

- `NEXT_PUBLIC_NESHAN_KEY` is not set on stage: the map falls back to a list on the directory and insurance centres screens.
- Kavenegar templates for the companion invite, directory and shop SMS are not registered yet. No SMS was sent
  (`COMPANION_SMS_PROVIDER=fake`).
- Clinical review is pending: `docs/canvas-build/clinical-review.md`.
- AI provider on stage is `fake`, so document extraction and voice parsing were not exercised against a real model.
- Stage has no directory or shop data of its own. This run seeded and removed throwaway rows; real content must be added in admin.
- `GET /profile/export` has no top-level keys for record documents, teen data or companions. Confirm that is intended.
- `/sitemap.xml` does not exist.
