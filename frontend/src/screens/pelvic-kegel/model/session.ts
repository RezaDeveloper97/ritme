/** The level numbers a guided session needs (a `PelvicLevel` fits). */
export interface KegelPlan {
  holdSec: number;
  restSec: number;
  reps: number;
  sets: number;
}

/** One step of the session: squeeze for `holdSec`, then relax for `restSec`; reps × sets of them. */
export interface KegelPhase {
  kind: 'hold' | 'relax';
  /** 1-based set and rep. */
  set: number;
  rep: number;
  /** Length of this step. */
  ms: number;
}

/** How many steps (holds + relaxes) a session has. */
export function phaseCount(plan: KegelPlan): number {
  return plan.sets * plan.reps * 2;
}

/** Step `index` (0-based): holds on even, relaxes on odd indexes. */
export function phaseAt(index: number, plan: KegelPlan): KegelPhase {
  const repIndex = Math.floor(index / 2);
  const kind = index % 2 === 0 ? 'hold' : 'relax';
  return {
    kind,
    set: Math.floor(repIndex / plan.reps) + 1,
    rep: (repIndex % plan.reps) + 1,
    ms: (kind === 'hold' ? plan.holdSec : plan.restSec) * 1000,
  };
}

/** Reps whose squeeze is over once step `index` is reached (a rep counts when its hold ends). */
export function repsDone(index: number): number {
  return Math.ceil(index / 2);
}

/** Whole sets finished by step `index`. */
export function setsDone(index: number, plan: KegelPlan): number {
  return Math.min(plan.sets, Math.floor(repsDone(index) / plan.reps));
}

/** Filled segments of the current set's rep bar at step `index`. */
export function repBarFilled(index: number, plan: KegelPlan): number {
  if (index >= phaseCount(plan)) return plan.reps;
  return repsDone(index) - (phaseAt(index, plan).set - 1) * plan.reps;
}

/** Session length to save: wall-clock seconds, clamped to the API's 1–3600. */
export function durationSeconds(startedAt: number, now: number): number {
  return Math.min(3600, Math.max(1, Math.round((now - startedAt) / 1000)));
}
