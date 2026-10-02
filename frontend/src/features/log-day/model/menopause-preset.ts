import type { LogCategory, LogDayValues, LogParamValue } from '@/entities/health-log';
import type { IconName, Tone } from '@/shared/ui';

import { asItems, asList, asString, toggleInList, type ItemsValue } from './draft';

/**
 * Menopause preset of the log sheet (CB-MENO-06, nbl_Meno_Log): the board's groups laid over ordinary
 * taxonomy slots. Mirrors `taxonomy.MenopausePreset()` in backend-go (internal/healthlog/taxonomy/
 * menopause.go, docs/canvas-build/menopause.md §3) — `GET /logs/taxonomy` doesn't expose the preset, so
 * the grouping lives here and a slot the taxonomy doesn't return is simply not drawn.
 */

/** The board's 13 symptom rows (item codes). */
export type MenopauseRowItem =
  | 'hot_flashes'
  | 'night_sweats'
  | 'palpitations'
  | 'insomnia'
  | 'anxiety'
  | 'irritability'
  | 'low_mood'
  | 'brain_fog'
  | 'joints'
  | 'fatigue'
  | 'vaginal_dryness'
  | 'bladder_symptoms'
  | 'low_libido';

/** One severity row: an item of an `items` param. */
export interface PresetRow {
  category: string;
  param: string;
  item: MenopauseRowItem;
}

export interface PresetGroup {
  code: 'vasomotor' | 'sleep' | 'mind' | 'body' | 'urogenital';
  icon: IconName;
  tone: Tone;
  rows: readonly PresetRow[];
}

const sym = (item: MenopauseRowItem): PresetRow => ({ category: 'symptoms', param: 'general', item });
const uro = (item: MenopauseRowItem): PresetRow => ({ category: 'urogenital', param: 'symptoms', item });

export const MENOPAUSE_GROUPS: readonly PresetGroup[] = [
  { code: 'vasomotor', icon: 'flame', tone: 'period', rows: [sym('hot_flashes'), sym('night_sweats'), sym('palpitations')] },
  { code: 'sleep', icon: 'moon', tone: 'brand', rows: [sym('insomnia')] },
  { code: 'mind', icon: 'brain', tone: 'bloom', rows: [sym('anxiety'), sym('irritability'), sym('low_mood'), sym('brain_fog')] },
  { code: 'body', icon: 'walk', tone: 'data', rows: [{ category: 'pain', param: 'location', item: 'joints' }, sym('fatigue')] },
  { code: 'urogenital', icon: 'urine', tone: 'period', rows: [uro('vaginal_dryness'), uro('bladder_symptoms'), uro('low_libido')] },
];

/** `bleeding.presence` (none · spotting · bleeding) and `menopause.triggers` (multi). */
export const MENOPAUSE_BLEEDING = { category: 'bleeding', param: 'presence' } as const;
export const MENOPAUSE_TRIGGERS = { category: 'menopause', param: 'triggers' } as const;

/** The four steps of a row; «ندارم» first. */
export const SEVERITY_LEVELS = ['no', 'mild', 'moderate', 'severe'] as const;
export type SeverityLevel = (typeof SEVERITY_LEVELS)[number];

/** The preset applies to the menopause taxonomy (the sheet keeps bloom's layout for every other mode). */
export function isMenopausePreset(mode: string | null): boolean {
  return mode === 'menopause';
}

/** Whether the taxonomy offers the row (its param exists and lists the item). */
export function hasRow(categories: readonly LogCategory[], row: PresetRow): boolean {
  const param = categories.find((c) => c.code === row.category)?.params.find((p) => p.code === row.param);
  return !!param && param.options.some((o) => o.value === row.item);
}

/** Whether the param supports an explicit «ندارم» level (symptoms do; pain locations don't — «ندارم» = no entry). */
export function acceptsNo(categories: readonly LogCategory[], row: PresetRow): boolean {
  const param = categories.find((c) => c.code === row.category)?.params.find((p) => p.code === row.param);
  return !!param && param.levels.some((l) => l.value === 'no');
}

/** The row's level, or `null` when nothing is logged for it. */
export function rowLevel(values: LogDayValues, row: PresetRow): SeverityLevel | null {
  const level = asItems(values[row.category]?.[row.param])[row.item]?.level;
  return (SEVERITY_LEVELS as readonly string[]).includes(level ?? '') ? (level as SeverityLevel) : null;
}

/**
 * The param's new items value after picking `level` on a row. Picking the current level again clears
 * the row; «ندارم» on a param without a `no` level (pain) removes the item.
 */
export function withRowLevel(
  values: LogDayValues,
  row: PresetRow,
  level: SeverityLevel,
  noLevel: boolean,
): ItemsValue {
  const items = { ...asItems(values[row.category]?.[row.param]) };
  const current = items[row.item]?.level ?? null;
  if (current === level || (level === 'no' && !noLevel)) delete items[row.item];
  else items[row.item] = { level, score: null };
  return items;
}

/** The bleeding choice, `null` when unanswered. */
export function bleedingChoice(values: LogDayValues): string | null {
  return asString(values[MENOPAUSE_BLEEDING.category]?.[MENOPAUSE_BLEEDING.param]);
}

/** Bleeding or spotting logged — the post-menopause note offers the alert screen. */
export function bleedingLogged(values: LogDayValues): boolean {
  const choice = bleedingChoice(values);
  return choice !== null && choice !== 'none';
}

/** Re-tapping the selected choice clears it. */
export function nextBleeding(values: LogDayValues, choice: string): LogParamValue | null {
  return bleedingChoice(values) === choice ? null : choice;
}

export function triggerList(values: LogDayValues): string[] {
  return asList(values[MENOPAUSE_TRIGGERS.category]?.[MENOPAUSE_TRIGGERS.param]);
}

export function toggleTrigger(values: LogDayValues, code: string): string[] {
  return toggleInList(triggerList(values), code);
}
