# Night & Bloom — screen → route map (B-N1-01)

All **183 unique artboards** (181 with a light file + `User_Library`/`User_Player`, dark-only). `L/D` = which theme
files exist. Owner = the bloom task that builds it (`bash bloom/bin/next.sh --all`).

**Kind:** `R` route · `V` state/variant of a route · `S` sheet (`?sheet=<id>`, `AppSheet`) · `P` panel inside a sheet ·
`A` admin-web · `I` instructor-web (new app) · `X` reference only, not built.
**Status:** `restyle` = existing route/screen gets the new look · `NEW` = does not exist yet · `—` = not built.

Totals: 55 restyle · 119 NEW · 9 reference-only. Routes are locale-agnostic (the `[locale]` prefix is added by the
i18n `Link`); today's routes live directly under `frontend/src/app/[locale]/` (there is no `(app)` route group yet).

Routing note: `frontend/CLAUDE.md` §4.1 still says "exactly five routed screens, everything else is a sheet", but the
app already routes checkups/reminders/fertility/pregnancy sub-screens. Night & Bloom screens with a back-button header
and their own scroll are **routes**; short decisions and pickers are **sheets**. B-N1-04 updates §4.1 to say so.

### a-start-plus — الف · شروع و اشتراک

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 1 | `Intro_1` | L+D | R | `/welcome (slide 1)` | restyle | B-N1-05 | widgets/intro-carousel |
| 2 | `Intro_2` | L+D | R | `/welcome (slide 2)` | restyle | B-N1-05 | widgets/intro-carousel |
| 3 | `Intro_3` | L+D | R | `/welcome (slide 3)` | restyle | B-N1-05 | widgets/intro-carousel |
| 4 | `Intro_4` | L+D | R | `/welcome (slide 4)` | restyle | B-N1-05 | widgets/intro-carousel |
| 5 | `Intro_5` | L+D | R | `/welcome (slide 5)` | restyle | B-N1-05 | widgets/intro-carousel |
| 6 | `Onb_Conditions` | L+D | R | `/onboarding/conditions` | restyle | B-N2-02 |  |
| 7 | `Onb_Cycle` | L+D | R | `/onboarding/cycle` | NEW (merges cycle-len, period-len, cycle-duration) | B-N2-02 | old routes redirect |
| 8 | `Onb_Gender` | L+D | R | `/onboarding/gender` | NEW | B-N2-02 | male → /onboarding/partner |
| 9 | `Onb_Goal` | L+D | R | `/onboarding/intention` | restyle | B-N2-02 | 4 goals incl. menopause |
| 10 | `Onb_Health` | L+D | R | `/onboarding/health` | NEW (merges birthday, height, weight) | B-N2-02 | old routes redirect |
| 11 | `Onb_Meno` | L+D | R | `/onboarding/menopause` | NEW | B-N2-02 |  |
| 12 | `Onb_Name` | L+D | R | `/onboarding/name` | restyle | B-N2-02 |  |
| 13 | `Onb_OTP` | L+D | R | `/otp` | restyle | B-N2-02 | 5 digits, WebOTP |
| 14 | `Onb_Partner` | L+D | R | `/onboarding/partner` | NEW | B-N4-05 | stub in B-N2-02 |
| 15 | `Onb_PartnerLinked` | L+D | R | `/onboarding/partner/linked` | NEW | B-N4-05 |  |
| 16 | `Onb_Phone` | L+D | R | `/signup` | restyle | B-N2-02 | + terms/health-data consent |
| 17 | `Onb_Preg` | L+D | R | `/onboarding/pregnancy-basis` | restyle | B-N2-02 |  |
| 18 | `Onb_Ready` | L+D | R | `/onboarding/setting-up` | restyle | B-N2-02 |  |
| 19 | `Onb_Welcome` | L+D | R | `/welcome (final card)` | restyle | B-N1-05 |  |
| 20 | `Prem_Checkout` | L+D | R | `/plus/checkout` | NEW | B-N2-07 | bank gateway only on web |
| 21 | `Prem_Manage` | L+D | R | `/plus/manage` | NEW | B-N2-07 |  |
| 22 | `Prem_Paywall` | L+D | R | `/plus` | NEW | B-N2-07 |  |
| 23 | `Prem_Plans` | L+D | R | `/plus/plans` | NEW | B-N2-07 |  |
| 24 | `Prem_Success` | L+D | R | `/plus/success` | NEW | B-N2-07 | gateway return URL |
| 25 | `Prem_TrialHome` | L+D | V | `/home (trial banner + ad slot)` | restyle | B-N2-08 | ad slot → B-N10-02 |
| 26 | `Prem_TrialSheet` | L+D | S | `?sheet=plus-trial` | NEW | B-N2-08 |  |
| 27 | `Splash` | L+D | R | `/splash` | restyle | B-N1-05 |  |
| 28 | `User_Library` | D | X | `→ /learn` | — | B-N8-03 | dark-only, superseded by Learn_Hub |
| 29 | `User_Player` | D | X | `→ /learn/[course]/[lesson]` | — | B-N8-03 | dark-only, superseded by Learn_Video/Audio |

### b1-cycle-log-analysis — ب۱ · سیکل، ثبت و تحلیل

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 30 | `An_Body` | L+D | R | `/analysis/body` | NEW | B-N3-09 |  |
| 31 | `An_Correlations` | L+D | R | `/analysis/correlations` | NEW | B-N3-09 |  |
| 32 | `An_Cycle` | L+D | R | `/analysis/cycle` | NEW | B-N3-09 |  |
| 33 | `An_Fertility` | L+D | R | `/analysis/fertility` | NEW | B-N3-11 | BBT confirmation detail |
| 34 | `An_Hub` | L+D | R | `/analysis` | NEW | B-N3-08 | cycle/teen/menopause hub |
| 35 | `An_Hub_Post` | L+D | V | `/analysis (postpartum)` | NEW | B-N5-07 |  |
| 36 | `An_Hub_Preg` | L+D | V | `/analysis (pregnancy)` | NEW | B-N3-12 |  |
| 37 | `An_Hub_TTC` | L+D | V | `/analysis (ttc)` | NEW | B-N3-11 |  |
| 38 | `An_Labs` | L+D | R | `/analysis/labs` | NEW | B-N3-10 | empty until B-N6-06 |
| 39 | `An_Monthly` | L+D | R | `/analysis/monthly/[ym]` | NEW | B-N3-10 |  |
| 40 | `An_Period` | L+D | R | `/analysis/period` | NEW | B-N3-09 |  |
| 41 | `An_PregWeight` | L+D | R | `/analysis/pregnancy-weight` | NEW | B-N3-12 |  |
| 42 | `An_Symptoms` | L+D | R | `/analysis/symptoms` | NEW | B-N3-09 |  |
| 43 | `Cycle_Calendar` | L+D | R | `/calendar` | restyle | B-N1-07 |  |
| 44 | `Cycle_EditPeriod` | L+D | S | `?sheet=edit-period` | NEW (replaces QuickEditSheet period edit) | B-N1-08 |  |
| 45 | `Cycle_History` | L+D | R | `/cycle` | restyle | B-N1-08 | leaves the bottom nav |
| 46 | `Cycle_Home` | L+D | R | `/home` | restyle | B-N1-06 | cycle mode |
| 47 | `Cycle_Home_During` | L+D | V | `/home (during period)` | restyle | B-N1-06 |  |
| 48 | `Cycle_Home_Near` | L+D | V | `/home (near period)` | restyle | B-N1-06 |  |
| 49 | `Cycle_Log` | L+D | R | `/log` | restyle | B-N3-03 | interim day log; v2 sheet supersedes |
| 50 | `Cycle_Phase` | L+D | S | `?sheet=phase` | restyle | B-N1-08 | screens/phase-details |
| 51 | `Cycle_Settings` | L+D | R | `/cycle/settings` | NEW | B-N1-09 |  |
| 52 | `Cycle_Symptoms` | L+D | R | `/cycle/symptoms` | NEW | B-N1-08 |  |
| 53 | `Log_Bleeding` | L+D | P | `log sheet · bleeding panel` | NEW | B-N3-03 |  |
| 54 | `Log_Contraction` | L+D | R | `/pregnancy/contractions` | NEW | B-N5-08 |  |
| 55 | `Log_Customize` | L+D | R | `/log/customize` | NEW | B-N3-04 |  |
| 56 | `Log_Day` | L+D | P | `log sheet · «ثبت با صدا» tab` | NEW | B-N3-05 | Plus |
| 57 | `Log_Feed` | L+D | R | `/children/[id]/feeding` | NEW | B-N5-07 |  |
| 58 | `Log_Kick` | L+D | R | `/pregnancy/kicks` | NEW | B-N5-08 |  |
| 59 | `Log_Measure` | L+D | P | `log sheet · weight/BBT/tests panel` | NEW | B-N3-03 |  |
| 60 | `Log_Pain` | L+D | P | `log sheet · pain panel + body map` | NEW | B-N3-03 |  |
| 61 | `Log_Sheet_Cycle` | L+D | S | `?sheet=log (full)` | NEW | B-N3-03 | FAB target (B-N1-04) |
| 62 | `Log_Sheet_Post` | L+D | S | `?sheet=log (postpartum)` | NEW | B-N3-06 |  |
| 63 | `Log_Sheet_Preg` | L+D | S | `?sheet=log (pregnancy)` | NEW | B-N3-06 | replaces /pregnancy/log entry |
| 64 | `Log_Taxonomy` | L | X | — (spec table, light only) | — | B-N3-01 | data spec, not a screen |

### b2-ttc-pregnancy-postpartum-child — ب۲ · باروری، بارداری و مادری

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 65 | `Main` | L+D | V | `/home (ttc)` | restyle | B-N1-13 | TTC tiles (M5) |
| 66 | `Preg_Baby` | L+D | X | `→ /pregnancy/weeks/[n]` | — | B-N1-14 | v1, superseded by PregFull_Week |
| 67 | `Preg_Home` | L+D | X | `→ /pregnancy` | — | B-N1-14 | v1, superseded by PregFull_Main |
| 68 | `Preg_Onboard` | L+D | X | `→ /pregnancy/onboarding` | — | B-N1-14 | v1; use as reference for the TTC→pregnancy prompt |
| 69 | `Preg_Symptoms` | L+D | X | `→ /pregnancy/log` | — | B-N1-14 | v1, superseded by PregFull_Log |
| 70 | `Preg_Timeline` | L+D | X | `→ /pregnancy/weeks` | — | B-N1-14 | v1; dark nb2_ file is EMPTY |
| 71 | `TTC_BBT` | L+D | R | `/fertility/bbt` | restyle | B-N1-13 |  |
| 72 | `TTC_Calendar` | L+D | V | `/calendar (ttc) — tab «باروری»` | restyle | B-N1-07 |  |
| 73 | `TTC_Insights` | L+D | R | `/fertility/insights` | restyle | B-N1-13 |  |
| 74 | `TTC_Log` | L+D | R | `/fertility/log` | restyle | B-N1-13 |  |
| 75 | `PregFull_Alerts` | L+D | R | `/pregnancy/alerts` | restyle | B-N1-14 |  |
| 76 | `PregFull_Calendar` | L+D | R | `/pregnancy/calendar` | restyle | B-N1-14 |  |
| 77 | `PregFull_Log` | L+D | R | `/pregnancy/log` | restyle | B-N1-14 |  |
| 78 | `PregFull_Main` | L+D | R | `/pregnancy` | restyle | B-N1-14 |  |
| 79 | `PregFull_Setup` | L+D | R | `/pregnancy/setup` | restyle | B-N1-14 |  |
| 80 | `PregFull_Week` | L+D | R | `/pregnancy/weeks (+ /[n])` | restyle | B-N1-14 | mode tab «بارداری» |
| 81 | `v15_AddChild` | L+D | R | `/children/new` | NEW | B-N5-05 |  |
| 82 | `v15_Children` | L+D | R | `/children` | NEW | B-N5-05 |  |
| 83 | `v15_Main` | L+D | R | `/postpartum` | NEW | B-N5-04 | postpartum «امروز» |
| 84 | `v15_MoodCheck` | L+D | R | `/postpartum/mood` | NEW | B-N5-04 | EPDS safety path |
| 85 | `v15_Recovery` | L+D | R | `/postpartum/recovery` | NEW | B-N5-04 |  |
| 86 | `v16_ChildHome` | L+D | R | `/children/[id]` | NEW | B-N5-05 | mode tab «کودک» |
| 87 | `v16_Growth` | L+D | R | `/children/[id]/growth` | NEW | B-N5-06 |  |
| 88 | `v16_Learn` | L+D | R | `/children/[id]/learn` | NEW | B-N5-06 |  |
| 89 | `v16_Milestones` | L+D | R | `/children/[id]/milestones` | NEW | B-N5-06 |  |
| 90 | `v16_Vaccines` | L+D | R | `/children/[id]/vaccines` | NEW | B-N5-06 |  |

### c-health-record — ج · سلامت و پرونده

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 91 | `Lab_Consent` | L+D | S | `?sheet=lab-consent` | NEW | B-N6-07 |  |
| 92 | `Lab_Intro` | L+D | R | `/labs` | NEW | B-N6-07 |  |
| 93 | `Lab_Marker` | L+D | R | `/labs/markers/[key]` | NEW | B-N6-07 |  |
| 94 | `Lab_Processing` | L+D | V | `/labs/[id] (processing)` | NEW | B-N6-07 |  |
| 95 | `Lab_Result` | L+D | R | `/labs/[id]` | NEW | B-N6-07 |  |
| 96 | `Lab_Upload` | L+D | R | `/labs/new` | NEW | B-N6-07 |  |
| 97 | `Lab_Verify` | L+D | R | `/labs/[id]/verify` | NEW | B-N6-07 |  |
| 98 | `Record_Export` | L+D | R | `/record/export` | NEW | B-N6-04 |  |
| 99 | `Record_Preview` | L+D | R | `/record/export/preview` | NEW | B-N6-04 |  |
| 100 | `Record_Summary` | L+D | R | `/record` | NEW | B-N6-03 |  |
| 101 | `Vitals_AddBP` | L+D | R | `/vitals/bp/new` | NEW | B-N6-02 |  |
| 102 | `Vitals_AddGlucose` | L+D | R | `/vitals/glucose/new` | NEW | B-N6-02 |  |
| 103 | `Vitals_AddHR` | L+D | R | `/vitals/heart-rate/new` | NEW | B-N6-02 |  |
| 104 | `Vitals_BPReport` | L+D | R | `/vitals/bp` | NEW | B-N6-02 |  |
| 105 | `Vitals_GlucoseReport` | L+D | R | `/vitals/glucose` | NEW | B-N6-02 |  |
| 106 | `Vitals_Hub` | L+D | R | `/vitals` | NEW | B-N6-02 |  |
| 107 | `v13_AddAppointment` | L+D | R | `/reminders/appointment/new (+/[id]/edit)` | restyle | B-N1-15 | «برای چه کسی» → B-N4-06 |
| 108 | `v13_AddChooser` | L+D | S | `?sheet=reminders-add` | restyle | B-N1-15 |  |
| 109 | `v13_AddMedication` | L+D | R | `/reminders/medication/new (+/[id])` | restyle | B-N1-15 | «برای چه کسی» → B-N4-06 |
| 110 | `v13_AppointmentDetail` | L+D | R | `/reminders/appointment/[id]` | restyle | B-N1-15 |  |
| 111 | `v13_Preg_Home` | L+D | V | `/pregnancy (reminders block)` | restyle | B-N1-14 |  |
| 112 | `v13_Reminders` | L+D | R | `/reminders` | restyle | B-N1-15 |  |
| 113 | `v14_CheckupDetail` | L+D | R | `/checkups/[id]` | restyle | B-N1-15 |  |
| 114 | `v14_Checkups` | L+D | R | `/checkups` | restyle | B-N1-15 |  |
| 115 | `v14_History` | L+D | R | `/checkups/history` | restyle | B-N1-15 |  |
| 116 | `v14_Main` | L+D | V | `/home (checkups card, ttc)` | restyle | B-N1-15 | widgets/checkups-card |
| 117 | `v14_MarkDone` | L+D | S | `?sheet=checkup-mark-done` | restyle | B-N1-15 |  |
| 118 | `v14_SelfExam` | L+D | R | `/checkups/self-exam` | restyle | B-N1-15 |  |

### companion-family — ریتمی همراه و خانواده

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 119 | `Hamdam_Access` | L+D | R | `/companions/new (step 2)` | NEW | B-N4-04 |  |
| 120 | `Hamdam_Children` | L+D | R | `/companions/new (step 3, spouse)` | NEW | B-N4-04 |  |
| 121 | `Hamdam_Done` | L+D | R | `/companions/new (done)` | NEW | B-N4-04 |  |
| 122 | `Hamdam_Home` | L+D | R | `/companion` | NEW | B-N4-05 | male companion home, own nav |
| 123 | `Hamdam_Invite` | L+D | R | `/companions/new (step 4)` | NEW | B-N4-04 |  |
| 124 | `Hamdam_List` | L+D | R | `/companions` | NEW | B-N4-04 |  |
| 125 | `Hamdam_RecordFor` | L+D | S | `inline AppSheet in med/appointment forms` | NEW | B-N4-06 |  |
| 126 | `Hamdam_Type` | L+D | R | `/companions/new (step 1)` | NEW | B-N4-04 |  |

### d-doctor-assistant — د · پزشک و مشاوره

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 127 | `v17_Booked` | L+D | R | `/services/bookings/[id]` | NEW | B-N7-05 |  |
| 128 | `v17_Chat` | L+D | R | `/services/bookings/[id]/chat` | NEW | B-N7-05 |  |
| 129 | `v17_DoctorProfile` | L+D | R | `/services/doctors/[id]` | NEW | B-N7-05 |  |
| 130 | `v17_Doctors` | L+D | R | `/services/doctors` | NEW | B-N7-05 |  |
| 131 | `v17_Main` | L+D | R | `/services` | NEW | B-N7-01 | placeholder in B-N1-04 |
| 132 | `v17_ReviewBook` | L+D | R | `/services/doctors/[id]/book` | NEW | B-N7-05 |  |
| 133 | `v18_AIChat` | L+D | R | `/assistant/chat/[id]` | NEW | B-N7-07 | SSE streaming |
| 134 | `v18_AssistantHub` | L+D | R | `/assistant` | NEW | B-N7-07 |  |
| 135 | `v18_DeptIntro` | L+D | R | `/assistant/[dept]` | NEW | B-N7-07 |  |
| 136 | `v18_Handoff` | L+D | R | `/assistant/chat/[id]/handoff` | NEW | B-N7-07 |  |
| 137 | `v18_History` | L+D | R | `/assistant/history` | NEW | B-N7-07 |  |
| 138 | `v18_Profile` | L+D | R | `/assistant/profile` | NEW | B-N7-07 |  |
| 139 | `v18_Summary` | L+D | R | `/assistant/chat/[id]/summary` | NEW | B-N7-07 |  |

### e-learning-instructor — هـ · آموزش

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 140 | `Ins_AddUser` | L+D | I | `instructor-web /students/add` | NEW | B-N8-07 |  |
| 141 | `Ins_Content` | L+D | I | `instructor-web /content` | NEW | B-N8-06 |  |
| 142 | `Ins_Course` | L+D | I | `instructor-web /courses/[id]` | NEW | B-N8-06 |  |
| 143 | `Ins_Dashboard` | L+D | I | `instructor-web /` | NEW | B-N8-06 |  |
| 144 | `Ins_Group` | L+D | I | `instructor-web /groups/[id]` | NEW | B-N8-07 |  |
| 145 | `Ins_Groups` | L+D | I | `instructor-web /groups` | NEW | B-N8-07 |  |
| 146 | `Ins_Students` | L+D | I | `instructor-web /students` | NEW | B-N8-07 |  |
| 147 | `Ins_Upload` | L+D | I | `instructor-web /upload` | NEW | B-N8-06 |  |
| 148 | `LearnEntry_HomeContinue` | L+D | V | `/home (learning card: continue)` | NEW | B-N8-03 |  |
| 149 | `LearnEntry_HomeNew` | L+D | V | `/home (learning card: new from instructor)` | NEW | B-N8-03 |  |
| 150 | `LearnEntry_TabOption` | L+D | X | — (rejected option B) | — | — | decision: Option A |
| 151 | `Learn_Audio` | L+D | V | `/learn/[course]/[lesson] (audio)` | NEW | B-N8-03 |  |
| 152 | `Learn_Course` | L+D | R | `/learn/[course]` | NEW | B-N8-03 |  |
| 153 | `Learn_Downloads` | L+D | R | `/learn/downloads` | NEW | B-N8-04 |  |
| 154 | `Learn_Hub` | L+D | R | `/learn («دوره‌های من», under من)` | NEW | B-N8-03 |  |
| 155 | `Learn_PDF` | L+D | V | `/learn/[course]/[lesson] (pdf)` | NEW | B-N8-03 |  |
| 156 | `Learn_Profile` | L+D | V | `/profile (learning rows)` | restyle | B-N8-03 | older Me variant; B-N1-10 owns the hub |
| 157 | `Learn_Unlocked` | L+D | R | `/learn/unlocked/[grant]` | NEW | B-N8-03 | push/notification target |
| 158 | `Learn_Video` | L+D | R | `/learn/[course]/[lesson] (video)` | NEW | B-N8-03 |  |

### f-tools — و · ابزارها

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 159 | `Todo_Add` | L+D | S | `?sheet=todo-add` | NEW | B-N6-08 |  |
| 160 | `Todo_Empty` | L+D | V | `/todo (empty)` | NEW | B-N6-08 |  |
| 161 | `Todo_Home` | L+D | R | `/todo` | NEW | B-N6-08 |  |
| 162 | `Todo_List` | L+D | R | `/todo/lists/[id]` | NEW | B-N6-08 |  |

### g-me-settings — ز · تب «من» و تنظیمات

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 163 | `Me_About` | L+D | R | `/profile/about` | restyle (?sheet=info) | B-N1-12 |  |
| 164 | `Me_Appearance` | L+D | R | `/profile/appearance` | NEW | B-N1-10 | no appearance sheet exists today |
| 165 | `Me_Hub` | L+D | R | `/profile («من»)` | restyle | B-N1-10 |  |
| 166 | `Me_Legal` | L+D | R | `/profile/legal` | restyle (?sheet=info) | B-N1-12 |  |
| 167 | `Me_Mode` | L+D | R | `/profile/mode` | NEW | B-N2-03 |  |
| 168 | `Me_Notifications` | L+D | R | `/profile/notifications` | NEW (screens/profile-notifications restyled) | B-N1-11 |  |
| 169 | `Me_Privacy` | L+D | R | `/profile/privacy` | NEW | B-N1-12 |  |
| 170 | `Me_Profile` | L+D | R | `/profile/account` | NEW (replaces ?sheet=personal) | B-N1-10 |  |
| 171 | `Me_Support` | L+D | R | `/profile/support` | NEW | B-N1-12 |  |

### h-admin — ح · پنل ادمین (وب)

| # | Artboard | L/D | Kind | Route / placement | Status | Owner | Note |
|---|---|---|---|---|---|---|---|
| 172 | `Admin_Challenges` | L+D | A | `admin-web /challenges` | restyle | B-N9-07 |  |
| 173 | `Admin_Config` | L+D | A | `admin-web /config` | NEW | B-N9-11 |  |
| 174 | `Admin_Engine` | L+D | A | `admin-web /engine` | NEW | B-N9-06 |  |
| 175 | `Admin_KPI` | L+D | A | `admin-web /kpi` | NEW | B-N9-08 |  |
| 176 | `Admin_MessageEditor` | L+D | A | `admin-web /messages/[id]` | restyle | B-N9-05 |  |
| 177 | `Admin_Messages` | L+D | A | `admin-web /messages` | restyle | B-N9-05 |  |
| 178 | `Admin_OKR` | L+D | A | `admin-web /okr` | NEW | B-N9-09 |  |
| 179 | `Admin_Overview` | L+D | A | `admin-web /` | restyle | B-N9-03 | shell B-N9-01 |
| 180 | `Admin_Roles` | L+D | A | `admin-web /roles (+ /audit)` | NEW | B-N9-02 | today: /admins |
| 181 | `Admin_Safety` | L+D | A | `admin-web /safety` | NEW | B-N9-10 |  |
| 182 | `Admin_UserDetail` | L+D | A | `admin-web /users/[id]` | restyle | B-N9-04 |  |
| 183 | `Admin_Users` | L+D | A | `admin-web /users` | restyle | B-N9-04 |  |
