/**
 * Domain types for **pregnancy v2** (M7, docs/pregnancy-v2/README.md) — the
 * Go-only `/api/v1/pregnancy/v2/*` group. They live beside the v1 types
 * (`./types`) until every screen has switched (T-M7-09…14); nothing here
 * changes a v1 shape.
 *
 * Conventions (same as the M3–M5 foundations):
 * - read models are camelCase; request bodies keep the API's snake_case,
 *   except the Log screen's {@link PregnancyDayInput}, which
 *   `features/track-pregnancy` maps (it has clear-vs-keep rules);
 * - dates are Gregorian `YYYY-MM-DD` at the boundary — display goes through
 *   `shared/lib/date` (§7). Where the server also sends a ready label
 *   (Jalali for `fa`), it rides beside the date as `…Label`;
 * - copy that an admin writes (week details, tips, alert texts, care plan)
 *   arrives already localized via `Accept-Language`, so it is a plain string.
 *
 * Privacy (§11): everything here is pregnancy health data — never put it in a
 * URL, a query string or a log line.
 */

// ── Enumerations (the API's wire values) ───────────────────────

/** The dating basis the Setup artboard's segmented control picks. */
export const DATING_SOURCES = ['lmp', 'ultrasound', 'manual'] as const;
export type DatingSource = (typeof DATING_SOURCES)[number];

export const CONFIDENCE_LEVELS = ['high', 'medium', 'low'] as const;
export type ConfidenceLevel = (typeof CONFIDENCE_LEVELS)[number];

/** The 9 toggles of the Log artboard, in artboard order (T-M7-03). */
export const DAY_SYMPTOMS = [
  'nausea',
  'vomiting',
  'fatigue',
  'headache',
  'back_pain',
  'breast_pain',
  'heartburn',
  'constipation',
  'spotting',
] as const;
export type DaySymptom = (typeof DAY_SYMPTOMS)[number];

export const SEVERITIES = ['mild', 'moderate', 'severe'] as const;
export type Severity = (typeof SEVERITIES)[number];

/**
 * Mood 1–5 (`pregnancy_daily_extras.mood`). 5 is the best day. The Log
 * artboard lists the faces best-first (خوب … سخت), so the UI renders
 * {@link MOODS_DISPLAY_ORDER}.
 */
export const MOODS = [1, 2, 3, 4, 5] as const;
export type Mood = (typeof MOODS)[number];
export const MOODS_DISPLAY_ORDER: readonly Mood[] = [5, 4, 3, 2, 1];

/** Water stepper bounds (Log artboard). */
export const WATER_MIN = 0;
export const WATER_MAX = 15;

/** The four message levels of the Alerts artboard, least to most pressing. */
export const ALERT_LEVELS_V2 = ['info', 'suggestion', 'follow_up', 'urgent'] as const;
export type AlertLevelV2 = (typeof ALERT_LEVELS_V2)[number];

/** Server-side alert actions (`POST /pregnancy/v2/alerts/{id}/actions/{action}`). */
export const ALERT_SERVER_ACTIONS = ['ack', 'add_to_visit_note'] as const;
export type AlertServerAction = (typeof ALERT_SERVER_ACTIONS)[number];

/** Visit stepper of the Calendar artboard (M3 appointment `meta.stage`). */
export const VISIT_STAGES = ['booked', 'done', 'result'] as const;
export type VisitStage = (typeof VISIT_STAGES)[number];

export const CARE_ITEM_KINDS = ['visit', 'test', 'scan', 'vaccine'] as const;
export type CareItemKind = (typeof CARE_ITEM_KINDS)[number];

/** A care-plan row's state (Calendar artboard: انجام شد / نوبت داری / رزرو). */
export const CARE_ITEM_STATES = ['done', 'booked', 'to_book'] as const;
export type CareItemState = (typeof CARE_ITEM_STATES)[number];

/** Where a week sits relative to today (Week chip strip: past / current / future). */
export type WeekRelation = 'past' | 'current' | 'future';

/** Tint of a week highlight's icon tile (Week artboard, tab «جنین»). */
export const HIGHLIGHT_TONES = ['brand', 'pink', 'teal'] as const;
export type HighlightTone = (typeof HIGHLIGHT_TONES)[number];

/**
 * Icon keys an admin can pick for a week highlight (T-M7-06/07). Each maps to
 * a glyph + default tint in `./v2` ({@link HIGHLIGHT_TONES}).
 */
export const HIGHLIGHT_ICONS = [
  'hand',
  'heart',
  'eye',
  'brain',
  'drop',
  'scale',
  'sparkle',
  'moon',
  'baby',
  'face',
] as const;
export type HighlightIcon = (typeof HIGHLIGHT_ICONS)[number];

/** Pregnancy spans 1–42 weeks in v2 (40 + two overdue weeks). */
export const V2_MIN_WEEK = 1;
export const V2_MAX_WEEK = 42;
/** The progress bar and «هفتهٔ N از ۴۰» are always out of 40. */
export const V2_TERM_WEEKS = 40;

// ── Shared pieces ──────────────────────────────────────────────

export interface DateRange {
  from: string;
  to: string;
}

export interface Confidence {
  level: ConfidenceLevel | null;
  /** Server label, e.g. «متوسط». The UI may use its own i18n off `level`. */
  label: string | null;
}

/** «۸ هفته و ۳ روز» — weeks + days since the dating anchor. */
export interface GestationalAgeV2 {
  weeks: number;
  days: number;
}

export interface WeekTask {
  key: string;
  text: string;
  done: boolean;
}

export interface WeekTip {
  week: number | null;
  title: string;
  body: string;
  readMinutes: number | null;
  /** In-app route or absolute URL for «درباره هفتهٔ N بیشتر بخون». */
  link: string | null;
}

// ── POST /pregnancy/v2/dating-preview ──────────────────────────

/** Request body — only the fields of the chosen `source` are sent. */
export interface DatingPreviewInput {
  source: DatingSource;
  lmp_date?: string;
  ultrasound_date?: string;
  ultrasound_weeks?: number;
  ultrasound_days?: number;
  manual_weeks?: number;
  manual_days?: number;
}

/** Setup step 3/3 (result). */
export interface DatingPreview {
  source: DatingSource | null;
  age: GestationalAgeV2;
  trimester: number | null;
  dueDate: string;
  dueDateLabel: string | null;
  /** Usual birth range (due date ± uncertainty). */
  range: DateRange | null;
  /** «بازهٔ معمول تولد: ۲۸ فروردین تا ۲۵ اردیبهشت.» (admin template, filled). */
  rangeLabel: string | null;
  uncertaintyDays: number | null;
  confidence: Confidence;
  /** Localized «مبنای این محاسبه …» sentence from `pregnancy_setup` templates. */
  basis: string | null;
  /** Admin-edited `pregnancy_setup/result` texts; null fields fall back to the bundle. */
  copy: SetupResultCopy;
}

/** `pregnancy_setup/result` copy (admin-editable); `confidence` keeps `{confidence}`. */
export interface SetupResultCopy {
  lead: string | null;
  suffix: string | null;
  dueLabel: string | null;
  confidence: string | null;
  primary: string | null;
  secondary: string | null;
}

// ── GET /pregnancy/v2/setup-copy ───────────────────────────────

/** `label` + `hint` of one dating source (`pregnancy_setup/source_<source>`). */
export interface SetupSourceCopy {
  label: string | null;
  hint: string | null;
}

/**
 * Admin-edited `pregnancy_setup` texts of the Setup screens, in the request
 * locale. Every null (or empty `benefits`) falls back to the bundled copy.
 * The result step reads `DatingPreview.copy` instead (filled templates).
 */
export interface SetupCopy {
  welcome: {
    title: string | null;
    body: string | null;
    benefits: string[];
    primary: string | null;
    secondary: string | null;
  };
  dating: { title: string | null; body: string | null };
  sources: Record<DatingSource, SetupSourceCopy>;
  history: { title: string | null; body: string | null; disclaimer: string | null; skip: string | null };
}

// ── GET /pregnancy/v2/today ────────────────────────────────────

/** One slide of the Today week carousel (prev / current / next). */
export interface WeekSummary {
  week: number;
  trimester: number | null;
  /** «۸ هفته و ۳ روز» for the current slide, «۹ هفته» for the others. */
  title: string | null;
  /** «جنین الان تقریباً به اندازهٔ یک تمشک است.» */
  sizeLine: string | null;
  illustrationKey: string | null;
  relation: WeekRelation;
}

export interface DueCard {
  date: string;
  dateLabel: string | null;
  daysLeft: number | null;
  range: DateRange | null;
}

export interface TrimesterSpan {
  trimester: 1 | 2 | 3;
  startDate: string | null;
  startLabel: string | null;
  /**
   * 0–100: where this trimester **starts** on the 40-week bar (0 / 32 / 70) —
   * not a fill. The fill of each segment comes from `trimesterFills`.
   */
  percent: number;
}

export interface PregnancyProgressV2 {
  week: number;
  /** 0–100 across the 40-week term. */
  percent: number;
  trimesters: TrimesterSpan[];
}

export interface NextVisit {
  /** M3 appointment id when booked; null when it is only a care-plan suggestion. */
  appointmentId: number | null;
  careItemKey: string | null;
  title: string;
  date: string | null;
  dateLabel: string | null;
  /** `HH:MM` — null for an unbooked suggestion. */
  time: string | null;
  week: number | null;
  daysUntil: number | null;
  stage: VisitStage | null;
}

export interface PregnancyToday {
  date: string;
  age: GestationalAgeV2;
  trimester: number | null;
  confidence: Confidence;
  uncertaintyDays: number | null;
  carousel: WeekSummary[];
  due: DueCard | null;
  progress: PregnancyProgressV2;
  nextVisit: NextVisit | null;
  tip: WeekTip | null;
  tasks: WeekTask[];
  unreadAlerts: number;
}

// ── GET /pregnancy/v2/weeks/{n} ────────────────────────────────

export interface WeekHighlight {
  icon: HighlightIcon | null;
  tone: HighlightTone;
  title: string;
  body: string;
}

export interface WeekSource {
  title: string;
  url: string | null;
}

/** Admin-defined week content (`pregnancy_week_details`, T-M7-01/06). */
export interface WeekDetails {
  sizeLabel: string | null;
  illustrationKey: string | null;
  /** Text ranges on purpose («~۱.۶», «۱۵۰ تا ۱۷۰») — shown as-is. */
  length: string | null;
  weight: string | null;
  heartRate: string | null;
  headline: string | null;
  highlights: WeekHighlight[];
  bodySymptoms: string[];
  bodyText: string | null;
  warning: string | null;
  reviewerName: string | null;
  reviewedAt: string | null;
  sources: WeekSource[];
}

export interface PregnancyWeek {
  week: number;
  trimester: number | null;
  relation: WeekRelation;
  range: DateRange | null;
  rangeLabel: string | null;
  bookmarked: boolean;
  details: WeekDetails;
  /** Week tasks merged with the user's done state. */
  tasks: WeekTask[];
  tip: WeekTip | null;
}

/** PUT /pregnancy/v2/weeks/{n}/state — every field optional. */
export interface WeekStateInput {
  bookmarked?: boolean;
  done_task_keys?: string[];
}

export interface WeekState {
  week: number;
  bookmarked: boolean;
  doneTaskKeys: string[];
}

// ── GET/PUT /pregnancy/v2/days/{date} ──────────────────────────

export type DaySymptoms = Partial<Record<DaySymptom, Severity>>;

export interface LastWeight {
  value: number;
  date: string;
}

export interface PregnancyDay {
  date: string;
  week: number | null;
  mood: Mood | null;
  symptoms: DaySymptoms;
  waterGlasses: number | null;
  /** Kilograms, one decimal. */
  weight: number | null;
  visitNote: string | null;
  /** Latest weight before this day, for «آخرین ثبت: …». */
  lastWeight: LastWeight | null;
  /** Alerts this save raised (empty on GET). */
  alerts: PregnancyAlertV2[];
}

/**
 * What the Log screen saves (camelCase; `features/track-pregnancy` turns it
 * into the PUT body). A key that is present is written — `null` clears it
 * (T-M7-03 §3); an absent key leaves the stored value alone. `symptoms`
 * replaces the day's whole symptom set.
 */
export interface PregnancyDayInput {
  mood?: Mood | null;
  symptoms?: DaySymptoms;
  waterGlasses?: number | null;
  /** Kilograms. */
  weight?: number | null;
  visitNote?: string | null;
}

// ── GET /pregnancy/v2/calendar?month= ──────────────────────────

export interface CalendarDayMarker {
  date: string;
  hasVisit: boolean;
  /** Pregnancy week that starts on this day, else null. */
  weekStart: number | null;
  isToday: boolean;
}

export interface CalendarVisit {
  appointmentId: number | null;
  careItemKey: string | null;
  title: string;
  date: string;
  dateLabel: string | null;
  time: string | null;
  week: number | null;
  /** «سه‌ماههٔ اول · هفتهٔ ۱۲ از ۴۰» (server label). */
  weekLabel: string | null;
  /** Age on the visit day, «۱۱ هفته و ۱ روز» (server label). */
  ageLabel: string | null;
  stage: VisitStage | null;
  prep: string | null;
  doctor: string | null;
  place: string | null;
  remindBefore: string | null;
  resultNote: string | null;
  daysUntil: number | null;
}

export interface CareItem {
  key: string;
  title: string;
  kind: CareItemKind | null;
  weekFrom: number | null;
  weekTo: number | null;
  window: DateRange | null;
  state: CareItemState;
  /** Date of the linked visit, or the suggested date when unbooked. */
  date: string | null;
  dateLabel: string | null;
  suggestedDate: string | null;
  appointmentId: number | null;
  prep: string | null;
}

export interface PregnancyCalendar {
  /** `YYYY-MM` as requested. */
  month: string;
  monthLabel: string | null;
  /** Pregnancy weeks this month spans («هفته‌های ۸ تا ۱۲»). */
  weekRange: { from: number; to: number } | null;
  days: CalendarDayMarker[];
  visits: CalendarVisit[];
  nextVisit: CalendarVisit | null;
  carePlan: CareItem[];
  sourceNote: string | null;
}

// ── GET /pregnancy/v2/alerts ───────────────────────────────────

export interface AlertActionV2 {
  /** `ack` / `add_to_visit_note` post to the server; anything else is a deep
   *  link key the screen maps (e.g. `log_weight` → the log, weight focused). */
  key: string;
  label: string | null;
}

export interface AlertContact {
  text: string;
  phone: string | null;
}

export interface PregnancyAlertV2 {
  id: number;
  ruleKey: string | null;
  level: AlertLevelV2;
  title: string;
  whatWeSaw: string | null;
  howSure: string | null;
  advice: string | null;
  actions: AlertActionV2[];
  contact: AlertContact | null;
  createdAt: string | null;
  /** `YYYY-MM-DD` the fact happened (week start, streak end), else the creation day. */
  factDate: string | null;
  dateLabel: string | null;
  isRead: boolean;
  isAcked: boolean;
}

export interface AlertLegendRow {
  level: AlertLevelV2;
  label: string | null;
  text: string | null;
}

export interface PregnancyAlertsV2 {
  windowDays: number;
  alerts: PregnancyAlertV2[];
  legend: AlertLegendRow[];
}

// ── GET /pregnancy/v2/report?from=&to= ─────────────────────────

export interface ReportDay {
  date: string;
  week: number | null;
  mood: Mood | null;
  symptoms: DaySymptoms;
  waterGlasses: number | null;
  visitNote: string | null;
}

export interface ReportWeight {
  date: string;
  week: number | null;
  value: number;
}

export interface PregnancyReport {
  range: DateRange;
  generatedAt: string | null;
  age: GestationalAgeV2 | null;
  dueDate: string | null;
  days: ReportDay[];
  weights: ReportWeight[];
}

export interface ReportRange {
  from: string;
  to: string;
}
