import {
  IVF_STAGES,
  IVF_STIMULATION_ROLES,
  type IvfCycle,
  type IvfCyclePatch,
  type IvfCycleStartInput,
  type IvfMed,
  type IvfStage,
} from '@/entities/ivf';

/*
 * Pure helpers of the IVF cycle setup + stage/date editor (CB-IVF-06b). No
 * React, no locale: dates are `Y-m-d`, clocks `HH:MM` (Tehran wall clock).
 */

/** Stages the setup offers: a cycle starts at preparation or already in stimulation. */
export const SETUP_STAGES = ['prep', 'stim'] as const satisfies readonly IvfStage[];
export type SetupStage = (typeof SETUP_STAGES)[number];

/** Default clock of a new appointment date (scan / retrieval / transfer). */
export const DEFAULT_CLOCK = '09:00';

/** A date + clock pair of the editor (`*_at` fields). */
export interface WallClock {
  date: string | null;
  time: string;
}

export interface SetupDraft {
  protocol: string | null;
  stage: SetupStage;
  startedOn: string;
  stimStartedOn: string | null;
}

export interface CycleDraft {
  stage: IvfStage;
  protocol: string | null;
  startedOn: string;
  stimStartedOn: string | null;
  nextScan: WallClock;
  retrieval: WallClock;
  transfer: WallClock;
  betaOn: string | null;
}

export type DateField = 'startedOn' | 'stimStartedOn' | 'nextScan' | 'retrieval' | 'transfer' | 'betaOn';
export type DateProblem = 'future' | 'beforeStart' | 'transfer' | 'beta';

export function emptySetup(today: string): SetupDraft {
  return { protocol: null, stage: 'prep', startedOn: today, stimStartedOn: null };
}

/** Picking «تحریک» fills the stimulation start with the cycle start (she can move it). */
export function setSetupStage(draft: SetupDraft, stage: SetupStage): SetupDraft {
  if (stage === 'prep') return { ...draft, stage, stimStartedOn: null };
  return { ...draft, stage, stimStartedOn: draft.stimStartedOn ?? draft.startedOn };
}

export function setupProblems(draft: SetupDraft, today: string): Partial<Record<DateField, DateProblem>> {
  const out: Partial<Record<DateField, DateProblem>> = {};
  if (draft.startedOn > today) out.startedOn = 'future';
  if (draft.stage === 'stim' && draft.stimStartedOn && draft.stimStartedOn < draft.startedOn) {
    out.stimStartedOn = 'beforeStart';
  }
  return out;
}

export function setupToInput(draft: SetupDraft): IvfCycleStartInput {
  return {
    protocol: draft.protocol,
    stage: draft.stage,
    startedOn: draft.startedOn,
    stimStartedOn: draft.stage === 'stim' ? draft.stimStartedOn : null,
  };
}

/** `Y-m-d H:i[:s]` → `{date, time}` (time defaults to 09:00). */
export function splitWallClock(value: string | null): WallClock {
  if (!value) return { date: null, time: DEFAULT_CLOCK };
  const match = /^(\d{4}-\d{2}-\d{2})(?:[ T](\d{2}):(\d{2}))?/.exec(value);
  if (!match?.[1]) return { date: null, time: DEFAULT_CLOCK };
  return { date: match[1], time: match[2] && match[3] ? `${match[2]}:${match[3]}` : DEFAULT_CLOCK };
}

/** `{date, time}` → `Y-m-d H:i`, null without a date. */
export function joinWallClock(value: WallClock): string | null {
  return value.date ? `${value.date} ${value.time}` : null;
}

export function draftFromCycle(cycle: IvfCycle): CycleDraft {
  return {
    stage: cycle.stage,
    protocol: cycle.protocol,
    startedOn: cycle.startedOn,
    stimStartedOn: cycle.stimStartedOn,
    nextScan: splitWallClock(cycle.nextScanAt),
    retrieval: splitWallClock(cycle.retrievalAt),
    transfer: splitWallClock(cycle.transferAt),
    betaOn: cycle.betaOn,
  };
}

/**
 * Moving to a stage whose start is not known yet fills it with `today`
 * (stimulation start), so the «روز ۱ تحریک» chip appears at once.
 */
export function setDraftStage(draft: CycleDraft, stage: IvfStage, today: string): CycleDraft {
  if (stage === 'stim' && !draft.stimStartedOn) return { ...draft, stage, stimStartedOn: today };
  return { ...draft, stage };
}

/** The same order checks as the API (422 on the field), so most mistakes never leave the form. */
export function draftProblems(draft: CycleDraft, today: string): Partial<Record<DateField, DateProblem>> {
  const out: Partial<Record<DateField, DateProblem>> = {};
  const start = draft.startedOn;
  if (start > today) out.startedOn = 'future';
  const notBeforeStart: [DateField, string | null][] = [
    ['stimStartedOn', draft.stimStartedOn],
    ['nextScan', draft.nextScan.date],
    ['retrieval', draft.retrieval.date],
    ['transfer', draft.transfer.date],
    ['betaOn', draft.betaOn],
  ];
  for (const [field, date] of notBeforeStart) {
    if (date && date < start) out[field] = 'beforeStart';
  }
  const retrieval = joinWallClock(draft.retrieval);
  const transfer = joinWallClock(draft.transfer);
  if (!out.transfer && retrieval && transfer && transfer < retrieval) out.transfer = 'transfer';
  if (!out.betaOn && draft.betaOn && draft.transfer.date && draft.betaOn < draft.transfer.date) out.betaOn = 'beta';
  return out;
}

/** Only what changed against the stored cycle (the API takes a partial body). */
export function draftToPatch(draft: CycleDraft, cycle: IvfCycle): IvfCyclePatch {
  const base = draftFromCycle(cycle);
  const patch: IvfCyclePatch = {};
  if (draft.stage !== base.stage) patch.stage = draft.stage;
  if (draft.protocol !== base.protocol) patch.protocol = draft.protocol;
  if (draft.startedOn !== base.startedOn) patch.startedOn = draft.startedOn;
  if (draft.stimStartedOn !== base.stimStartedOn) patch.stimStartedOn = draft.stimStartedOn;
  const clocks = [
    ['nextScan', 'nextScanAt'],
    ['retrieval', 'retrievalAt'],
    ['transfer', 'transferAt'],
  ] as const;
  for (const [field, key] of clocks) {
    const next = joinWallClock(draft[field]);
    if (next !== joinWallClock(base[field])) patch[key] = next;
  }
  if (draft.betaOn !== base.betaOn) patch.betaOn = draft.betaOn;
  return patch;
}

const stageIndex = (stage: IvfStage) => IVF_STAGES.indexOf(stage);

/**
 * Medicines to offer stopping after a stage change: moving from preparation /
 * stimulation to retrieval or later leaves the active stimulation and
 * suppression medicines (never the trigger, never luteal support) [needs clinical review].
 */
export function medsToStop(meds: readonly IvfMed[], from: IvfStage, to: IvfStage): IvfMed[] {
  const pastStim = stageIndex('retrieval');
  if (stageIndex(from) >= pastStim || stageIndex(to) < pastStim) return [];
  return meds.filter(
    (m) => m.isActive && !m.isTrigger && m.reminderId !== null && IVF_STIMULATION_ROLES.includes(m.role),
  );
}
