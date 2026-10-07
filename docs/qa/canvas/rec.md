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
