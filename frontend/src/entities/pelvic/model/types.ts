/*
 * Pelvic floor program (CB-PELV-01 API, screens CB-PELV-02): an 8-week Kegel plan with a level per week, one guided
 * session row per trained day and a daily bladder diary. Health data (§11): never log any of it.
 */

/** `leak` answers of the bladder diary, in board order. */
export const PELVIC_LEAKS = ['none', 'cough', 'urgency', 'unexplained'] as const;
export type PelvicLeak = (typeof PELVIC_LEAKS)[number];

/** UTI symptoms, in board order. */
export const PELVIC_UTI_SYMPTOMS = ['burning', 'frequency', 'cloudy_odor'] as const;
export type PelvicUtiSymptom = (typeof PELVIC_UTI_SYMPTOMS)[number];

/** Upper bound of `night_voids` (the API's max). */
export const PELVIC_MAX_NIGHT_VOIDS = 20;

/** A `pelvic_levels` row: one session = sets × reps × (hold + rest). */
export interface PelvicLevel {
  code: string;
  number: number;
  /** Admin copy in the request locale (null → bundled). */
  title: string | null;
  /** The exercise instruction (null → bundled). */
  body: string | null;
  holdSec: number;
  restSec: number;
  reps: number;
  sets: number;
  sessionSec: number;
}

export interface PelvicWeekDay {
  date: string;
  done: boolean;
}

export interface PelvicProgram {
  startedOn: string;
  week: number;
  weeksTotal: number;
  completed: boolean;
  level: PelvicLevel | null;
  streakDays: number;
  todayDone: boolean;
  /** Today's Saturday-start week. */
  weekDays: PelvicWeekDay[];
}

export interface PelvicDiary {
  /** `YYYY-MM-DD` (today in Tehran on the overview). */
  date: string;
  leak: PelvicLeak | null;
  nightVoids: number | null;
  utiSymptoms: PelvicUtiSymptom[];
  utiAlert: boolean;
}

export interface PelvicOverview {
  /** Null until she starts the program. */
  program: PelvicProgram | null;
  diaryToday: PelvicDiary;
}

/** A diary write: only the keys sent change; `null` clears one. */
export interface PelvicDiaryPatch {
  leak?: PelvicLeak | null;
  night_voids?: number | null;
  uti_symptoms?: PelvicUtiSymptom[] | null;
}

export interface PelvicSessionInput {
  setsCompleted: number;
  durationSec: number;
}

/** `pelvic_alerts` catalog copy (null → bundled string). */
export interface PelvicAlerts {
  utiWarning: string | null;
  suitability: string | null;
}
