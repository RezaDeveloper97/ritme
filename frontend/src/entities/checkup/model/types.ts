/**
 * Periodic checkups (M4): the screening plan, records and the self-exam guide.
 * Contract: docs/checkups/README.md (`/api/v1/checkups/*`, Go only).
 *
 * Personal health data (§11) — display only, never logged, never put in
 * analytics, error reports or URLs beyond what the API needs. Report files
 * (photo / PDF) never reach the server at all: they live on the device
 * (`model/attachments.ts`) and the API only stores `has_attachment`.
 *
 * Dates cross the boundary as `Y-m-d` and are converted to the locale calendar
 * only for display (`shared/lib/date`, §7). The server already localizes the
 * human labels (`interval_label`, `timing_label`, `next_due_label`).
 */

export const CHECKUP_CATEGORIES = [
  'monthly',
  'six_monthly',
  'annual',
  'multi_year',
  'age_based',
  'custom',
] as const;
export type CheckupCategory = (typeof CHECKUP_CATEGORIES)[number];

/** Worst → best; `not_yet` = below `age_min`. */
export const CHECKUP_STATUSES = ['overdue', 'due', 'soon', 'up_to_date', 'not_yet', 'disabled'] as const;
export type CheckupStatus = (typeof CHECKUP_STATUSES)[number];

/** List sections in the order the artboard shows them (the server sends its own order). */
export const CHECKUP_SECTIONS = ['this_month', 'overdue', ...CHECKUP_CATEGORIES] as const;
export type CheckupSection = (typeof CHECKUP_SECTIONS)[number];

export const CHECKUP_TONES = ['rose', 'violet', 'amber', 'teal', 'green', 'neutral'] as const;
export type CheckupTone = (typeof CHECKUP_TONES)[number];

export const CHECKUP_PERFORMERS = ['self', 'doctor', 'lab', 'dentist'] as const;
export type CheckupPerformer = (typeof CHECKUP_PERFORMERS)[number];

export const CHECKUP_RESULTS = ['normal', 'follow_up', 'pending'] as const;
export type CheckupResult = (typeof CHECKUP_RESULTS)[number];

/** `?filter=` of GET /checkups (tabs همه / نیاز به اقدام / انجام‌شده). */
export const CHECKUP_LIST_FILTERS = ['all', 'action', 'done'] as const;
export type CheckupListFilter = (typeof CHECKUP_LIST_FILTERS)[number];

/** `?filter=` of GET /checkups/records (tabs همه / امسال / با پیوست). */
export const CHECKUP_RECORD_FILTERS = ['all', 'this_year', 'with_attachment'] as const;
export type CheckupRecordFilter = (typeof CHECKUP_RECORD_FILTERS)[number];

/** Interval chips of the custom-checkup form (۱ ماه / ۶ ماه / ۱ سال / ۲ سال / ۳ سال). */
export const CUSTOM_INTERVAL_MONTHS = [1, 6, 12, 24, 36] as const;

export interface CheckupSummary {
  total: number;
  /** Includes `soon`. */
  upToDate: number;
  due: number;
  overdue: number;
}

/** One row of the plan (GET /checkups items, /checkups/home highlights). */
export interface CheckupItem {
  id: number;
  /** Catalog key (`breast_self_exam`, `pap_smear`…); null for a custom checkup. */
  key: string | null;
  title: string;
  subtitle: string | null;
  category: CheckupCategory;
  section: CheckupSection;
  status: CheckupStatus;
  /** Icon name from `shared/ui/Icon` — resolve with `checkupIcon()`. */
  icon: string | null;
  tone: CheckupTone;
  /** «هر ۳ سال». */
  intervalLabel: string | null;
  /** «روز ۷ تا ۱۰ سیکل» (cycle-timed types only). */
  timingLabel: string | null;
  lastDoneOn: string | null;
  nextDueOn: string | null;
  /** «۳ روز دیگر» / «عقب‌افتاده از فروردین» / «از ۱۴۰۹ (۴۰ سالگی)». */
  nextDueLabel: string | null;
  isCustom: boolean;
}

/** GET /checkups. */
export interface CheckupList {
  /** Age the plan was computed for; null when the profile has no birthday. */
  age: number | null;
  summary: CheckupSummary;
  items: CheckupItem[];
}

/** GET /checkups/home — null when nothing applies (the card hides). */
export interface CheckupHome {
  summary: CheckupSummary;
  /** At most 2: due/overdue, cycle-timed first. */
  highlights: CheckupItem[];
}

export interface CheckupGuideStep {
  title: string;
  body: string | null;
}

export interface CheckupFindingOption {
  key: string;
  label: string;
  /** «چیزی متفاوت نبود» — selecting it clears the others. */
  exclusive: boolean;
}

export interface CheckupSettings {
  enabled: boolean;
  remind: boolean;
}

export interface CheckupRecord {
  id: number;
  checkupTypeId: number;
  /** Present on the history list (records of several types). */
  checkupTitle: string | null;
  /** History list only: the type's catalog key, icon and tone (null / `neutral` elsewhere). */
  checkupKey: string | null;
  checkupIcon: string | null;
  checkupTone: CheckupTone;
  doneOn: string;
  result: CheckupResult;
  /** Self-exam finding keys. */
  findings: string[];
  note: string | null;
  hasAttachment: boolean;
  /** The user's override of the computed next due date. */
  nextDueOn: string | null;
}

/** GET /checkups/{id}. */
export interface CheckupDetail extends CheckupItem {
  why: string | null;
  performedBy: CheckupPerformer;
  intervalMonths: number | null;
  cycleDayFrom: number | null;
  cycleDayTo: number | null;
  prepSteps: string[];
  guideSteps: CheckupGuideStep[];
  findingOptions: CheckupFindingOption[];
  /** Latest 2. */
  records: CheckupRecord[];
  settings: CheckupSettings;
}

/** One page of GET /checkups/records. */
export interface CheckupRecordPage {
  records: CheckupRecord[];
  page: number;
  lastPage: number;
  total: number;
}

/** GET /checkups/preview-next — the MarkDone banner. */
export interface CheckupNextPreview {
  nextDueOn: string | null;
  /** «شهریور ۱۴۰۸». */
  nextDueLabel: string | null;
  /** «۳ سال بعد». */
  intervalLabel: string | null;
  /** «یادآوری ۱ ماه قبل». */
  reminderLabel: string | null;
}

/**
 * POST/PUT record result. The README promises the recomputed item; the created
 * record (needed to key the on-device attachment) is read when the server
 * sends it as `record` — see the note in `api/schema.ts`.
 */
export interface CheckupRecordResult {
  item: CheckupItem | null;
  record: CheckupRecord | null;
}
