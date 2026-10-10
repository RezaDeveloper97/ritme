# REC — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png` (`roadmap/bin/shot-board.sh`). Screens: 390 px, headless Chrome over
CDP against a worktree Go API :8251 + Next dev :3118. Colours are checked against the token map
(`docs/canvas-build/README.md` §4), not the board hex.

Test user **`09120004118`** «سارا» (created for CB-REC-04 via test OTP read from `ritme_dev`): onboarding name/gender/goal
= cycle, blood type O+, allergy «پنی‌سیلین», extras (1 surgery, 3 family-history rows), Plus trial, `ai_documents`
consent; documents 1 imaging «سونوگرافی بارداری» (AI read + confirmed), 2 prescription «نسخه متخصص زنان» (AI read,
needs review; a PNG file), 3 visit (manual), 4 hospital «سزارین» (2025-03-04 → 06), 5 imaging «ماموگرافی».

## CB-REC-04

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Rec_Home` | `/record` (bloom's screen + additions) | [home](rec/CB-REC-04/home.light.png) · [upload sheet](rec/CB-REC-04/upload.light.png) · [family history sheet](rec/CB-REC-04/extras.light.png) | [home](rec/CB-REC-04/home.dark.png) · [upload sheet](rec/CB-REC-04/upload.dark.png) | ✔ | Added under bloom's person hero: allergy card «حساسیت دارویی: …» + «روی کارت اضطراری نمایش داده می‌شود» with a switch (PUT extras flag), 3×3 category grid with counts (GET /health-record/categories; board's diseases/meds/pregnancy/vaccines tiles are bloom's section cards below, so the grid holds the document kinds + surgeries + family history + «همه اسناد»), «needs review» card (instead of the board's claim-waiting card), «همه اسناد» + «افزودن سند». Insurance links omitted (CB-INS not built); «کارت اضطراری» button omitted (CB-REC-03 frontend not built). Nav stays hidden on `/record` (bloom: back header). Surgeries / family-history tiles open edit sheets. |
| `nbl_Rec_Timeline` | `/record/timeline[?kind=]` | [fa](rec/CB-REC-04/timeline.light.png) · [en](rec/CB-REC-04/timeline.en.light.png) | [fa](rec/CB-REC-04/timeline.dark.png) | ✔ | Same order: back header «سوابق و اسناد», kind chips (همه · آزمایش · تصویربرداری · ویزیت · نسخه · بستری + «سایر»), Jalali month groups in cards, rows icon + title + «day · centre / doctor / بستری n روز / n مورد», trailing chevron or a badge, info note. Claim badges come only from `links` (none locally); «منتظر بررسی» / «در حال خواندن» badges added from review state. «+» header action opens the upload sheet; «بیشتر» loads the next cursor page. |
| `nbl_Rec_Doc` | `/record/documents/[id]` | [confirmed](rec/CB-REC-04/doc.light.png) · [review form](rec/CB-REC-04/review.light.png) · [manual](rec/CB-REC-04/manual.light.png) · [consent](rec/CB-REC-04/consent.light.png) · [en](rec/CB-REC-04/doc.en.light.png) | [confirmed](rec/CB-REC-04/doc.dark.png) · [review form](rec/CB-REC-04/review.dark.png) | ✔ | Preview card (photo via the signed link, PDF icon + size), «اطلاعات خوانده‌شده از سند / اگر اشتباه است، ویرایش کن» rows (GA merged «۱۲ هفته و ۳ روز», EDD in Jalali), «این سند کجا استفاده شده؟» rows, download + delete (with confirm). Added: review form (low-confidence hint, prescription rows), AI start / pending (polls) / failed / Plus (402 → manual form) states, «مشخصات سند» card + edit sheet, pregnancy dating card (explicit confirm sheet). «اشتراک با پزشک» omitted (CB-REC-03 frontend). Locally the photo preview falls back to the icon: the app CSP allows `img-src https:` only and the local API is http (works on stage/prod). Dating card not screenshotted (needs a pregnancy-mode user). |

## CB-REC-05

Screens: worktree Go API :8271 + Next dev :3120, persona 04 **`09900000004`** «سارا رضایی» (the CB-PRIV-01 user: O+,
پنی‌سیلین on the card, thyroid, contact «علی · همسر», insurance «تکمیلی» ••••4821). Test data: one 24h code (opened once by
code from an iPhone UA and once by token from a desktop UA → the access history), then revoked; the public emergency
link was created, rotated by the shot and then turned off — every token in these images is dead. File names never
carry a token (`*_token.*`).

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Rec_Share` | `/record/share` | [share](rec/CB-REC-05/fa_record_share.light.png) · [code sheet](rec/CB-REC-05/fa_record_share.code.light.png) | [share](rec/CB-REC-05/fa_record_share.dark.png) · [code sheet](rec/CB-REC-05/fa_record_share.code.dark.png) | ✔ | Board order: back header «چه کسی پرونده‌ات را می‌بیند؟», «پزشک» card (QR icon disc, «دسترسی موقت با کد», «ساخت کد»; live codes listed under it with «تا HH:mm», view count and «لغو کد»; button off at 5 live codes or when the server has no pepper), then bloom's «لینک ۷ روزه گزارش» row → `/record/export` (not duplicated). «بیمه» rows are informational (DECISIONS #12): the board's switches became status pills «فقط با تو» / «هرگز» + a note that Ritme sends nothing to an insurer. «همدم و خانواده»: companion rows «داروها و نوبت‌ها · بدون دسترسی به اسناد» → `/companions/{id}` (empty → «هنوز همدمی…» → `/companions`). «سابقه دسترسی»: only doctor-code / QR / 7-day-link opens (`via`, coarse device), no insurer rows. Code sheet: range 3/6/12 months, sections (all on), up to 10 own documents; the result (code `XXXX-XXXX` + on-device QR of `qr_url` or origin + `qr_path`, «یا وارد کردن کد در …/d») lives only in mutation memory and is dropped on close. |
| `nbl_Rec_Emergency` | `/record/emergency` | [card](rec/CB-REC-05/fa_record_emergency.light.png) · [with QR](rec/CB-REC-05/fa_record_emergency.qr.light.png) | [card](rec/CB-REC-05/fa_record_emergency.dark.png) · [with QR](rec/CB-REC-05/fa_record_emergency.qr.dark.png) | ✔ | CB-PRIV-01's `EmergencyCardView` (owner variant: full name, blood pill, حساسیت / بیماری / داروی دائمی / [وضعیت] / بیمه masked / تماس) with the QR inside the card as on the board (only right after the link is created — the token is shown once). Switches «نمایش روی صفحه قفل · بدون نیاز به رمز», «نمایش وضعیت بارداری» (+ «نمایش حساسیت روی کارت», the CB-REC-01 flag), shield note. Added: «اطلاعات کارت» rows → contact sheet (name, relation, phone 4–15 digits) and insurance sheet (label + exactly 4 digits), «لینک عمومی کارت» card (create / new link / copy / turn off, view count). QR keeps dark-on-light in dark mode (scanner polarity). |
| `nbl_Rec_Home` (entries) | `/record` | [home](rec/CB-REC-05/fa_record.light.png) | [home](rec/CB-REC-05/fa_record.dark.png) | ✔ | New row under CB-REC-04's actions: «کارت اضطراری» (QR icon) → `/record/emergency`, «اشتراک پرونده» (lock icon, the board's header lock → Rec_Share) → `/record/share`. |
| — (public) | `/d` | [code entry](rec/CB-REC-05/fa_d.light.png) | [code entry](rec/CB-REC-05/fa_d.dark.png) | ✔ | No board. Public `(web)` page (noindex, no-referrer, dynamic): bloom's public shell «خلاصه پرونده بیمار / فقط خواندنی», code field, «نمایش پرونده» → POST /shared-reports/code → bloom's paper (+ the chosen documents, blob download only), «کد دیگر». 404/422 → «کد درست نیست یا منقضی شده», 429 → wait. |
| — (public) | `/d/[token]` | [summary](rec/CB-REC-05/fa_d_token.light.png) | [summary](rec/CB-REC-05/fa_d_token.dark.png) | ✔ | Short alias of bloom's `/shared/report/[token]` (same screen; the server's QR path still points at `/shared/report/…`). A summary lists its documents under the paper. |
| `nbl_Rec_Emergency` (public) | `/emergency/[token]` | [card](rec/CB-REC-05/fa_emergency_token.light.png) | [card](rec/CB-REC-05/fa_emergency_token.dark.png) | ✔ | Minimal card (`variant="lock"`: first name, no insurance), title + «اطلاعاتی که صاحب کارت…», «جای ارزیابی پزشکی را نمی‌گیرد» note. No nav, no install prompt, noindex, no-referrer; 404 → «کارت پیدا نشد». |

## CB-REC-06 — epic QA

Run on `stage` at `5eb70d21` (with the CB-REC-06 security fixes): worktree Go API :8391 (own DB copy, migrated to 55) + Next dev :3132, fresh
test user «مریم» (09120006006, test OTP from the DB): onboarding steps → name/gender/goal/health, blood type A+, allergies
«پنی‌سیلین، لاتکس», one surgery + one family-history row, Plus trial, `ai_documents` consent. 390 px, light + dark, `fa`.
Security result: [docs/security/canvas-record-review.md](../../security/canvas-record-review.md) (no open high findings).
Live tokens never appear in file names or images' URLs; the DB copy and every code/link were throw-away and are dead.

Journey (all ✔):

1. **Record home** — hero, allergy card with the emergency-card switch, 3×3 category grid with counts (1 imaging, 1 visit, 1 prescription, 1 surgery, 1 family history, 3 files), «needs review» card, actions row incl. «کارت اضطراری» / «اشتراک پرونده».
2. **Upload** — PNG (prescription) + PDF (imaging, visit) uploaded as `record_document` files and attached to three documents; (sheet screenshots: CB-REC-04).
3. **AI extract (fake provider)** → `needs_review` with fields + 2 medication rows → review → **confirmed** (`review_state=confirmed`, «تأییدشده» on the detail page).
4. **Timeline** — Jalali month groups, kind chips, «منتظر بررسی» badge on the unreviewed imaging doc.
5. **Document detail** — confirmed extraction table + «مشخصات سند» + where-used + delete. The file preview shows «پیش‌نمایش در دسترس نیست» only because the local API's `APP_URL` has no port, so `safeFileUrl` rejects the signed link (dev-env artefact, not a bug; previews were fine in the CB-REC-04 run).
6. **Sharing** — 24 h code created (9-char code, QR path, 1 document picked); opened via the `/d` code API (POST /shared-reports/code, iPhone UA) → summary payload with the picked document; access history row «پزشک خلاصه پرونده را با کد دید · موبایل» and «۱ بار دیده شد»; `/d/<token>` opens the summary and **the token disappears from the address bar** (`/fa/d` after load; same for `/emergency/<token>` → `/fa/emergency`); IP/UA detail not stored.
7. **Emergency card** — edited (contact, insurance ••••4821, lock-screen on), public link created; `/emergency/<token>` answers `Cache-Control: no-store`, `X-Robots-Tag: noindex, nofollow`, `Referrer-Policy: no-referrer`.
8. **Lock screen** — with a device passcode set, «کارت اضطراری · بدون نیاز به رمز» opens only the minimal card (`GET /emergency-card/lock`).

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_Rec_Home` | `/record` | [home](rec/CB-REC-06/home.light.png) | [home](rec/CB-REC-06/home.dark.png) | ✔ | none |
| `nbl_Rec_Timeline` | `/record/timeline` | [timeline](rec/CB-REC-06/timeline.light.png) | [timeline](rec/CB-REC-06/timeline.dark.png) | ✔ | none |
| `nbl_Rec_Doc` | `/record/documents/[id]` | [confirmed](rec/CB-REC-06/doc.light.png) · [imaging](rec/CB-REC-06/doc_imaging.light.png) | [confirmed](rec/CB-REC-06/doc.dark.png) | ✔ | preview placeholder = local APP_URL artefact (above) |
| `nbl_Rec_Share` | `/record/share` | [share](rec/CB-REC-06/share.light.png) | [share](rec/CB-REC-06/share.dark.png) | ✔ | none |
| `nbl_Rec_Emergency` | `/record/emergency` | [card](rec/CB-REC-06/emergency.light.png) | [card](rec/CB-REC-06/emergency.dark.png) | ✔ | none |
| — (public) | `/d` code entry | [entry](rec/CB-REC-06/d_code.light.png) | [entry](rec/CB-REC-06/d_code.dark.png) | ✔ | automated typing into the code field did not enable the button in headless Chrome; the open-by-code path was verified through the API instead |
| — (public) | `/d/[token]` | [summary](rec/CB-REC-06/d_token.light.png) | [summary](rec/CB-REC-06/d_token.dark.png) | ✔ | token scrubbed from the URL |
| `nbl_Rec_Emergency` (public) | `/emergency/[token]` | [card](rec/CB-REC-06/emergency_public.light.png) | [card](rec/CB-REC-06/emergency_public.dark.png) | ✔ | token scrubbed |
| — (lock) | app lock → emergency | [lock](rec/CB-REC-06/lock.light.png) · [card](rec/CB-REC-06/lock_emergency.light.png) | [lock](rec/CB-REC-06/lock.dark.png) · [card](rec/CB-REC-06/lock_emergency.dark.png) | ✔ | none |

Observations (no ✘): with an app-lock set, the public `/d` page on the same device stays hidden behind the lock (pre-paint
lock attribute) — irrelevant for a doctor's own device. The public emergency card also carries blood type and the emergency
contact (CB-PRIV-01 scope), beyond the minimal CB-REC-03 description.
