import type { CorrelationKey } from '@/entities/analysis';
import type { IconName, Tone } from '@/shared/ui';

/** Icon disc + accent of each pair (An_Correlations: sleep violet, energy amber, exercise turquoise). */
export const PAIR_STYLE: Record<CorrelationKey, { icon: IconName; tone: Tone }> = {
  sleep_mood: { icon: 'bed', tone: 'brand' },
  phase_energy: { icon: 'zap', tone: 'warm' },
  exercise_cramps: { icon: 'run', tone: 'data' },
  phase_mood: { icon: 'smile', tone: 'bloom' },
};

/** Strength pill tone: strong reads as the data accent, the weaker ones as amber (artboard). */
export function strengthTone(strength: string | null): Tone {
  return strength === 'strong' ? 'data' : 'warm';
}

export const PHASE_GROUPS = ['period', 'follicular', 'fertile', 'luteal'] as const;
export type PhaseGroup = (typeof PHASE_GROUPS)[number];

export function isPhaseGroup(key: string): key is PhaseGroup {
  return (PHASE_GROUPS as readonly string[]).includes(key);
}

export const PAIR_GROUPS = ['sleep_6_plus', 'sleep_under_6', 'no_exercise', 'exercise'] as const;
export type PairGroup = (typeof PAIR_GROUPS)[number];

export function isPairGroup(key: string): key is PairGroup {
  return (PAIR_GROUPS as readonly string[]).includes(key);
}

/** Decorative bars behind the Plus lock (no user data — a locked report sends none). */
export const TEASER = [
  { key: 't1', value: 40 },
  { key: 't2', value: 85 },
  { key: 't3', value: 60 },
  { key: 't4', value: 25 },
] as const;
