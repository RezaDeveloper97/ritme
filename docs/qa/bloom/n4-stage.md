# N4 stage e2e — B-N4-10 (two-account companion flow)

| | |
|---|---|
| Date | 2026-10-04 (13:50–14:25 Tehran) |
| Target | https://stage.ritmeapp.ir (staging only; production `/opt/ritme` / compose `ritme` not touched) |
| Commit deployed | `stage` branch `aa11509` ("chore(bloom): B-N6-05b AI security follow-up task"); local HEAD at the time of the e2e `4d97399d` |
| Deploy | `./deploy-stage.sh` → exit 0 (the orchestrator ran it before this e2e; this run deployed nothing) |
| Migration line | `{"level":"INFO","msg":"migrations","action":"goose_managed","applied":[27,28,29,30,31,32],"version":32}` |
| Backend logs during e2e | `ritme-stage-backend-go-1`, last 75 min (295 lines): **0 5xx, 0 panics, 0 WARN**. **1 `ERROR` at boot**: `PRIVATE_NOTE_KEY is not set … private notes are disabled (503)` (see B-5). 4xx were all expected: 17× 409 `/messages/mode` (companion account, by design B-N4-05), 5× 422 `/companions/accept` (my wrong-code probes), 3+1× 403 `/care/appointments|medications` (delegation refusals I triggered), 403 `/children/{id}` + measurements (spouse read-only probe), 404 `/pregnancy/profile` (pregnancy setup before the profile exists), 5× 422 `/logs/days` (my seed script sent a bad `mood` shape), one 401 burst on the signup page before the token existed. `ritme-stage-frontend-1` / `ritme-stage-admin-web-1`: 0 error lines. |
| Method | `bloom/bin/stage-smoke.mjs` helpers (gate cookie + OTP read at runtime over ssh, headless Chrome 390×844 @2x, `fa`, light + dark, PII masked before capture). Every flow used real UI clicks except setup data and the probes marked "API". Each capture's 4xx/5xx and console errors are in `n4-stage/summary.{json,md}` (109 captures: **0 console errors, 0 exceptions**; network only shows the expected 409 `/messages/mode`, 404 `/pregnancy/profile` and my one wrong-code 422). |
| Screenshots | `docs/qa/bloom/n4-stage/` (`<prefix>-<step>.<theme>.png`, 109 PNGs). Prefixes: `A-` woman A, `B-` man B, `S-` spouse + child, `D-`/`E-` extra male sign-ups, `ADM-` admin panel |

## Test users (all created today through public flows)

| | Number | Role | Notes |
|---|---|---|---|
| A «سارا» | `0990•••••61` | woman, cycle → pregnancy (1w4d) | onboarding via API (`/onboarding/steps/*`), 3 past periods (`POST /cycle/period` 06-29, 07-27, 08-24) + onboarding LMP 09-21; med «آهن», appointment «سونوگرافی» 28 Mehr; child «مینا» (`POST /children`) |
| B «علی» | `0990•••••49` | man (companion) | signed up in the UI (phone → OTP → name → gender male → partner code → linked → `/companion`) |
| C | `0990•••••87` | no account | recipient of the by-phone invite (fake SMS); invite revoked at the end |
| D, E «علی» | `0990•••••…` | men, no link | dark-theme male sign-up and the «بعداً وصل می‌شوم» path |

## Checks

Shot names are under `n4-stage/`, `.light.png` / `.dark.png`.

| Page / flow | Light | Dark | Result | Notes |
|---|---|---|---|---|
| Owner: companions list (empty) → «افزودن همدم» → type پارتنر | `A-code-01-list`, `A-code-02-type`, `-02b-type-picked` | `A-phone-01…02b` | PASS | |
| Owner: access step (cycle view, meds edit, appointments view) | `A-code-03-access` | `A-phone-03-access` | PASS | Default none on every section; pregnancy row shown |
| Owner: invite by code («فقط کد بساز») | `A-code-04-invite`, `-05-code`, `-06-done`, `-07-list-pending` | — | PASS | `POST /companions` 201 `invite:{code:<6>, expires_at:+24h, phone:null, sms_sent:false}`; card shows «پارتنر · در انتظار قبول» + 3 grant chips. See I-1 for «۲۵ ساعت» |
| Owner: invite by phone (fake SMS) | — | `A-phone-04…07` | PASS | `invite:{phone:"0990****387", sms_sent:true}` (masked, boolean). Backend log `Companion invite SMS (fake provider, not sent)` with masked number. UI «پیامک دعوت به … فرستاده شد» |
| Man B sign-up: phone + consents → OTP → name → gender «مرد» | `B-01…B-04` | `D-01…D-04` | PASS | Male goes straight to the partner-code step |
| Partner code step: wrong code → uniform error | `B-05-partner-empty`, `B-06-partner-wrong` | `D-05-partner-empty` | PASS | «این کد درست نیست یا منقضی شده است…» |
| Correct code → linked screen → «ورود به پنل همدم» → `/companion` | `B-07-partner-filled`, `B-08-linked`, `B-09-companion` | `B-09-companion` | PASS | Linked: «به سارا وصل شدی», today card (phase باروری, next period ~15 days). Panel shows only meds (ویرایش), appointments (فقط دیدن), cycle (فقط دیدن); «علائم … و بارداری … به اشتراک گذاشته نشده» |
| `/companion/links` | `B-10-links` | `B-10-links` | PASS | |
| Record-for picker: B adds a medication for A (meds edit) | `B-11-med-new`, `B-12-recordfor-sheet`, `B-13-med-form-for-sara`, `B-13b-confirm-sheet`, `B-14-med-saved` | same | PASS | Picker lists «خودم» / «سارا … اجازه ویرایش داروها را داده»; save asks to confirm «ثبت برای سارا»; `POST /care/medications` 201; the med shows up in A's `GET /care/medications`, not in B's |
| Appointments (view only): no picker, delegated create refused | `B-15-appt-new`, `B-16-appt-new-for-sara` | same | PASS | No «ثبت برای» row; `?for=<A>` falls back to self (I-3). API: `POST /care/appointments {for_user_id:A}` → **403 `companion_forbidden`** |
| Owner notification + audit | `A-linked-home`, `A-linked-notifications`, `A-linked-detail` | same | PASS | Inbox: «علی دعوت همدمی‌ات را پذیرفت», «علی یک دارو برایت ثبت کرد» ×2 (`type:companion`, links `/companions` / `/reminders`). Detail «فعالیت اخیر»: ثبت · داروها ×2, دیدن · نوبت‌ها/داروها/پریود. `GET /companions/audit` has invited/accepted/read/write/grants_changed, no payload |
| Owner companions list + privacy rows | `A-linked-companions`, `A-linked-privacy` | same | PASS | Privacy «دسترسی دیگران»: «ریتمی همراه — علی · داروها، پریود، نوبت‌ها», «مدیریت همدم‌ها — ۱ دعوت در انتظار قبول» |
| Grants change: meds → «نبیند» → B loses access at once | `A-grants-01-meds-none`, `B-after-meds-none-companion`, `B-after-meds-none-reminders_medication_new` | `B-after-meds-none-*` | PASS | Checked < 1 s after the save: `sections/meds` 403 `section_not_shared`, `GET /care/medications/130?for_user_id=A` 403, delegated create 403, home `meds:null`, picker gone from the med form. See B-4 for the list copy |
| A switches to pregnancy (mode page → `/pregnancy/setup` 4 steps) with no pregnancy grant → B's cycle view neutral (CMP-H1) | `A-preg-01-mode`, `-02-after-click`, `-03-setup`, `-04-step2`, `-05-step3/4`, `-06-home`; `B-pregnancy-companion` | `B-pregnancy-companion` | PASS | B: `cycle:{has_data:false, cycle_day:null, main_phase:null, days_late:null…}`, `pregnancy:null`, phase not «pregnancy»; panel says «سارا هنوز سیکلش را در ریتمی ثبت نکرده است» (I-4). No pregnancy hint in tips. See B-3 for the half-switched state |
| Revoke (detail → «حذف همدم» → «بله، حذف شود») → B empty state | `A-revoke-02-after` (dark only), `B-revoked-companion` | `A-revoke-01-confirm`, `A-revoke-02-after`, `B-revoked-companion` | PASS | `DELETE /companions/1` 200; B `sections/cycle` 404, home `has_partners:false` + `empty_state.enter_code`; B's panel shows code entry |
| Brute force / uniform refusal (API) | — | — | PASS | Random code, already-used code, code bound to another phone, malformed `ab` → all **422** with the same body (`invite_invalid`, same message). 5 failed attempts in total, well under the breaker limits |
| Spouse flow: wizard type همسر → children step | `S-01-access`, `S-02-children-step`, `S-03-code` | — | **FAIL (B-1)** | Children step is still the «بخش فرزندان به‌زودی می‌آید» placeholder with a disabled «افزودن فرزند», though A has a child and the API accepts `child_ids` |
| Spouse with a child (child shared via API `PUT /companions/3/children {child_ids:[1]}`) → B enters code on `/companion/links` | `S-04-b-code-entry`, `S-05-b-after-connect`, `B-spouse-companion`, `B-spouse-companion_links` | `B-spouse-*` | PASS (API) / not visible in UI yet (I-5) | API: link `family.shared_child_ids:[1]`, companion home `child:{count:1, items:[مینا, ۴ ماه و ۲۴ روز]}`, `GET /children` → `role:shared, can_edit:false`; `PUT /children/1` and `POST /children/1/measurements` → **403 `child_read_only`**. B's panel still shows «پروفایل کودک به‌زودی» (the child card UI is in B-N5-05, not on stage) |
| Owner spouse views | `S-06-a-companions`, `S-07-a-detail`, `S-08-a-privacy` | same | PASS | |
| Male with no link: «بعداً وصل می‌شوم» → ready → `/companion` empty state + Me hub | `D-07-companion-empty`, `D-08-me`, `E-06-later`, `E-07-after-ready` | `D-06-later`, `D-07-companion-empty`, `D-08-me`, `E-06-later` | PASS / see B-6, I-6 | |
| Admin `/panel/companions/tips` (+ pregnancy phase) | `ADM-companions_tips`, `ADM-companions_tips_phase_pregnancy` | same | PASS | |
| Admin `/panel/companions/links` | `ADM-companions_links` | same | PASS | 3 links (active spouse, invited partner, revoked partner), names `س•••`/`ع•••`, phones 2-digit mask, `grants_count` only, no codes / sections. Super-only not tested: the credentials file has only the super admin (B-N4-08b int tests cover it). See B-7 |

**Summary: 24 checks: 22 PASS (some with low findings), 1 FAIL (spouse children step UI), 1 PASS by API only (child visible to the spouse, UI pending in B-N5-05).** The release gates hold on stage: grants take effect on the next request, CMP-H1 neutral view, uniform 422 on bad codes, delegation refused without edit, revoke cuts access at once, spouse child read-only. No 5xx, no console errors, dark mode readable on every screen.

## Bugs / findings

| ID | Severity | Finding | Repro | Suspected file |
|---|---|---|---|---|
| B-1 | **Medium** | The owner can't share a child from the UI. The spouse wizard children step is still the B-N4-04 «به‌زودی» placeholder (disabled «افزودن فرزند»), and the companion detail page has no children editor. B-N5-02 made `POST /companions {child_ids}` and `PUT /companions/{id}/children` work. No queued task owns this: B-N5-05 touches only `screens/children*` / `entities/child`. | A (has child «مینا») → `/fa/companions/new` → همسر → ادامه → ادامه (`S-02-children-step.light`) | `frontend/src/features/invite-companion/ui/InviteCompanionFlow.tsx` `ChildrenStep`; `frontend/src/screens/companion-detail` |
| B-2 | Low | Male «ready» screen copy is the women's copy: «ریتمیت آماده است — با ریتمی ریتم بدنت را بهتر بشناس» for a companion account. | New male → «بعداً وصل می‌شوم» (`E-06-later.dark`) | `frontend/src/screens/onboarding-flow/ui/ReadyStep.tsx` / `messages/fa/onboarding.json` `flow.ready` |
| B-3 | Low | Mode page → «بارداری» → «ادامه» → setup step 1 «حالت بارداری رو روشن کن» already stores `stored_mode:pregnancy` (effective mode stays `cycle`). If the user leaves the wizard, she stays half-switched. A reload of `/pregnancy/setup` restarts at step 1 (wizard state is not kept). | A → `/fa/profile/mode` → بارداری → ادامه → step 1 button → reload → `GET /profile/life-stage` → `{mode:"cycle", stored_mode:"pregnancy"}` | `frontend/src/screens/pregnancy-onboarding` (activate at step 1) / `backend-go/internal/pregnancy` `Activate` |
| B-4 | Low | The «not shared» list on the companion home uses an English-style serial comma: «علائم و حال روزانه،‏ داروها، و بارداری با تو به اشتراک گذاشته نشده است». Persian lists don't put «،» before «و». | B after meds → none (`B-after-meds-none-companion`) | `frontend/src/screens/companion-home/ui/CompanionHomePage.tsx` (`Intl.ListFormat` / join) |
| B-5 | Low (env) | Stage backend boots with `ERROR PRIVATE_NOTE_KEY is not set … private notes are disabled (503)`. Not a companion issue, but the stage `.env` is missing a key, so the private-notes feature can't be tested on stage. | `docker logs ritme-stage-backend-go-1 \| grep PRIVATE_NOTE_KEY` | `/opt/ritme-stage/.env` + `.env.stage.example` (owner of private notes) |
| B-6 | Low (unconfirmed) | Right after «بعداً وصل می‌شوم» (ready screen still showing), a direct navigation to `/fa/companion` in the same tab landed on `/fa/onboarding/name`. The server already had `completed:true`. A fresh load or the «ورود به ریتمی» button works (`E-07-after-ready`). Probably a stale cached auth user in that tab. | D: partner step → «بعداً وصل می‌شوم» → wait 4 s → open `/fa/companion` | `frontend/src/features/auth` landing guard / onboarding cache |
| B-7 | Low | The admin links table calls the partner type «همراه», but the app calls it «پارتنر» (owner UI) and the product calls it «ریتمی همراه»/«همدم». Mixed naming for support staff. | `/panel/companions/links` (`ADM-companions_links.light`) | `admin-web/src/screens/companions` labels |
| I-1 | Info | The invite validity reads «تا ۲۵ ساعت دیگر» on a 24 h code: `hoursUntil` uses `Math.ceil`, and the test machine's clock was ~75 s behind the server. Real phones with a slow clock will see the same thing. `Math.round` (or clamping to 24) would fix it. | `A-code-07-list-pending.light`, `A-phone-05-code.dark` | `frontend/src/entities/companion/lib/grants.ts` `hoursUntil` |
| I-2 | Info | The invite response masks the phone as `0990****387` (last 3 digits). The admin overview shows only the last 2. Both are fine; noting the inconsistency. | | `backend-go/internal/companion` `MaskMobile` |
| I-3 | Info | `/reminders/appointment/new?for=<owner>` with a view-only grant quietly falls back to «self», with no message. That matches B-N4-06 (preselect only with edit). | `B-16-appt-new-for-sara` | — |
| I-4 | Info | CMP-H1 neutral copy «سارا هنوز سیکلش را در ریتمی ثبت نکرده است» is shown for an owner with months of data, now in pregnancy. It leaks nothing, but the wording isn't accurate. A neutral «سارا فعلاً خلاصه سیکلش را به اشتراک نمی‌گذارد» would be more honest. | `B-pregnancy-companion` | `frontend/src/screens/companion-home` / `messages/fa/companion-home.json` |
| I-5 | Info | The companion home child card and the shared-child page aren't on stage yet («پروفایل کودک به‌زودی»), though the API returns `child.items`. The uncommitted B-N5-05 work in the tree (`SharedChildren`) covers it; re-check in the N5 rollout. | `B-spouse-companion` | B-N5-05 |
| I-6 | Info | 409 `/messages/mode` on every companion screen is the designed nav-mode probe (B-N4-05), but it logs a 4xx for every companion page view. A `/profile/life-stage`-based probe would avoid the noise. | summary.md | `frontend/src/widgets/*/use-nav-mode` |
| I-7 | Info | A partner with `cycle: view` sees «فاز باروری» and the fertile-window copy. This is by design (onboarding says «پنجره باروری (اگر شریکت بخواهد)»), but there is no separate fertile toggle; the cycle grant implies it. Product call. | `B-09-companion` | — |
| I-8 | Info | Appointment views to a companion include `notes`, `with` and `location`. `docs/security/bloom-companion.md` says this is intended. | `GET /companions/links/{id}/sections/appointments` | — |

## Not covered

- Global failed-accept breaker (CMP-M3): deliberately not triggered. Accept throttles were not hit.
- Real SMS delivery (stage SMS provider is `fake`; only the `sms_sent`/masked `phone` shape and the log line were checked).
- Admin panel as a non-super admin (no such credentials on stage).
- Teen-owner guard (CMP-M1) and the `parent` link type (canvas CB-TEEN). Locale `en`. Android (never).
- Companion «leave» from the man's side (`DELETE /companions/links/{id}`).

## Test data left on stage

- A `0990•••••61`: now in **pregnancy** mode (LMP 1 Mehr, 1w4d), 3 past periods + onboarding LMP, meds «آهن» / «فولیک اسید» / «ویتامین د» (the last two recorded by B), appointment «سونوگرافی» (notes «یادداشت خصوصی تست», location «کلینیک تست»), child «مینا» (2026-05-10). Companions: #1 partner (revoked), #2 by-phone invite (revoked by me so the code shown in `A-phone-05-code` is dead), #3 **active spouse** B with child shared.
- B `0990•••••49`: male, linked to A as spouse.
- D / E: males with no links. C: no account (only a revoked invite bound to the number).
