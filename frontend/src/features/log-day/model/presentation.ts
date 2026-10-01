import type { IconName, Tone } from '@/shared/ui';

/**
 * Icon + accent per category (nbl_/nbd_Log_Sheet_Cycle). Presentation only — the taxonomy itself comes
 * from the API; a category the client doesn't know yet (CB-MENO-01, condition programs…) still renders
 * with the neutral fallback.
 */
const LOOK: Record<string, { icon: IconName; tone: Tone }> = {
  bleeding: { icon: 'drop', tone: 'period' },
  pain: { icon: 'symptom', tone: 'bloom' },
  mood: { icon: 'smile', tone: 'brand' },
  symptoms: { icon: 'gut', tone: 'warm' },
  discharge: { icon: 'dropLine', tone: 'data' },
  sleep: { icon: 'bed', tone: 'brand' },
  appetite_energy: { icon: 'zap', tone: 'warm' },
  activity: { icon: 'run', tone: 'data' },
  urogenital: { icon: 'urine', tone: 'brand' },
  sex: { icon: 'heartLine', tone: 'bloom' },
  measurements: { icon: 'scaleSquare', tone: 'data' },
  meds: { icon: 'tablet', tone: 'bloom' },
  skin_hair: { icon: 'sparkle', tone: 'warm' },
  breasts: { icon: 'heart', tone: 'bloom' },
  pregnancy: { icon: 'heartLine', tone: 'bloom' },
  baby: { icon: 'sprout', tone: 'data' },
  note: { icon: 'note', tone: 'neutral' },
  custom: { icon: 'star', tone: 'brand' },
};

/** Param-level tiles that read better with their own glyph («وزن», «دمای پایه», «تست LH»). */
const PARAM_LOOK: Record<string, { icon: IconName; tone: Tone }> = {
  'measurements.weight': { icon: 'scaleSquare', tone: 'data' },
  'measurements.bbt': { icon: 'thermo', tone: 'brand' },
  'measurements.lh_test': { icon: 'flaskLh', tone: 'warm' },
  'measurements.pregnancy_test': { icon: 'flask', tone: 'bloom' },
  'measurements.bp_systolic': { icon: 'heartLine', tone: 'bloom' },
  'measurements.heart_rate': { icon: 'heartLine', tone: 'bloom' },
  'pregnancy.kicks': { icon: 'heartLine', tone: 'bloom' },
  'pregnancy.contractions': { icon: 'clock', tone: 'warm' },
  'baby.feeding': { icon: 'sprout', tone: 'data' },
};

const FALLBACK = { icon: 'note' as IconName, tone: 'neutral' as Tone };

export function categoryLook(code: string): { icon: IconName; tone: Tone } {
  return LOOK[code] ?? FALLBACK;
}

export function tileLook(key: string): { icon: IconName; tone: Tone } {
  return PARAM_LOOK[key] ?? categoryLook(key.split('.')[0]);
}

/**
 * Categories with a dedicated detail panel (Log_Bleeding, Log_Pain, Log_Measure); every other category
 * keeps its optional «جزئیات» params inline.
 */
export type PanelKind = 'bleeding' | 'pain' | 'measure';

export function panelOf(category: string): PanelKind | null {
  if (category === 'bleeding') return 'bleeding';
  if (category === 'pain') return 'pain';
  if (category === 'measurements') return 'measure';
  return null;
}

/**
 * Where a `link` param (a tile fed by another feature) opens. `null` = that feature is not built yet
 * (contraction timer B-N5-08, feeding/diapers B-N5-07, postpartum check-in) → «به‌زودی».
 */
export function linkHref(source: string | null): string | null {
  switch (source) {
    case 'kick_counter':
      return '/pregnancy/log?tab=movement';
    default:
      return null;
  }
}
