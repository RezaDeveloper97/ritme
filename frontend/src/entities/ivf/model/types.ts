/**
 * IVF treatment (CB-IVF-01 API, `/api/v1/ivf*`, Go only). A sub-mode of TTC:
 * bloom's «IVF/IUI» switch (`ivf_iui`) turns it on, an open cycle drives the
 * home (nbl_IVF_Home). Health data — never log it (CLAUDE.md §11).
 */

export const IVF_STAGES = ['prep', 'stim', 'retrieval', 'transfer', 'tww', 'test'] as const;
export type IvfStage = (typeof IVF_STAGES)[number];

export type IvfStepStatus = 'done' | 'current' | 'todo';

export const IVF_ROUTES = ['subcutaneous', 'intramuscular', 'oral', 'vaginal', 'other'] as const;
export type IvfRoute = (typeof IVF_ROUTES)[number];

export const IVF_ROLES = ['stimulation', 'suppression', 'trigger', 'luteal_support', 'other'] as const;
export type IvfRole = (typeof IVF_ROLES)[number];

export interface IvfTimelineStep {
  stage: IvfStage;
  status: IvfStepStatus;
  /** The stage's start (`Y-m-d`) when known. */
  date: string | null;
}

export interface IvfCycle {
  id: number;
  /** «سیکل اول» = 1. */
  number: number;
  protocol: string | null;
  stage: IvfStage;
  status: 'open' | 'closed';
  startedOn: string;
  betaOn: string | null;
  notifyCompanion: boolean;
  /** Day within the current stage («روز ۷ تحریک»), null when unknown. */
  stageDay: number | null;
  daysToBeta: number | null;
  timeline: IvfTimelineStep[];
}

/** One scheduled dose of a cycle medicine on a day. */
export interface IvfDose {
  medId: number;
  name: string;
  dose: string | null;
  /** Free text; usually a unit code (`iu`, `mg`…). */
  unit: string | null;
  route: IvfRoute;
  role: IvfRole;
  isTrigger: boolean;
  date: string;
  /** `HH:MM`. */
  slot: string;
  taken: boolean;
  site: string | null;
}

export interface IvfDoseDay {
  date: string;
  doses: IvfDose[];
}

export type IvfAppointmentKind = 'scan' | 'retrieval' | 'transfer' | 'beta';

export interface IvfNextAppointment {
  kind: IvfAppointmentKind;
  /** The care appointment (open it at `/reminders/appointment/{id}`). */
  id: number;
  title: string;
  /** Tehran wall clock `Y-m-d H:i:s`. */
  scheduledAt: string;
  daysUntil: number | null;
  /** First prep note («ناشتا نیاز نیست»), if any. */
  prep: string | null;
}

export interface IvfCompanion {
  /** An active «همدم» link exists — the reminder toggle is hidden without it. */
  linked: boolean;
  notify: boolean;
}

/** `GET /ivf` — the IVF «امروز». */
export interface IvfHome {
  enabled: boolean;
  cycle: IvfCycle | null;
  cyclesCount: number;
  today: IvfDoseDay;
  nextAppointment: IvfNextAppointment | null;
  companion: IvfCompanion;
}

/** A catalog `ivf_stages` row: admin-editable title + timeline hint (request locale). */
export interface IvfStageInfo {
  code: string;
  title: string | null;
  body: string | null;
}

/** `POST /ivf/meds/{id}/doses` (and the DELETE undo). */
export interface IvfDoseInput {
  medId: number;
  date: string;
  slot: string;
  site?: string | null;
}
