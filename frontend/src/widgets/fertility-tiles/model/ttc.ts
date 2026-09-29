import type { CycleFertilityLevel, MainPhase } from '@/entities/cycle';

/** The four phases of the `v19_Main` legend. */
export const TTC_PHASES = ['period', 'follicular', 'fertile', 'luteal'] as const;
export type TtcPhase = (typeof TTC_PHASES)[number];

/**
 * The legend pill the engine's main phase lights up (task.md §13): an expected
 * but unconfirmed period still reads as the late luteal phase; `unknown` lights
 * nothing rather than guessing.
 */
export function ttcPhase(mainPhase: MainPhase | null): TtcPhase | null {
  switch (mainPhase) {
    case 'menstrual':
      return 'period';
    case 'follicular':
      return 'follicular';
    case 'fertile':
      return 'fertile';
    case 'luteal':
    case 'period_expected':
      return 'luteal';
    default:
      return null;
  }
}

/**
 * How much of the chance donut is filled for a v1.1 fertility level (§26).
 * `high` is the artboard's 80 %; `unknown` / no level draws an empty ring.
 */
export function chanceFraction(level: CycleFertilityLevel | null): number {
  switch (level) {
    case 'peak':
      return 1;
    case 'high':
      return 0.8;
    case 'medium':
      return 0.55;
    case 'low':
      return 0.3;
    case 'none':
      return 0.1;
    default:
      return 0;
  }
}
