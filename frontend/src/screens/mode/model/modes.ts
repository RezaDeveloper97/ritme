import type { LifeMode } from '@/entities/user';
import type { IconName, Tone } from '@/shared/ui';

/** One card of the mode screen (`nbl_Me_Mode`), in artboard order; copy lives under `me.mode.cards.<key>`. */
export interface ModeCardDef {
  key: LifeMode;
  icon: IconName;
  /** Accent of the card: icon disc, selected border/tint, radio dot (artboard colours = these tokens). */
  tone: Tone;
}

export const MODE_CARDS: readonly ModeCardDef[] = [
  { key: 'cycle', icon: 'drop', tone: 'period' },
  { key: 'ttc', icon: 'target', tone: 'warm' },
  { key: 'pregnancy', icon: 'user', tone: 'bloom' },
  { key: 'postpartum', icon: 'heart', tone: 'data' },
  { key: 'menopause', icon: 'moon', tone: 'brand' },
  { key: 'teen', icon: 'sprout', tone: 'success' },
];

/**
 * What choosing `target` while in `current` takes. Pregnancy is owned by the
 * pregnancy domain (an active pregnancy profile keeps the effective mode on
 * pregnancy whatever is stored), so the switcher coordinates with it:
 *
 * - `enterPregnancy` — open the pregnancy setup (`/pregnancy/setup`, which skips
 *   itself when one is active); it activates the profile and stores the mode
 *   only on its final step, so leaving it changes nothing (stage B-3);
 * - `leavePregnancy` — `POST /pregnancy/deactivate` first, then store the mode;
 * - `store` — just `PUT /profile/life-stage {mode}`.
 */
export type SwitchPlan =
  | { kind: 'none' }
  | { kind: 'store'; target: LifeMode }
  | { kind: 'enterPregnancy' }
  | { kind: 'leavePregnancy'; target: LifeMode };

export function planSwitch(current: LifeMode, target: LifeMode): SwitchPlan {
  if (current === target) return { kind: 'none' };
  if (target === 'pregnancy') return { kind: 'enterPregnancy' };
  if (current === 'pregnancy') return { kind: 'leavePregnancy', target };
  return { kind: 'store', target };
}

/** Does this plan ask the user first? Leaving/entering pregnancy changes the whole app, so yes. */
export function needsConfirm(plan: SwitchPlan): boolean {
  return plan.kind === 'enterPregnancy' || plan.kind === 'leavePregnancy';
}
