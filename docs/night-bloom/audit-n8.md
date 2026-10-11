# N8 design-fidelity audit (B-N8-09)

Every N8 screen side by side with its artboard (light + dark, 390 px) after B-N8-01 … B-N8-08:
`docs/design/night-bloom/e-learning-instructor/nb{l,d}_Learn_*|LearnEntry_*|Ins_*`. Unlike B-N8-03/04/06/07 (empty
states / mock API) this run used **real data**: an own Go API (`ritme_n809` DB on the test MariaDB, migrations from the
API start, port 8809), an approved instructor with a published 4-lesson course (chapters incl. a time-locked one,
lessons video / audio / pdf with **real files uploaded through the tus media API**), a standalone video, a second
instructor's course, two groups, grants to persona phones (unlimited / 30 / 90 days / until, one expired, three
pending numbers), a student with progress, a fresh student for the «جدید» home card, an applicant and a pending
instructor, a super admin. Dev servers: frontend :3809, instructor-web :3812, admin-web :3811 (own `NEXT_DIST_DIR`).
Screens: `docs/qa/bloom/B-N8-09/screens/`; side-by-sides (board · before · after) for every fixed item:
`docs/qa/bloom/B-N8-09/side-by-side/`.

Data-driven differences (names, counts, dates, which groups exist, masked phones / «بدون پیشرفت ثبت‌شده» under the
H1 rule of B-N8-01b) are not deviations. Deliberate defaults (`bloom/QUESTIONS.md` #121–#126) are accepted: PDF in an
`<iframe>` instead of the board's painted page + thumbnail strip + zoom, device-only notes, no instructor-web
«ارسال پیامک دعوت», CSV instead of Excel, per-lesson visibility «به‌زودی». Boards draw a fake status bar and ↑ share
button — accepted (QUESTIONS #30). admin-web learning screens (B-N8-08) have no board: judged against the other admin-web
screens.

Severity: **high** = broken control / unreadable / misleading; **med** = visible departure from the board, a missing
or wrong control, the digit rules; **low** = polish. Totals: **high 0 · med 8 · low 11** — fixed: 7 med;
**1 med open (backend, not mine)**; **11 low open**.

## App — home entry, /learn, /learn/[course], players, unlocked, downloads

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| A0 | Learn hub (resume hero, tabs, per-instructor groups, progress bars, access days, expired dimmed + «برای تمدید…»), course page (hero, access pill, progress, «ادامه · درس n», chapters, locked-chapter note + date), video / audio / PDF players with real media, `/learn/unlocked/[grant]`, `/learn/downloads` (usage meter, empty state, privacy note), home «ادامه یادگیری» card: match in both themes. | — | — | match |
| A1 | Home «جدید · از مدرس» was a compact row card (icon + text + chevron); the board is a hero card — gradient art with the «جدید · از …» chip, play disc, big title, then a meta footer with «+n دوره دیگر». | med | `frontend/src/widgets/learn-home-card/ui/LearnHomeCard.tsx`, `globals.css` (end, `.lrn-home-new*`) | **Fixed** — hero card. `side-by-side/A1-home-new.{light,dark}.png`. |
| A2 | Course page had no per-lesson «download» arrow on the rows (the board has one on every lesson); the only download entry was inside the player. | med | `frontend/src/screens/learn-course/ui/LearnCoursePage.tsx` (`RowDownload`), `globals.css` (`.lrn-li`, `.lrn-row-dl`) | **Fixed** — arrow beside each unlocked, ready lesson (save / cancel with % / saved ✓ → remove), hidden when offline storage is unsupported. `side-by-side/A2-A3-course.*.png`. |
| A3 | A lesson the student had already finished still carried the «جدید» pill (e.g. lesson 1 with the ✓). | med | `LearnCoursePage.tsx` (`LessonRow`) | **Fixed** — pill only while not done. |
| A4 | Hub header had no «دانلودها» button (board: round download button at the start of the title row → `/learn/downloads`); the page was reachable only from «من». | med | `frontend/src/screens/learn/ui/LearnHubPage.tsx`, `globals.css` (`.lrn-hub-head.has-action`) | **Fixed.** `side-by-side/A4-hub.*.png`. |
| A5 | Video stage: speed button drew a 44×52 box instead of the board's small pill, and the clocks read right-to-left (current on the right, duration on the left) against an LTR scrubber that fills from the left — the board is `05:02 ▬▬ 12:00 1.25x ⛶` left to right. | med | `globals.css` (`.lrn-speed`, `.lrn-stage-bar`) | **Fixed** — visible pill inside a 44px hit area; bar is `direction: ltr`. `side-by-side/A5-video.*.png`. |
| A6 | Course course-wide «دانلود برای تماشای آفلاین» toggle card («۱۲ درس · ۳۱۰ مگابایت») not built; the per-lesson arrows (A2) cover the function. | low | `LearnCoursePage.tsx` | Open — needs a sequential multi-download queue + size estimate; polish. |
| A7 | Course / hub / video headers lack the board's «⋯» menu and the ↑ share button; the board's audio / video top-left sun and captions buttons are absent. | low | `screens/learn*` | Open — no actions behind them in the spec. |
| A8 | `/learn/unlocked/[grant]`: board shows the lock-screen notification peek at the top and a sparkle row; ours leaves the top empty. | low | `screens/learn-unlocked` | Open — polish. |
| A9 | Audio: board header shows the album title only; ours shows the course title and the lesson count line differently («جلسه ۲ از ۴»). | low | `screens/learn-lesson` | Open — copy polish. |
| A10 | `/learn/downloads` with downloaded items (rows, edit mode, in-sheet player) could not be shot: the store is per browser profile (IndexedDB) and each headless run starts clean. Empty state + usage meter verified in light/dark; the row UI was checked in B-N8-04. | low | — | Note — manual check on a device. |
| A11 | Local-dev only: playback fails when the frontend host differs from the API host (`localhost:3000` → `127.0.0.1:8020`, or any IP + port pair): the stream response carries `Cross-Origin-Resource-Policy: same-site` (B-N8-02b L3), Chrome then reports MEDIA_ELEMENT_ERROR «Format error». Production (`web.ritme.app` ↔ `api.ritme.app`) and stage (one origin) are same-site, so unaffected. Worked around here with a header-stripping proxy; browse local dev via the same host as the API. | low | `backend-go/internal/media` (not touched) | Note — document in `local-dev` skill if it bites. |

## instructor-web — login / apply / pending, dashboard, content, course editor, upload, groups, students, add students

390 px against the boards, 1280 px for the desktop rail (no desktop boards): login, OTP, apply, pending, dashboard,
content list + filter, course editor (chapters, scheduled unlock, lesson menu, access card), upload (resume,
destination, audience), groups list, group detail, students, add students — match in both themes apart from the rows
below.

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| I0 | Pages listed above. | — | — | match |
| I1 | `/students` had no search field (board: «جستجوی نام یا شماره» above the status tabs). | med | `instructor-web/src/screens/students/ui/StudentsScreen.tsx`, `messages/fa.json` (`students.search`, `emptySearch`) | **Fixed** — client-side filter over name / phone / target, Persian and Latin digits match each other. `side-by-side/I1-students.*.png`. |
| I2 | Every student / member row carries a full-width outlined «لغو دسترسی» button; the board rows are compact (status on the end, revoke hidden). Rows are about 2× taller. | low | `entities/student` (`RevokeButton`, `GrantRow`) | Open — move behind a «⋯» / swipe; polish. |
| I3 | Header name «دکتر مریم صادقی · متخصص زنان…» is ellipsised on 390; board has a short «نام · ماما». | low | `widgets/panel-shell/ui/PanelShell.tsx` | Open — data length; two-line wrap would fix. |
| I4 | Course editor is a superset of the board (chapter edit / delete buttons, per-lesson «⋯», «+ فصل جدید», status chips). | low | `screens/course` | Accepted — functional editor. |
| I5 | Unselected radios in upload «چه کسی ببیند؟» are rings; board draws a filled white dot. | low | `screens/upload` (`.radio` in globals.css) | Open — polish. |

## admin-web — `/learning/instructors`, `/learning/courses`, `/learning/courses/[id]`, `/learning/reviews`

No boards (B-N8-08 had no screenshots): consistent with the other admin-web screens in both themes — table / panel
chrome, status badges, tabs with counts, action buttons, pagination.

| # | Deviation | Sev | file:line | Resolution |
|---|---|---|---|---|
| M1 | Course detail printed a stale developer note under «درس‌ها»: «…پخش رسانه با خط لوله رسانه (B-N8-02) اضافه می‌شود» — the pipeline exists, and a task id is not user copy. | med | `admin-web/messages/{fa,en}.json` (`learning.courseDetail.lessonsNote`) | **Fixed** — «پیش‌نویس‌ها هم نمایش داده می‌شوند.» |
| M2 | Lesson durations in the course detail and the review queue were Latin («0:12», «1:05») among Persian digits. | med | `LearningCourseDetailScreen.tsx`, `LearningReviewsScreen.tsx` | **Fixed** — `toLocaleDigits` (the pure `formatDuration` and its test unchanged). |
| M3 | A moderator can approve / flag / unpublish a lesson but cannot **open its media** from the review queue or the course detail — approval is blind to the content. Needs an admin-scoped signed playback endpoint (`/api/admin/v1/learning/lessons/{id}/playback`) + a viewer in admin-web. | med | `backend-go/internal/admin/learning` (backend — out of this task's edit scope) | **Open — for the orchestrator** (B-N9-02 / a follow-up): endpoint + player; frontend follows. |
| M4 | Course / review tables show times as «۱۴۰۵/۰۷/۱۸ ۲۳:۵۱»; other admin screens print «۱۸ مهر ۱۴۰۵». | low | `LearningReviewsScreen.tsx` | Open — shared `formatDateTime` variant. |

## Environment notes

- `instructor-web/next.config.ts` and `admin-web/next.config.ts` gained `distDir: process.env.NEXT_DIST_DIR || '.next'`
  (same as `frontend`) so a dev server and `npm run build` can coexist; no behaviour change when unset.
- The OTP throttle (send-otp per IP) and `oauth_clients` (a personal-access client row for `users`) have to be set up for
  a fresh DB; `APP_URL` must point at the API origin, otherwise playback URLs are minted as `http://localhost/...`.
- Temporary DB `ritme_n809`, API on :8809, dev servers and `.next-n809` dirs removed at the end.
