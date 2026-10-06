import type { GlucoseUnit } from './types';

/*
 * Glucose units (pure, no locale): the server stores mg/dL and converts with
 * the ADA factor 18 (`thresholds.go` MmolFactor), rounding mg/dL to a whole
 * number and mmol/L to one decimal. The add form edits in either unit; reports
 * arrive in both, and the screen shows the unit the user picked.
 */

export const MMOL_FACTOR = 18;

const round1 = (n: number) => Math.round(n * 10) / 10;

/** mg/dL of a typed value (1 decimal, like the server's `MgDlFrom`). */
export function toMgDl(value: number, unit: GlucoseUnit): number {
  return unit === 'mmol_l' ? round1(value * MMOL_FACTOR) : round1(value);
}

/** A mg/dL value in `unit`, rounded as shown (mg/dL whole, mmol/L 1 decimal). */
export function fromMgDl(mgdl: number, unit: GlucoseUnit): number {
  return unit === 'mmol_l' ? round1(mgdl / MMOL_FACTOR) : Math.round(mgdl);
}

/** The value after switching the unit toggle, so the number on screen keeps meaning the same thing. */
export function convertGlucose(value: number, from: GlucoseUnit, to: GlucoseUnit): number {
  if (from === to) return value;
  return fromMgDl(toMgDl(value, from), to);
}

/** −/+ step of the add form per unit. */
export function glucoseStep(unit: GlucoseUnit): number {
  return unit === 'mmol_l' ? 0.1 : 1;
}

/** Adds `delta` steps and rounds away float noise (0.1 + 0.2). */
export function stepGlucose(value: number, unit: GlucoseUnit, delta: number): number {
  const next = value + delta * glucoseStep(unit);
  return unit === 'mmol_l' ? round1(next) : Math.round(next);
}

/** Decimals a value prints with in `unit`. */
export function glucoseDecimals(unit: GlucoseUnit): number {
  return unit === 'mmol_l' ? 1 : 0;
}

/** ASCII text of a glucose value in `unit` («5.2», «94»); locale digits are the screen's job. */
export function glucoseText(value: number, unit: GlucoseUnit): string {
  return unit === 'mmol_l' ? round1(value).toFixed(1) : String(Math.round(value));
}

/** Unit as printed everywhere (Latin symbol in every locale, like «mmHg»). */
export function unitSymbol(unit: GlucoseUnit): string {
  return unit === 'mmol_l' ? 'mmol/L' : 'mg/dL';
}
