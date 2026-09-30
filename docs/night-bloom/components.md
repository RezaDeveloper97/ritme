# Night & Bloom — component inventory (B-N1-01)

Detected by markup patterns across the light artboards (dark files are structurally identical). Artboard names are
given without prefix (`Cycle_Home` = `nbl_Cycle_Home` + `nbd_Cycle_Home`). Primitives are built by **B-N1-03** in
`frontend/src/shared/ui`; tokens in [tokens.md](tokens.md).

| Component | Spec (from artboards) | Used in (examples / count) |
|---|---|---|
| **ScreenHeader** | row, padding `6px 16px 10px`; 44px round back button (`--surface-2`, 1px `--line`, 20px stroke-1.8 icon) at start, centred title 17/800 + optional subtitle 11.5/600 `--text-2`, optional 44px action (share, settings, filter) at end or 44px spacer | 152 artboards — every non-hub screen (back = «بازگشت», close = «بستن» on sheets) |
| **HubHeader** | date 12/700 `--text-2` over greeting 20/800; end side: 44px round buttons (`--brand-soft` bg) for notifications / settings | `Cycle_Home*`, `Main` (TTC), `PregFull_Main`, `v15_Main`, `Hamdam_Home` |
| **DateStrip** | 7 days, cell 42×54 radius 16, weekday 10.5/700 above; today = `--brand-fill` + `--on-brand`; dots for logged days | `Cycle_Home`, `_Near`, `_During`, `Prem_TrialHome`, `Main`, `v14_Main`, `LearnEntry_Home*`, `Cycle_Log`, `Log_Day` |
| **Card** | `--surface`, 1px `--line`, radius 24 (20 secondary), padding 14–18, no shadow | everywhere |
| **HeroCard** | Card with `--hero-tint` gradient + `--shadow-hero`; Lalezar number | home hero, `PregFull_Main` week, `Prem_*`, `An_Hub` top finding |
| **SectionTitle** | 15/800 + optional end link 12.5/800 `--brand-strong` («جزئیات», «همه») | all hubs |
| **PrimaryButton** | 54px, radius 27, `--brand-fill`, 15/800 `--on-brand`, `--shadow-cta`, full width | forms, onboarding, Plus, sheets |
| **SecondaryButton** | 54px ghost: 1.5px `--line-strong`, `--brand-strong` text; text-link variant «هنوز نه» | `Cycle_EditPeriod`, `Cycle_Home_Near`, onboarding |
| **TileButton** | 58px, radius 16, 1.5px `--line`, icon over label | `Cycle_Home_During` prompts, quick actions |
| **PillChip** | 40px, radius 20/14, `aria-pressed`; on = `--brand-fill` (single) or `--brand` 15% fill + 1.5px `--brand` border (multi); day-chip dashed `--period` for suggested days | 13: `Cycle_Log`, `Log_Sheet_Cycle`, `Log_Day`, `Log_Pain`, `Cycle_EditPeriod`, `Cycle_Calendar`, `Onb_Meno`, `Onb_Preg`, `v13_Add*`, `v14_MarkDone`, `Hamdam_Access`, `v17_DoctorProfile` |
| **SegmentedTabs** | `role=tablist` track `--surface-2`, 1px `--line`, radius 18, padding 4, gap 4; tab 40px radius 14, 13/800; selected `--brand-fill`/`--on-brand` | 22: `An_Hub`, `An_Body`, `An_Symptoms`, `Cycle_Calendar` (ماه/سال), `Cycle_Symptoms`, `Log_Feed`, `PregFull_Week`, `v16_Growth`, `v16_Vaccines`, `TTC_BBT`, `TTC_Calendar`, `Vitals_*Report`, `v13_Reminders`, `v14_Checkups`, `v14_History`, `v17_Doctors`, `v18_History`, `Ins_Content`, `Ins_Students`, `Learn_Hub`, `Me_Legal` |
| **NumberStepper** | 44px round −/+ (1.5px `--line`), value in Lalezar 30–40 + unit caption | 10: `Onb_Cycle`, `Onb_Health`, `Log_Measure`, `v15_Recovery`, `Preg_Symptoms`, `TTC_Log`, `Vitals_AddBP/Glucose/HR`, `v13_AddMedication` |
| **Switch** | `role=switch`, 48×30 (46×28 compact), on `--brand-fill`, knob `--shadow-knob` | 19: `Cycle_Settings`, `Cycle_Log`, `Log_Customize`, `Me_Notifications`, `Me_Privacy`, `Me_Appearance`, `Me_Mode`, `Prem_Manage`, `Onb_Ready`, `Record_Export`, `v13_*`, `v17_ReviewBook`, `v18_DeptIntro`, `v18_Profile`, `Learn_Course`, `v15_AddChild` |
| **Checkbox / TaskRow** | 20–22px, `role=checkbox`; challenge rows with progress «۵ از ۷» | `Cycle_Home*`, `Prem_TrialHome`, `Todo_*`, `Onb_Phone` consents |
| **StatusPill** | radius 999, 10.5–11/800, tinted bg (`--data-soft`, `--warm-soft`, `--period-soft`, `--brand-soft`) + deep text; «پلاس» lock pill in `--warm` | `An_Hub*` (منظم / طبیعی / پلاس), `Prem_Plans` (محبوب), `v14_CheckupDetail`, `v18_AIChat`, all admin tables |
| **PlusLock** | «پلاس» pill + blurred/locked card body | `An_Hub*`, `Log_Day`, `Log_Sheet_Cycle`, `Prem_Paywall`, `Admin_Users` |
| **ListRow / SettingsRow** | 52–56px, 36px icon circle (`--brand-soft`), label 13.5/700, value/chevron at end; grouped inside a Card with 1px dividers | `Me_*`, `Cycle_Settings`, `Record_Summary`, `v18_Profile`, `Hamdam_List` |
| **Accordion** | category row expands to chip grid; `aria-expanded` | `Log_Sheet_Cycle` (all categories), `Me_Support` (FAQ), `Todo_List` (done section) |
| **BottomSheet** | `role=dialog`, `--page` bg, radius 30–32 top corners, grip 40×5 `--line`, padding `10px 16px 130px`, scrim `--scrim` | 14: `Log_Sheet_Cycle/Preg/Post` (full), `Log_Bleeding/Pain/Measure` panels, `Cycle_EditPeriod`, `Cycle_Phase`, `Prem_TrialSheet`, `Lab_Consent`, `v13_AddChooser`, `v14_MarkDone`, `Hamdam_RecordFor`, `Learn_Unlocked`, `Todo_Add` — maps onto the existing `AppSheet` (half/full) |
| **FAB** | 56px circle `--brand-fill`, plus icon 24 stroke 2.4 `--on-brand`, `--shadow-fab`; centre of nav | 32 artboards with nav |
| **BottomNav** | floating glass pill — see [nav.md](nav.md) | 37 artboards |
| **ProgressSteps** | row of 4px bars radius 2 (`--brand-fill` done / `--line` todo) in the header | 16: all `Onb_*`, `Prem_Checkout`, `Prem_Plans`, `Cycle_Log`, `Learn_Video` |
| **ProgressRing / CycleRing** | SVG ring, phase arcs, Lalezar 72 centre («حلقه سیکل», «حلقه هفته‌های بارداری») | `Cycle_Home*` (11×), `PregFull_Main`, `v15_Main`, `Prem_Manage` |
| **InfoNote** | small footnote card `--surface-3`, 11.5/600 `--text-3`, icon; source line (FIGO, IOM, ADA, WHO) | every `An_*`, `Lab_*`, `Vitals_*` |
| **UrgentCard** | `--danger-soft` + call CTA (115) | `v15_MoodCheck`, `v18_AIChat`, `Log_Contraction`, `v17_Main` |
| **Charts** (SVG, `direction:ltr` plot) | line (BBT, weight, BP, glucose, growth P3–P97 band), bars (cycle lengths, feeds, kicks), heat strip (symptom × cycle day), stacked phase bar, donut; axis 9.5 `--text-3`, grid `--line`, today dot `--glow-data` | 75 artboards: `An_*`, `TTC_BBT`, `TTC_Insights`, `v16_Growth`, `Vitals_*Report`, `Lab_Marker`, `Cycle_History`, `Cycle_Symptoms`, `Record_Summary`, admin KPI/overview |
| **Table** | admin/desktop + record preview | `Admin_*` (1440 wide), `Record_Preview`, `Log_Taxonomy` |
| **AdSlot** | labelled «تبلیغ», sizes 342×170 / 166×166 / 342×76 | `Prem_TrialHome` (B-N10-02) |
| **Toast / status** | `role=status` inline banner | `Prem_TrialHome`, `Cycle_Home_During` |
| **Avatar** | circle with initial on `--avatar-grad` | `Hamdam_*`, `Learn_*`, `v17_*`, `Me_Hub` |
| **EmptyState** | illustration disc + title + CTA | `Todo_Empty`, `An_Labs` (no data), `v15_Children` |

Not drawn but required by the bloom acceptance (loading skeletons, error states) — B-N1-03 derives them from Card
geometry with `--surface-3` shimmer (static under `prefers-reduced-motion`).
