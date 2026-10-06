/*
 * Domain shapes of `/api/v1/vitals*` (B-N6-01, D-63). The server classifies every
 * reading (ACC/AHA 2017 blood pressure, ADA 2025 glucose, AHA resting heart rate)
 * and stores glucose in mg/dL; a reading keeps the unit it was typed in.
 */

export const VITAL_TYPES = ['bp', 'hr', 'glucose'] as const;
export type VitalType = (typeof VITAL_TYPES)[number];

export const ARMS = ['left', 'right'] as const;
export type Arm = (typeof ARMS)[number];
export const POSITIONS = ['sitting', 'standing', 'lying'] as const;
export type Position = (typeof POSITIONS)[number];

export const GLUCOSE_UNITS = ['mg_dl', 'mmol_l'] as const;
export type GlucoseUnit = (typeof GLUCOSE_UNITS)[number];
export const GLUCOSE_CONTEXTS = ['fasting', 'before_meal', 'after_meal', 'bedtime', 'random'] as const;
export type GlucoseContext = (typeof GLUCOSE_CONTEXTS)[number];
export const GLUCOSE_METHODS = ['glucometer', 'lab', 'sensor'] as const;
export type GlucoseMethod = (typeof GLUCOSE_METHODS)[number];

export const HR_CONTEXTS = ['resting', 'after_exercise', 'after_waking', 'stress'] as const;
export type HrContext = (typeof HR_CONTEXTS)[number];

/** Report ranges of `GET /vitals/reports/{type}?range=`. */
export const REPORT_RANGES = ['7d', '14d', '30d', '90d'] as const;
export type ReportRange = (typeof REPORT_RANGES)[number];
export const DEFAULT_RANGE: Record<VitalType, ReportRange> = { bp: '7d', hr: '7d', glucose: '14d' };
/** Glucose report filter: every context or one. */
export type GlucoseFilter = 'all' | GlucoseContext;

/** Input limits — the server's validation (internal/vitals/model.go). */
export const LIMITS = {
  systolic: { min: 50, max: 300 },
  diastolic: { min: 30, max: 200 },
  pulse: { min: 30, max: 250 },
  mgDl: { min: 20, max: 600 },
  mmolL: { min: 1.1, max: 33.3 },
  noteMax: 500,
  backfillDays: 730,
} as const;

/** Colour of a class on the clients (server `tone`): ok · watch · high · urgent. */
export type VitalTone = 'ok' | 'watch' | 'high' | 'urgent';

export interface VitalClass {
  code: string;
  tone: VitalTone;
}

export interface BloodPressureValue {
  systolic: number;
  diastolic: number;
  pulse: number | null;
  arm: Arm | null;
  position: Position | null;
}

export interface GlucoseValue {
  mgDl: number;
  mmolL: number;
  /** The value in the unit it was typed in (mg/dL whole, mmol/L 1 decimal). */
  value: number;
  unit: GlucoseUnit;
  context: GlucoseContext | null;
  method: GlucoseMethod | null;
}

export interface HeartRateValue {
  bpm: number;
  context: HrContext | null;
}

export type ReadingPeriod = 'morning' | 'afternoon' | 'night';

/** One reading: a timed vitals row, or a log-sheet day value (`source: log`, read-only, no time). */
export interface VitalReading {
  id: number | null;
  source: 'vitals' | 'log';
  type: VitalType;
  /** `Y-m-d` (Tehran). */
  date: string;
  /** `HH:mm` or null for a log-sheet value. */
  time: string | null;
  measuredAt: string | null;
  period: ReadingPeriod | null;
  bloodPressure: BloodPressureValue | null;
  glucose: GlucoseValue | null;
  heartRate: HeartRateValue | null;
  classification: VitalClass | null;
  urgent: boolean;
  note: string | null;
  editable: boolean;
}

export type PlanDayState = 'done' | 'missed' | 'due';

export interface PlanWeekItem {
  type: VitalType;
  slot: string;
  planned: number;
  done: number;
  days: ReadonlyArray<{ date: string; state: PlanDayState }>;
}

export interface PlanWeek {
  from: string;
  to: string;
  planned: number;
  done: number;
  items: readonly PlanWeekItem[];
}

export interface VitalsHub {
  date: string;
  latest: Record<VitalType, VitalReading | null>;
  plan: PlanWeek;
  recent: readonly VitalReading[];
  remindersEnabled: boolean;
}

export interface PlanItem {
  type: VitalType;
  slot: string;
  /** Weekdays, 0 = Saturday … 6 = Friday. */
  days: readonly number[];
  /** `HH:mm` or null. */
  remindAt: string | null;
}

export interface VitalsPlan {
  items: readonly PlanItem[];
  week: PlanWeek;
  slots: Record<VitalType, readonly string[]>;
  remindersEnabled: boolean;
}

/** The urgent modal of a saved reading over a safety threshold (copy is admin-editable). */
export interface VitalAlert {
  rule: string;
  title: string | null;
  whatWeSaw: string | null;
  advice: string | null;
  contact: string | null;
  actions: ReadonlyArray<{ key: string; label: string; phone: string | null }>;
}

export interface SavedReading {
  reading: VitalReading;
  alert: VitalAlert | null;
}

export interface DistributionEntry {
  code: string;
  tone: VitalTone;
  count: number;
  percent: number;
}

export interface BpAverage {
  systolic: number;
  diastolic: number;
  readings: number;
  classification: VitalClass | null;
}
export interface GlucoseAverage {
  mgDl: number;
  mmolL: number;
  readings: number;
}
export interface HrAverage {
  bpm: number;
  readings: number;
}

interface ReportBase<A> {
  range: { key: ReportRange; from: string; to: string; days: number };
  readings: number;
  average: A | null;
  min: VitalReading | null;
  max: VitalReading | null;
  distribution: readonly DistributionEntry[];
  morningVsNight: { morning: A | null; night: A | null; nightOutOfRange: number };
  timeInRange: { inRange: number; below: number; above: number; readings: number; percent: number };
}

export interface BpReport extends ReportBase<BpAverage> {
  type: 'bp';
  pulse: HrAverage | null;
  target: { systolicMax: number; diastolicMax: number };
  series: ReadonlyArray<{ date: string; readings: number; systolic: number; diastolic: number }>;
}

export interface GlucoseReport extends ReportBase<GlucoseAverage> {
  type: 'glucose';
  filter: GlucoseFilter;
  byContext: ReadonlyArray<{
    context: GlucoseContext;
    readings: number;
    average: GlucoseAverage | null;
    aboveTarget: number;
    target: { min: number; max: number };
  }>;
  series: ReadonlyArray<{
    date: string;
    readings: number;
    fasting: number | null;
    afterMeal: number | null;
    other: number | null;
  }>;
}

export interface HrReport extends ReportBase<HrAverage> {
  type: 'hr';
  restingAverage: HrAverage | null;
  byContext: ReadonlyArray<{ context: HrContext; readings: number; average: HrAverage | null }>;
  target: { min: number; max: number };
  series: ReadonlyArray<{ date: string; readings: number; bpm: number }>;
}

export type VitalReport = BpReport | GlucoseReport | HrReport;

/** `GET /vitals/thresholds`, the parts the clients draw with. */
export interface VitalThresholds {
  version: string;
  bp: {
    elevatedSystolic: number;
    stage1: { systolic: number; diastolic: number };
    stage2: { systolic: number; diastolic: number };
    urgentAbove: { systolic: number; diastolic: number };
  };
  glucose: {
    lowBelow: number;
    urgentBelow: number;
    mmolFactor: number;
    contexts: Record<GlucoseContext, { targetMin: number; targetMax: number; veryHighFrom: number }>;
  };
  hr: { min: number; max: number; classifiedContexts: readonly string[] };
}

/** New-reading inputs (the add forms). */
export type ReadingInput =
  | {
      type: 'bp';
      systolic: number;
      diastolic: number;
      pulse: number | null;
      arm: Arm | null;
      position: Position | null;
      measuredAt: string | null;
      note: string | null;
    }
  | {
      type: 'glucose';
      value: number;
      unit: GlucoseUnit;
      context: GlucoseContext;
      method: GlucoseMethod | null;
      measuredAt: string | null;
      note: string | null;
    }
  | { type: 'hr'; bpm: number; context: HrContext; measuredAt: string | null; note: string | null };
