import type { LifeStage } from '@/entities/user';

/** The Today header chip's mode (IA_Nav «7 modes»): IVF is the TTC sub-mode with bloom's «IVF/IUI» switch. */
export type ChipMode = 'cycle' | 'ttc' | 'ivf' | 'pregnancy' | 'postpartum' | 'menopause' | 'teen';

/** Token tone of the chip dot — the same per-mode tones as the «من» hub mode pill (B-N1-10 MODE_TONE). */
export const CHIP_TONE: Record<ChipMode, 'period' | 'warm' | 'bloom' | 'data' | 'brand'> = {
  cycle: 'period',
  ttc: 'warm',
  ivf: 'warm',
  pregnancy: 'bloom',
  postpartum: 'bloom',
  menopause: 'data',
  teen: 'brand',
};

/** Which chip to draw for `life` (`null` = none: unknown yet, or a companion account, who has no mode to switch). */
export function chipMode(life: Pick<LifeStage, 'mode' | 'companion' | 'ivfIui'> | null | undefined): ChipMode | null {
  if (!life || life.companion) return null;
  if (life.mode === 'ttc' && life.ivfIui) return 'ivf';
  return life.mode;
}
