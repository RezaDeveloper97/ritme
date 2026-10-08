import type { IconName, Tone } from '@/shared/ui';

import type { ConditionProgram } from './types';

/** The programs hub (`/programs`, CB-COND-02). */
export const PROGRAMS_HUB_PATH = '/programs';

/** Icon disc of each card (nbl_Cond_Hub: pain = period, mood = brand, flow = period, PCOS = data). */
export const CONDITION_LOOK: Record<ConditionProgram, { icon: IconName; tone: Tone }> = {
  endo: { icon: 'flame', tone: 'period' },
  pmdd: { icon: 'brain', tone: 'brand' },
  heavy_bleeding: { icon: 'drop', tone: 'period' },
  pcos: { icon: 'sprout', tone: 'data' },
};

/**
 * Where a program's own screen lives: the diaries of CB-COND-03/04/05 (the same paths the nudges link to) and, for
 * PCOS — which has no diary of its own and reads the existing logs — the cycle analysis.
 */
export const CONDITION_PATH: Record<ConditionProgram, string> = {
  endo: '/programs/pain',
  pmdd: '/programs/pmdd',
  heavy_bleeding: '/programs/bleeding',
  pcos: '/analysis/cycle',
};

/** bloom's «برنامه‌های مراقبتی» services tiles (catalog `services_programs`) that are condition programs. */
const SERVICE_TILE_PROGRAM: Readonly<Record<string, ConditionProgram>> = {
  pain_endometriosis: 'endo',
  pmdd: 'pmdd',
  heavy_bleeding: 'heavy_bleeding',
};

/** Is this services-hub program tile backed by the programs hub? */
export function isConditionServiceTile(code: string): boolean {
  return Object.hasOwn(SERVICE_TILE_PROGRAM, code);
}

const IN_APP_PATH = /^\/[a-z0-9][a-z0-9/_-]{0,127}$/;

/** An API-supplied link, only when it is an in-app path (never a scheme, host or `//`). */
export function safeInAppPath(value: unknown): string | null {
  return typeof value === 'string' && IN_APP_PATH.test(value) ? value : null;
}
