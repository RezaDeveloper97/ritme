/*
 * Treatment & care (CB-MENO-10 on the CB-MENO-03 API): HRT, supplements and
 * lifestyle goals with the Saturday week's intakes, the next doctor review,
 * the day's side effects and the `meno_tips` placed on the screen.
 * Names and doses are what she typed — the app never prescribes.
 */

export const TREATMENT_KINDS = ['hrt', 'supplement', 'lifestyle'] as const;
export type TreatmentKind = (typeof TREATMENT_KINDS)[number];

/** Intake slots of hrt / supplement (care reminder times 08 · 13 · 18 · 22, weekly 09:00). */
export const TREATMENT_SCHEDULES = ['morning', 'noon', 'evening', 'night', 'weekly'] as const;
export type TreatmentSchedule = (typeof TREATMENT_SCHEDULES)[number];

export const TREATMENT_GOAL_UNITS = ['sessions', 'minutes'] as const;
export type TreatmentGoalUnit = (typeof TREATMENT_GOAL_UNITS)[number];

/** Side-effect codes in the board's order; labels are UI strings (`menopause.treatment.sideEffects.codes`). */
export const SIDE_EFFECT_CODES = ['breast_tenderness', 'spotting', 'headache', 'bloating', 'mood_change'] as const;
export type SideEffectCode = (typeof SIDE_EFFECT_CODES)[number];

/** API caps (backend-go internal/menopause/treatment_items.go). */
export const TREATMENT_LIMITS = {
  nameMax: 120,
  doseMax: 120,
  goalSessionsMax: 21,
  goalMinutesMax: 3000,
  intakeMinutesMax: 600,
  backfillDays: 30,
} as const;

export interface TreatmentDay {
  date: string;
  taken: boolean;
  /** Lifestyle minutes / sessions of the day. */
  amount: number | null;
}

export interface TreatmentGoal {
  target: number;
  unit: TreatmentGoalUnit;
  amount: number;
  done: boolean;
}

export interface TreatmentReminder {
  id: number;
  time: string;
  notify: boolean;
  active: boolean;
}

export interface TreatmentItem {
  id: number;
  kind: TreatmentKind;
  name: string;
  dose: string | null;
  schedule: TreatmentSchedule | null;
  form: string | null;
  startedOn: string | null;
  reviewOn: string | null;
  stoppedOn: string | null;
  active: boolean;
  takenToday: boolean;
  /** Seven days, Saturday → Friday. */
  week: TreatmentDay[];
  daysTaken: number;
  /** Scheduled days of the week so far. */
  days: number;
  adherencePct: number | null;
  goal: TreatmentGoal | null;
  reminder: TreatmentReminder | null;
}

export interface TreatmentTip {
  code: string;
  title: string | null;
  body: string | null;
  placement: string | null;
  weeklyGoal: number | null;
  goalUnit: TreatmentGoalUnit | null;
}

export interface TreatmentScreen {
  /** The day the screen was drawn for (today unless `?date=`). */
  date: string;
  week: { from: string; to: string };
  items: Record<TreatmentKind, TreatmentItem[]>;
  stopped: TreatmentItem[];
  /** Next review with her doctor; `suggested` = start + the catalog's months, not a date she set. */
  review: { on: string; itemId: number | null; suggested: boolean } | null;
  sideEffects: { codes: SideEffectCode[]; today: SideEffectCode[]; week: { date: string; codes: SideEffectCode[] }[] };
  tips: TreatmentTip[];
}

/** The add / edit form (PUT is a full replace: send every field). */
export interface TreatmentItemInput {
  kind: TreatmentKind;
  name: string;
  dose: string | null;
  schedule: TreatmentSchedule | null;
  form: string | null;
  startedOn: string | null;
  reviewOn: string | null;
  stoppedOn: string | null;
  weeklyGoal: number | null;
  goalUnit: TreatmentGoalUnit | null;
  remind: boolean;
}

/** The form's starting values for an existing item. */
export function treatmentItemInput(item: TreatmentItem): TreatmentItemInput {
  return {
    kind: item.kind,
    name: item.name,
    dose: item.dose,
    schedule: item.schedule,
    form: item.form,
    startedOn: item.startedOn,
    reviewOn: item.reviewOn,
    stoppedOn: item.stoppedOn,
    weeklyGoal: item.goal?.target ?? null,
    goalUnit: item.goal?.unit ?? null,
    remind: item.reminder?.notify ?? true,
  };
}

/** Snake-case body of POST / PUT /menopause/treatment/items; lifestyle drops the medicine fields and vice versa. */
export function toTreatmentItemBody(input: TreatmentItemInput, create: boolean): Record<string, unknown> {
  const body: Record<string, unknown> = {
    name: input.name.trim(),
    started_on: input.startedOn,
    review_on: input.reviewOn,
    stopped_on: input.stoppedOn,
  };
  if (create) body.kind = input.kind;
  if (input.kind === 'lifestyle') {
    body.weekly_goal = input.weeklyGoal;
    body.goal_unit = input.goalUnit;
  } else {
    body.dose = input.dose?.trim() ? input.dose.trim() : null;
    body.schedule = input.schedule;
    if (input.form) body.form = input.form;
    body.remind = input.remind;
  }
  return body;
}
