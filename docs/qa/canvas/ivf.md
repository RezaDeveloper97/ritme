# IVF — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png` (`roadmap/bin/shot-board.sh`). Screens: 390 px, headless Chrome over
CDP (`bloom/bin/shot.mjs --token`) against a local Go API (:8121) + Next dev (:3101). Test user **`09900002021`**
(created for CB-IVF-02): life stage `ttc` + «IVF/IUI» on, cycle 1 (antagonist) started 1405-07-04, stimulation from
1405-07-05 (→ «روز ۷ تحریک» on 1405-07-11), two medicines (suppression 08:00, FSH 150 IU 20:00), next scan tomorrow
09:00, beta 1405-07-28; linked partner companion **`09900002022`** (meds + appointments view) so the reminder toggle
shows. Colours are checked against the token map (`docs/canvas-build/README.md` §4), not the board hex.

## CB-IVF-02

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbl_IVF_Home` | `/ivf` (`/home` redirects for ttc + «IVF/IUI») | [home](ivf/CB-IVF-02/fa_ivf.home.light.png) · [dose logged](ivf/CB-IVF-02/fa_ivf.logged.light.png) · [no cycle](ivf/CB-IVF-02/fa_ivf.start.light.png) · [en](ivf/CB-IVF-02/en_ivf.home.light.png) | [home](ivf/CB-IVF-02/fa_ivf.home.dark.png) · [no cycle](ivf/CB-IVF-02/fa_ivf.start.dark.png) · [en](ivf/CB-IVF-02/en_ivf.home.dark.png) | ✔ | Same hierarchy and copy: eyebrow «درمان ناباروری» + Lalezar «IVF · سیکل اول» + bloom egg disc; «مرحله فعلی» card with the outlined «روز ۷ تحریک» chip and CB-CORE-02 `StepTimeline` (6 stages, done = turquoise check, current = brand ring, titles/hints from catalog `ivf_stages`, a known future date such as the beta day under its step); «تزریق‌های امروز» rows (syringe disc turquoise when done / brand to do, «۰۸:۰۰ · زیرجلدی · ۱۵۰ واحد», «انجام شد» turquoise pill ↔ «ثبت» brand outline — tap logs / undoes via `POST`/`DELETE /ivf/meds/{id}/doses`); «نوبت بعدی» row (calendar bloom disc, «فردا ۰۹:۰۰», chevron → `/reminders/appointment/{id}`); companion card «همدمت هم در جریان باشد» + switch (`PUT /ivf/cycles/current {notify_companion}`), hidden without a linked companion. Nav: امروز · درمان (syringe) · + · خدمات · من. Deliberate differences: «برنامه» link and the «ثبت نتیجه سونو» / «دو هفته انتظار» pills are hidden until CB-IVF-03/04/05 add `/ivf/meds`, `/ivf/scan`, `/ivf/tww` (`IVF_SCREENS_READY` in `screens/ivf/model/home.ts`) — no 404 links; «درمان» opens `/ivf#ivf-doses` until then (`NAV_READY.ivfMeds`). Medicine titles are the names the user typed (the board shows class names). No board for the no-cycle / switch-off states: minimal `EmptyState` (start a cycle → `POST /ivf/cycles` at «آماده‌سازی»; switch off → link to `/profile/mode`). |
| `IA_Nav` (IVF row) | `/home` with «IVF/IUI» off | [ttc nav](ivf/CB-IVF-02/fa_home.flag-off.light.png) | — | ✔ | Flag off → the plain TTC nav (امروز `/home` · باروری · + · خدمات · من) and the cycle home; flag on → `/home` hands over to `/ivf`. Nav shown on `/ivf` (tab root, `app-nav.ts`). |
