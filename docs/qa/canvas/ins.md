## CB-INS-03

Worktree Go API :8341 + Next dev :3141 against `ins03_dev` (fresh goose 00082, user `09120000301`, 3 catalog insurers, policy with 5
coverages, 2 claims: one `docs_requested`, one `reviewing`). 390 px, fa, light + dark. Colours come from the app tokens, not the board hex.

| Board | Route | Light | Dark | Verdict | Fix / note |
| --- | --- | --- | --- | --- | --- |
| nbl_Ins_Home (`boards/nbl_Ins_Home.png`) | `/insurance` (policy exists) | `CB-INS-03/fa_insurance.home.light.png` | `CB-INS-03/fa_insurance.home.dark.png` | Matches: policy card, five actions, action card, remaining bars, recent claims, nav | Greeting «سلام سارا» omitted (no profile name read here); Rec_Home link not added |
| nbl_Ins_Home (empty state) | `/insurance` (no policy) | `CB-INS-03/fa_insurance.setup.light.png` | `CB-INS-03/fa_insurance.setup.dark.png` | New (no board): setup form + link to the info page; no bottom nav | — |
| nbl_Ins_Coverage (`boards/nbl_Ins_Coverage.png`) | `/insurance/coverage` | `CB-INS-03/fa_insurance_coverage.coverage.light.png` | `CB-INS-03/fa_insurance_coverage.coverage.dark.png` | Matches: deductible tiles, caps + usage, waiting notes, not-covered note | «دانلود متن کامل بیمه‌نامه» omitted: the API stores no policy document |

## CB-INS-06

Next dev :3146 with the API mocked at the Playwright network layer (4 centres: 2 direct, 2 reimbursed, one without coordinates). 390 px, fa, light + dark. No `NEXT_PUBLIC_NESHAN_KEY` in dev, so the map card is not drawn (list-only fallback, DECISIONS #16); the light shot also shows the PWA install banner.

| Board | Route | Light | Dark | Verdict | Fix / note |
| --- | --- | --- | --- | --- | --- |
| nbl_Ins_Centers (`boards/nbl_Ins_Centers.png`) | `/insurance/centers` | `CB-INS-06/fa_insurance_centers.centers.light.png` | `CB-INS-06/fa_insurance_centers.centers.dark.png` | Matches: back header, search, type chips, cards with direct/reimbursed badge + call/locate buttons, deductible note; map card appears only with a Neshan key | Distance (km) omitted: the API returns none and we never read her location; «نقشه» toggle omitted |
## CB-INS-05

Next dev :3151 with the `/api/v1/insurance/*` calls mocked at the Playwright network layer (no private DB copy; renewal questionnaire, 6 of 9 answers from record, one doc-requested claim). 390 px, fa, light + dark. Colours come from the app tokens, not the board hex.

| Board | Route | Light | Dark | Verdict | Fix / note |
| --- | --- | --- | --- | --- | --- |
| nbl_Ins_Health (`boards/nbl_Ins_Health.png`) | `/insurance/questionnaire` (section 2) | `CB-INS-05/fa_questionnaire.history.light.png` | `CB-INS-05/fa_questionnaire.history.dark.png` | Matches: section stepper + «بخش ۲ از ۴», yes/no rows with «از پرونده سلامت» tag, explanation card with record items, warning, continue | Board warning «پاسخ‌ها فقط برای بیمه‌گر ارسال می‌شود» replaced: nothing is sent; answers stay in her record and she can export a PDF |
| nbl_Ins_Health (review) | `/insurance/questionnaire` (last step) | `CB-INS-05/fa_questionnaire.review.light.png` | `CB-INS-05/fa_questionnaire.review.dark.png` | New (no board): per-section summary, edit links, confirm | Confirm disabled while a question is unanswered |
| nbl_Ins_Health (confirmed) | `/insurance/questionnaire` | `CB-INS-05/fa_questionnaire.done.light.png` | `CB-INS-05/fa_questionnaire.done.dark.png` | New: confirmation, on-device PDF export, link to status | PDF drawn on the device (shared/lib/pdf), never uploaded |
| nbl_Ins_Status (`boards/nbl_Ins_Status.png`) | `/insurance/status` | `CB-INS-05/fa_status.light.png` | `CB-INS-05/fa_status.dark.png` | Matches: questionnaire card with 4-row timeline, open claims | Steps are recorded by her (sheet with step chips + date); no insurer ETA («۲ روز کاری») since nothing is fetched |
## CB-INS-04

Next dev :3144 with the API mocked at the Playwright network layer (no DB copy, no Go server): 3 members, 5 claims (one `docs_requested`, one `reviewing`, two `paid`, one `rejected`), 2 record documents. 390 px, fa, light + dark. Colours come from the app tokens, not the board hex. The PWA install card was removed from the shots.

| Board | Route | Light | Dark | Verdict | Fix / note |
| --- | --- | --- | --- | --- | --- |
| nbl_Ins_Claims (`boards/nbl_Ins_Claims.png`) | `/insurance/claims` | `CB-INS-04/fa_claims.light.png` | `CB-INS-04/fa_claims.dark.png` | Matches: totals card, search, status chips, month groups, status pills | Add «+» in the header (board has none); self-track note added |
| nbl_Ins_ClaimNew (`boards/nbl_Ins_ClaimNew.png`) | `/insurance/claims/new` | `CB-INS-04/fa_claim-new.light.png` | `CB-INS-04/fa_claim-new.dark.png` | Matches: member chips, cost-type chips, centre/date/amount, document slots + «از پرونده», masked IBAN (only when the policy has one), sticky save | Nine cost types per API; camera lives in the picker sheet |
| nbl_Ins_ClaimDetail (`boards/nbl_Ins_ClaimDetail.png`) | `/insurance/claims/{id}` | `CB-INS-04/fa_claim-detail.light.png` | `CB-INS-04/fa_claim-detail.dark.png` | Matches: payable breakdown, missing-documents card with record picker, dot timeline she advances, appeal | Message card shows her own step note (no insurer sends one); detach/delete/step sheet added |

## CB-INS-07

Real end-to-end pass, no mocks: worktree Go API :8371 + frontend :3171 + admin-web :3172 against a private MariaDB copy `ins07_dev` (fresh goose to 00087, dropped afterwards). Users `09120007001` (the journey), `…002` (IDOR), `…003` (clean setup form); admin `super` plus an `analyst` admin. 390 px, fa + en, light + dark, RTL for fa. Screenshots: `docs/qa/canvas/ins/CB-INS-07/`. Colours come from the app tokens, not the board hex. The headless browser is my own Chrome via Playwright (the shared MCP browser is used by parallel agents). The Go integration tests could not run (the shared test stack was too loaded: `testdb` clone timed out after 120 s); `go test ./internal/insurance/...` unit tests pass.

### Journey (admin creates insurer + centre, then the user)

| # | Step | Result | Evidence |
| --- | --- | --- | --- |
| 1 | Admin-web: create insurer (code, site, phone), duplicate code, bad URL, bad code, centre (type, insurer, lat/lng), direct centre | PASS: lists refresh, 422s show under the field, no-permission role (analyst) gets the «نقش شما اجازه…» state | `admin_insurers.light`, `admin_insurer-validation.light`, `admin_centres.light/dark`, `admin_no-permission.light` |
| 2 | User sets up policy (insurer chip from the admin catalog, plan, number, validity, member, 7 coverages) | PASS: only `number_last4` stored in DB, months→days (3 → 90), Jalali dates | `fa_setup.light`, `fa_setup.filled.light` |
| 3 | New claim with a record document picked «از پرونده» | PASS: only her own documents offered (user B's document is not listed), payable estimate 20 % deductible = 400k / 1.6M | `fa_claim-new.*`, `fa_claim-new.picker.light` |
| 4 | Record steps: docs missing (note) → attach a second record document → paid (amount) | PASS: timeline, document list, status chip and note follow; paid shows 1,600,000 | `fa_claim-detail.docs-requested.light`, `fa_claim-step-sheet.light`, `fa_claim-detail.paid.light` |
| 5 | Coverage updates after «paid» | PASS: paraclinic used 1,600,000, remaining 28,400,000 (home bars + coverage page) | `fa_home.*`, `fa_coverage.*` |
| 6 | Questionnaire: start (renewal) → 4 sections incl. a «yes» with explanation → review → confirm | PASS: nothing is prefilled for a user with no record, review lists answers + explanation | `fa_questionnaire.review.light`, `fa_questionnaire.done.light` |
| 7 | Export PDF | PASS: downloaded on-device, no network request during export, Persian RTL page, footer says Ritme did not send it. (In headless Chrome `navigator.share` hangs the spinner; with `canShare` removed it downloads. Real mobile Chrome/Safari resolve the share sheet.) | `fa_questionnaire.pdf-export-1.png` |
| 8 | Status: record a step on the questionnaire; open claims list | PASS | `fa_status.*` |
| 9 | Centres: admin-created centres appear, direct vs reimbursed badges, call link, no map without a Neshan key | PASS | `fa_centers.*`, `en_centers.*` |
| 10 | Delete a claim | PASS: 200, then 404 | API |

### Matrix: every board screen, real data

| Board | Route | fa light | fa dark | en light | en dark | Verdict |
| --- | --- | --- | --- | --- | --- | --- |
| nbl_Ins_Home | `/insurance` | `fa_home.light` | `fa_home.dark` | `en_home.light` | `en_home.dark` | PASS |
| nbl_Ins_Coverage | `/insurance/coverage` | `fa_coverage.light` | `fa_coverage.dark` | `en_coverage.light` | `en_coverage.dark` | PASS |
| nbl_Ins_Claims | `/insurance/claims` | `fa_claims.light` | `fa_claims.dark` | `en_claims.light` | `en_claims.dark` | PASS |
| nbl_Ins_ClaimNew | `/insurance/claims/new` | `fa_claim-new.light` | `fa_claim-new.dark` | `en_claim-new.light` | `en_claim-new.dark` | PASS (icon fixed, see below) |
| nbl_Ins_ClaimDetail | `/insurance/claims/{id}` | `fa_claim-detail.light` | `fa_claim-detail.dark` | `en_claim-detail.light` | `en_claim-detail.dark` | PASS |
| nbl_Ins_Health | `/insurance/questionnaire` | `fa_questionnaire.light` | `fa_questionnaire.dark` | `en_questionnaire.light` | `en_questionnaire.dark` | PASS |
| nbl_Ins_Status | `/insurance/status` | `fa_status.light` | `fa_status.dark` | `en_status.light` | `en_status.dark` | PASS |
| nbl_Ins_Centers | `/insurance/centers` | `fa_centers.light` | `fa_centers.dark` | `en_centers.light` | `en_centers.dark` | PASS |

All 32 renders: `dir=rtl` for fa and `ltr` for en, `data-theme` matches, zero horizontal overflow, zero console errors or warnings, zero 4xx/5xx from the API. Persian text inside en pages is user/admin-entered data only (plan, insurer, member, centre names).

### Security and privacy

| Check | Result |
| --- | --- |
| User B GET/PUT/DELETE user A's claim, POST events, attach/detach documents | PASS: 404 each |
| B attaches A's record document to own claim / creates a claim with A's document or A's member id | PASS: 404 / 422 |
| B PUT policy with A's member id | PASS: id ignored, a new member row is created, A's member untouched |
| B GET/PUT/confirm/status on A's questionnaire | PASS: 404 each; B's list is empty |
| B's claims list only contains own claims | PASS |
| Unauthenticated policy request; user token on admin API | PASS: 401 / 401 |
| Validation: negative / huge / future-dated amount, bad cost type, bad step status, deductible 150 | PASS: 422 |
| Admin API reaches only insurers and centres (no policy/claim/questionnaire route) | PASS (code + routes) |
| Policy number kept as last 4 only | PASS: DB column `number_last4` = 7890, full number nowhere |
| API / Next / admin server logs: policy number, amounts, notes, answers, centre, names, mobile | PASS: 0 hits (request log is method/path/status only) |
| Frontend analytics for insurance screens | PASS: none emitted from insurance code; `analytics_events` stayed empty |
| Third-party requests while using questionnaire and PDF | PASS: none |
| Copy never says anything is sent to an insurer | PASS: every «send» string in fa/en is a negation or about her own sending (documents, her own step) |

### Found

| ID | Severity | Finding | Status |
| --- | --- | --- | --- |
| F1 | low (copy/UX) | «Save claim» button used the paper-plane `send` icon next to copy that says nothing is sent to the insurer | FIXED (`icon="check"`, `NewClaimPage.tsx`) |
| F2 | low | `PUT /insurance/policy` accepts a junk `iban` («abc» is stored as last4 `ABC`) and `members: []` / `coverages: []`; the UI never sends an IBAN and always sends a member, so no user path hits it | OPEN (follow-up: add an IBAN shape rule, require ≥ 1 member) |
| F3 | low | Admin validation messages use global attribute labels: `code` renders as «کد تایید» (the OTP label) and `website` as the raw word (admin insurers form). The message sits under the right field, so it reads wrong only in wording | OPEN (needs per-domain attribute labels in `form.Msg` / `form.Validate`, shared admin code) |
| F4 | info | Centres type chips have no «Other» chip (deliberate in `CentersPage.tsx`) and no «All»: she clears a filter by tapping the pressed chip again; centres typed `other` show only unfiltered | NOTE, no change |
| F5 | info | Status timeline shows a generic ETA «معمولاً ۷ روز کاری بعد از تکمیل مدارک» on the claim detail (static copy, not from an insurer) | NOTE, acceptable but could be reworded if legal asks |
| F6 | info | Not covered here: prefill «از پرونده سلامت» with a real record (no record data for the QA users; covered by INS-05 mocked), the Neshan map (no key), real-phone share sheet | NOTE |

A first policy setup run saved only one coverage row: that was my script firing `fill()` calls without awaiting. Re-run with sequential input saves all seven rows correctly, so it is not an app bug.
