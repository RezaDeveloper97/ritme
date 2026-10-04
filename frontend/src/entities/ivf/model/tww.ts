import type { IvfCycle, IvfRoute } from './types';

/* The two-week wait (`GET /ivf/tww`) and the cycle outcome (CB-IVF-05 on the CB-IVF-01 API). */

/** The daily check-in chips (backend `ivf.Moods`), in board order. */
export const IVF_TWW_MOODS = ['calm', 'hopeful', 'worried', 'tired'] as const;
export type IvfTwwMood = (typeof IVF_TWW_MOODS)[number];

/** One luteal-support medicine with today's slots («۱ از ۲»). */
export interface IvfLutealMed {
  medId: number;
  name: string;
  dose: string | null;
  unit: string | null;
  route: IvfRoute;
  slots: { slot: string; taken: boolean }[];
  taken: number;
  total: number;
}

export interface IvfTww {
  /** The open cycle, null when there is none (then every other field is empty). */
  cycle: IvfCycle | null;
  transferOn: string | null;
  betaOn: string | null;
  /** today − transfer day; null before the transfer or when it is unknown. */
  daysSinceTransfer: number | null;
  /** beta day − today; 0 = today, negative once the day has passed; null when unknown. */
  daysToBeta: number | null;
  /** Tehran `Y-m-d` of today (the PUT /ivf/tww/{date} key). */
  today: string;
  todayMood: IvfTwwMood | null;
  moods: { date: string; mood: IvfTwwMood }[];
  lutealSupport: IvfLutealMed[];
}

export const IVF_OUTCOMES = ['positive', 'negative', 'cancelled'] as const;
export type IvfOutcomeResult = (typeof IVF_OUTCOMES)[number];

/** Follow-ups the API offers: positive → pregnancy setup; negative → the loss path or another cycle. */
export type IvfNextStep = 'pregnancy_setup' | 'loss' | 'new_cycle';

export interface IvfOutcome {
  result: IvfOutcomeResult;
  outcomeOn: string | null;
  nextSteps: IvfNextStep[];
}

/** Catalog `ivf_danger_signs` (OHSS, fever after a procedure) — admin-editable. */
export interface IvfDangerSign {
  code: string;
  title: string | null;
  body: string | null;
  hotlines: string[];
}
