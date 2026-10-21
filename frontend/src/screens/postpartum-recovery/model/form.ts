import type {
  BreastSymptom,
  LochiaAmount,
  LochiaColor,
  PainLevel,
  PainLocation,
  PostpartumRecovery,
  RecoveryUpdate,
} from '@/entities/postpartum';

/** The recovery form (nbl_v15_Recovery). `null` = not answered today. */
export interface RecoveryForm {
  lochiaAmount: LochiaAmount | null;
  lochiaColor: LochiaColor | null;
  painLevel: PainLevel | null;
  painLocations: PainLocation[];
  breasts: BreastSymptom[] | null;
  feedsCount: number | null;
  sleepHours: number | null;
  /** Red flags (T-M9-06); `null` = not answered. */
  temperatureC: number | null;
  padSoakedWithinHour: boolean | null;
  severePain: boolean | null;
  breathless: boolean | null;
}

/** The three yes/no red flags, in display order. */
export const RED_FLAGS = ['padSoakedWithinHour', 'severePain', 'breathless'] as const;
export type RedFlag = (typeof RED_FLAGS)[number];

/** Where the temperature stepper starts when she first enters one. */
export const TEMPERATURE_START = 37;

/** One decimal, inside the API bounds (the stepper adds 0.1 in floating point). */
export function roundTemperature(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, Math.round(value * 10) / 10));
}

export function formFrom(r: PostpartumRecovery | undefined): RecoveryForm {
  return {
    lochiaAmount: r?.lochiaAmount ?? null,
    lochiaColor: r?.lochiaColor ?? null,
    painLevel: r?.painLevel ?? null,
    painLocations: r?.painLocations ?? [],
    breasts: r?.breasts ?? null,
    feedsCount: r?.feedsCount ?? null,
    sleepHours: r?.sleepHours ?? null,
    temperatureC: r?.temperatureC ?? null,
    padSoakedWithinHour: r?.padSoakedWithinHour ?? null,
    severePain: r?.severePain ?? null,
    breathless: r?.breathless ?? null,
  };
}

/** Pain locations only matter for a real pain level. */
export function needsLocation(level: PainLevel | null): boolean {
  return level === 'mild' || level === 'moderate' || level === 'severe';
}

export function locationMissing(form: RecoveryForm): boolean {
  return needsLocation(form.painLevel) && form.painLocations.length === 0;
}

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

/**
 * The partial PUT body: only what changed. Pain level and locations travel
 * together (the API validates them as one value); «no pain» clears the places.
 */
export function diffForm(saved: RecoveryForm, form: RecoveryForm): RecoveryUpdate {
  const out: RecoveryUpdate = {};
  if (!same(saved.lochiaAmount, form.lochiaAmount)) out.lochiaAmount = form.lochiaAmount;
  if (!same(saved.lochiaColor, form.lochiaColor)) out.lochiaColor = form.lochiaColor;
  if (!same(saved.painLevel, form.painLevel) || !same(saved.painLocations, form.painLocations)) {
    out.painLevel = form.painLevel;
    out.painLocations = needsLocation(form.painLevel) ? form.painLocations : [];
  }
  if (!same(saved.breasts, form.breasts)) out.breasts = form.breasts;
  if (!same(saved.feedsCount, form.feedsCount)) out.feedsCount = form.feedsCount;
  if (!same(saved.sleepHours, form.sleepHours)) out.sleepHours = form.sleepHours;
  if (!same(saved.temperatureC, form.temperatureC)) out.temperatureC = form.temperatureC;
  for (const flag of RED_FLAGS) {
    if (!same(saved[flag], form[flag])) out[flag] = form[flag];
  }
  return out;
}

export function toggleIn<T>(list: readonly T[], value: T): T[] {
  return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
}
