import type { CompanionPhase, CompanionSignal, CompanionSignals } from '@/entities/companion';

/*
 * «وضعیت کلی امروز» (companion v2, compHome): one probabilistic sentence from her coarse phase (F-01: phase
 * only — no day numbers, no countdown) plus at most one gentle addition from a shared «today» signal. The copy
 * lives under `companionHome.status.*`; it never ties mood to the period, and a `null` signal (not shared)
 * is simply absent.
 */

/** Which shared signal speaks first when several are on today. Mood last: the least physical, the most personal. */
export const SIGNAL_PRIORITY: readonly CompanionSignal[] = ['pain', 'energy', 'sleep', 'mood'];

export interface StatusCopy {
  /** `status.<phase>.main` / `.sub`. */
  phase: CompanionPhase;
  /** `status.signals.<signal>`, or null for no addition. */
  signal: CompanionSignal | null;
}

export function statusCopy(phase: CompanionPhase, signals: CompanionSignals): StatusCopy {
  const signal = SIGNAL_PRIORITY.find((s) => signals[s] === true) ?? null;
  return { phase, signal };
}
